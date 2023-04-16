local NuiTree = require("nui.tree")
local NuiLine = require("nui.line")
local helpers = require("dbee.helpers")

---@class Node
---@field id string
---@field text string
---@field is_expanded fun(self:Node):boolean
---@field is_master boolean

---@class MasterNode: Node
---@field getter fun():layout

---@class Drawer
---@field private tree table NuiTree
---@field private handler Handler
---@field private editor Editor
---@field private bufnr integer
---@field private winid integer
---@field private win_cmd fun():integer function which opens a new window and returns a window id
local Drawer = {}

local SCRATCHPAD_NODE_ID = "scratchpad_node"

---@param opts? { handler: Handler, editor: Editor, win_cmd: string | fun():integer }
---@return Drawer
function Drawer:new(opts)
  opts = opts or {}

  if opts.handler == nil then
    error("no Handler provided to drawer")
  end

  if opts.editor == nil then
    error("no Editor provided to drawer")
  end

  local win_cmd
  if type(opts.win_cmd) == "string" then
    win_cmd = function()
      vim.cmd(opts.win_cmd)
      return vim.api.nvim_get_current_win()
    end
  elseif type(opts.win_cmd) == "function" then
    win_cmd = opts.win_cmd
  else
    win_cmd = function()
      vim.cmd("to 40vsplit")
      return vim.api.nvim_get_current_win()
    end
  end

  -- class object
  local o = {
    tree = nil,
    handler = opts.handler,
    editor = opts.editor,
    win_cmd = win_cmd,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@private
---@return table tree
function Drawer:create_tree(bufnr)
  return NuiTree {
    bufnr = bufnr, -- dummy to suppress error
    prepare_node = function(node)
      local line = NuiLine()

      line:append(string.rep("  ", node:get_depth() - 1))

      if node:has_children() or not node:get_parent_id() then
        line:append(node:is_expanded() and " " or " ", "SpecialChar")
      else
        line:append("  ")
      end

      -- if connection is the active one, apply a special highlihgt on the master
      if node.is_master and tostring(self.handler:connection_details().id) == node.id then
        line:append(node.text, "Title")
      else
        line:append(node.text)
      end

      return line
    end,
    get_node_id = function(node)
      if node.id then
        return node.id
      end
      return math.random()
    end,
  }
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

    if node.is_master then
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
---@param master_node_id string master node id
function Drawer:refresh_node(master_node_id)
  ---@type MasterNode
  local master_node = self.tree:get_node(master_node_id)

  local layout = master_node.getter()

  ---@param _layout layout[]
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
            local connection_id = tonumber(master_node_id)
            if not connection_id then
              error("master_node_id is not a valid number")
            end
            local details = self.handler:connection_details(connection_id)
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
                  connection_id
                )
              end
            end)
            self.handler:set_active(details.id)
          elseif _l.type == "history" then
            local connection_id = tonumber(master_node_id)
            if not connection_id then
              error("master_node_id is not a valid number")
            end
            -- TODO: make propper history ids
            self.handler:history(_l.name, connection_id)
            self.handler:set_active(connection_id)
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

      -- get existing node from the current tree and check if it is expanded
      local _ex_node = self.tree:get_node(_id)
      if _ex_node and _ex_node:is_expanded() then
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
  ---@type MasterNode[]
  local existing_nodes = self.tree:get_nodes()

  ---@param _id string
  local function _exists(_id)
    for _, _n in ipairs(existing_nodes) do
      if _n.id == _id then
        return true
      end
    end
    return false
  end

  -- connections
  local cons = self.handler:list_connections()
  for _, con in ipairs(cons) do
    if not _exists(tostring(con.id)) then
      ---@type MasterNode
      local node = NuiTree.Node {
        id = tostring(con.id),
        text = con.name,
        is_master = true,
        getter = function()
          return self.handler:layout(con.id)
        end,
      }
      self.tree:add_node(node)
    end
  end

  -- scratchpads
  if not _exists(SCRATCHPAD_NODE_ID) then
    ---@type MasterNode
    local node = NuiTree.Node {
      id = SCRATCHPAD_NODE_ID,
      text = "scratchpads",
      is_master = true,
      getter = function()
        return self.editor:layout()
      end,
    }
    self.tree:add_node(node)
  end

  -- refresh open master nodes
  for _, n in ipairs(existing_nodes) do
    if n:is_expanded() then
      self:refresh_node(n.id)
    end
  end
end

-- Show drawer on screen
---@param winid? integer
function Drawer:open(winid)
  winid = winid or self.winid
  if not winid or not vim.api.nvim_win_is_valid(winid) then
    winid = self.win_cmd()
  end

  -- if buffer doesn't exist, create it
  local bufnr = self.bufnr
  if not bufnr or not vim.api.nvim_buf_is_valid(bufnr) then
    bufnr = vim.api.nvim_create_buf(false, true)
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
    self:refresh()
  end

  self:map_keys(bufnr)
  self.tree.bufnr = bufnr

  self.bufnr = bufnr
  self.winid = winid

  self.tree:render()
end

function Drawer:close()
  vim.api.nvim_win_close(self.winid, false)
end

return Drawer
