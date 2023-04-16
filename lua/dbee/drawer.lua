local NuiTree = require("nui.tree")
local NuiLine = require("nui.line")
local helpers = require("dbee.helpers")

---@class Drawer
---@field private tree table NuiTree
---@field private handler Handler
---@field private editor Editor
---@field private ui_opts { win_cmd: string, bufnr: integer, winid: integer}
local Drawer = {}

local SCRATCHPAD_NODE_ID = "scratchpad_node"

---@param opts? { handler: Handler, editor: Editor, win_cmd: string }
---@return Drawer|nil
function Drawer:new(opts)
  opts = opts or {}

  if opts.handler == nil then
    print("no Handler provided to drawer")
    return
  end

  if opts.editor == nil then
    print("no Editor provided to drawer")
    return
  end

  -- class object
  local o = {
    tree = nil,
    handler = opts.handler,
    editor = opts.editor,
    ui_opts = {
      win_cmd = opts.win_cmd or "to 40vsplit",
      bufnr = nil,
      winid = nil,
    },
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@private
---@return table tree
function Drawer:create_tree(bufnr)
  local tree = NuiTree {
    bufnr = bufnr, -- dummy to suppress error
    prepare_node = function(node)
      local line = NuiLine()

      line:append(string.rep("  ", node:get_depth() - 1))

      if node:has_children() or not node:get_parent_id() then
        line:append(node:is_expanded() and " " or " ", "SpecialChar")
      else
        line:append("  ")
      end

      line:append(node.text)

      return line
    end,
    get_node_id = function(node)
      if node.id then
        return node.id
      end
      return math.random()
    end,
  }

  -- add scratchpad node
  local scratch_node = NuiTree.Node { id = SCRATCHPAD_NODE_ID, text = "scratchpads", type = "scratch" }
  tree:add_node(scratch_node)

  -- add connections
  local cons = self.handler:list_connections()
  for _, c in ipairs(cons) do
    local db = NuiTree.Node { id = c.name, connection_id = c.id, text = c.name, type = "db" }
    tree:add_node(db)
  end

  return tree
end

---@private
---@return { string: boolean } expanded map of node ids that are expanded
function Drawer:get_expanded_ids()
  local expanded = {}

  local function process(node)
    if node:is_expanded() then
      expanded[node:get_id()] = true

      if node:has_children() then
        for _, n in ipairs(self.tree:get_nodes(node:get_id())) do
          process(n)
        end
      end
    end
  end

  for _, node in ipairs(self.tree:get_nodes()) do
    process(node)
  end

  return expanded
end

-- Map keybindings to split window
---@private
---@param bufnr integer which buffer to map the keys in
function Drawer:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }

  -- quit
  vim.keymap.set("n", "q", function()
    vim.api.nvim_win_close(0, false)
  end, map_options)

  -- manual refresh
  vim.keymap.set("n", "r", function()
    self:refresh()
  end, map_options)

  -- confirm
  vim.keymap.set("n", "<CR>", function()
    local node = self.tree:get_node()
    if type(node.action) == "function" then
      node.action()
    end
  end, map_options)

  local function _collapse_node(node)
    if node:collapse() then
      self.tree:render()
    end
  end

  local function _expand_node(node)
    -- expand all children nodes with only one field
    local function __expand_all_single(n)
      local children = n:get_child_ids()
      if #children == 1 then
        local nested_node = self.tree:get_node(children[1])
        nested_node:expand()
        __expand_all_single(nested_node)
      end
    end

    local expanded = node:is_expanded()

    __expand_all_single(node)

    if node.type == "db" or node.type == "scratch" then
      self:refresh_node(node.id)
    end

    node:expand()

    if expanded ~= node:is_expanded() then
      self.tree:render()
    end
  end

  -- collapse current node
  vim.keymap.set("n", "c", function()
    local node = self.tree:get_node()
    if not node then
      return
    end
    _collapse_node(node)
  end, map_options)

  -- expand current node
  vim.keymap.set("n", "e", function()
    local node = self.tree:get_node()
    if not node then
      return
    end
    _expand_node(node)
  end, map_options)

  -- toggle collapse/expand
  vim.keymap.set("n", "o", function()
    local node = self.tree:get_node()
    if not node then
      return
    end
    if node:is_expanded() then
      _collapse_node(node)
    else
      _expand_node(node)
    end
  end, map_options)
end

---@private
---@param master_node_id integer master node id
function Drawer:refresh_node(master_node_id)
  local master_node = self.tree:get_node(master_node_id)

  local layout = {}
  if master_node.type == "db" then
    local details = self.handler:connection_details(master_node.connection_id)
    layout = self.handler:layout(details.id)
  elseif master_node.type == "scratch" then
    layout = self.editor:layout()
  end

  local expanded = self:get_expanded_ids() or {}

  ---@param _layout schema[]
  ---@param _parent_id? string
  ---@return table nodes list of NuiTreeNodes
  local function _layout_to_tree_nodes(_layout, _parent_id)
    _parent_id = _parent_id or ""

    if not _layout or _layout == vim.NIL then
      return {}
    end

    -- sort keys
    table.sort(_layout, function(k1, k2)
      return k1.name < k2.name
    end)

    local _nodes = {}
    for _, _l in ipairs(_layout) do
      local _id = _parent_id .. _l.name
      local _node = NuiTree.Node({
        id = _id,
        master_id = master_node_id,
        text = _l.name,
        action = function()
          -- get action from type
          if _l.type == "table" then
            local details = self.handler:connection_details(master_node.connection_id)
            local table_helpers = helpers.get(details.type)
            local helper_keys = {}
            for k, _ in pairs(table_helpers) do
              table.insert(helper_keys, k)
            end
            -- select a helper to execute
            vim.ui.select(helper_keys, {
              prompt = "select a helper to execute:",
            }, function(_selection)
              if _selection then
                self.handler:execute(
                  helpers.expand_query(
                    table_helpers[_selection],
                    { table = _l.name, schema = _l.schema, dbname = _l.database }
                  ),
                  details.id
                )
              end
            end)
            self.handler:set_active(details.id)
          elseif _l.type == "history" then
            local details = self.handler:connection_details(master_node.connection_id)
            -- TODO: make propper history ids
            self.handler:history(_l.name, details.id)
            self.handler:set_active(details.id)
          elseif _l.type == "record" then
            self.tree:get_node(_id):expand()
          elseif _l.type == "scratch" then
            self.editor:set_active_scratch(_l.name)
            self.editor:open()
          end

          self:refresh_node(master_node_id)
        end,
        -- recurse children
      }, _layout_to_tree_nodes(_l.children, _id))

      -- expand if it was expanded
      if expanded[_id] then
        _node:expand()
      end

      table.insert(_nodes, _node)
    end

    return _nodes
  end

  local children = _layout_to_tree_nodes(layout, tostring(master_node_id))

  self.tree:set_nodes(children, master_node_id)
  self.tree:render()
end

function Drawer:refresh()
  local existing_nodes = self.tree:get_nodes()

  local function _exists(_id)
    for _, _n in ipairs(existing_nodes) do
      if _n.connection_id == _id then
        return true
      end
    end
    return false
  end

  local cons = self.handler:list_connections()

  for _, con in ipairs(cons) do
    -- add connection if it doesn't exist, refresh it if it does
    if not _exists(con.id) then
      local db = NuiTree.Node { id = con.name, connection_id = con.id, text = con.name, type = "db" }
      self.tree:add_node(db)
    else
      for _, n in ipairs(existing_nodes) do
        if n.connection_id == con.id and n:is_expanded() then
          self:refresh_node(n.id)
        end
      end
    end
  end
end

-- Show drawer on screen
---@param winid? integer if provided, use it instead of creating new window
function Drawer:open(winid)
  -- if buffer doesn't exist, create it
  local bufnr = self.ui_opts.bufnr
  if not bufnr or not vim.api.nvim_buf_is_valid(bufnr) then
    bufnr = vim.api.nvim_create_buf(false, true)
  end

  -- if window doesn't exist, create it
  winid = winid or self.ui_opts.winid
  if not winid or not vim.api.nvim_win_is_valid(winid) then
    vim.cmd(self.ui_opts.win_cmd)
    winid = vim.api.nvim_get_current_win()
  end

  vim.api.nvim_win_set_buf(winid, bufnr)
  vim.api.nvim_set_current_win(winid)
  vim.api.nvim_buf_set_name(bufnr, "dbee-drawer")

  -- set options
  local buf_opts = {
    buflisted = false,
    bufhidden = "delete",
    buftype = "nofile",
    swapfile = false,
  }
  local win_opts = {
    wrap = false,
    winfixheight = true,
    winfixwidth = true,
    number = false,
  }
  for opt, val in pairs(buf_opts) do
    vim.api.nvim_buf_set_option(bufnr, opt, val)
  end
  for opt, val in pairs(win_opts) do
    vim.api.nvim_win_set_option(winid, opt, val)
  end

  -- tree
  if not self.tree then
    self.tree = self:create_tree(bufnr)
  end

  self:map_keys(bufnr)
  self.tree.bufnr = bufnr

  self.ui_opts.bufnr = bufnr
  self.ui_opts.winid = winid

  self.tree:render()
end

function Drawer:close()
  vim.api.nvim_win_close(self.ui_opts.winid, false)
end

return Drawer
