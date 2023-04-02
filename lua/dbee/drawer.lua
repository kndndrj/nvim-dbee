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

    if node.type == "db" then
      self:refresh_connection(node.id)
    elseif node.type == "scratch" then
      self:refresh_scratches()
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
---@param node_id integer master node id
function Drawer:refresh_connection(node_id)
  local master_node = self.tree:get_node(node_id)

  local con_details = self.handler:connection_details(master_node.connection_id)
  local expanded = self:get_expanded_ids() or {}

  -- schemas
  local schemas = self.handler:schemas(con_details.id)
  -- table helpers
  local table_helpers = helpers.get(con_details.type)
  if not table_helpers then
    print("no table_helpers")
    return
  end
  -- history
  local history = self.handler:list_history(con_details.id)

  -- structure
  local schema_nodes = {}
  for sch_name, tbls in pairs(schemas) do
    local sch_id = con_details.name .. sch_name
    -- tables
    local tbl_nodes = {}
    for _, tbl_name in ipairs(tbls) do
      local tbl_id = sch_id .. tbl_name
      -- helpers
      local helper_nodes = {}
      for helper_name, helper_query in pairs(table_helpers) do
        local helper_id = tbl_id .. helper_name
        local h = NuiTree.Node {
          id = helper_id,
          master_id = node_id,
          text = helper_name,
          type = "query",
          action = function()
            self.handler:set_active(con_details.id)

            self.handler:execute(
              helpers.expand_query(helper_query, { table = tbl_name, schema = sch_name, dbname = con_details.name }),
              con_details.id
            )

            self:refresh_connection(node_id)
          end,
        }
        if expanded[h.id] then
          h:expand()
        end
        table.insert(helper_nodes, h)
      end
      local t = NuiTree.Node({ id = tbl_id, text = tbl_name, type = "table" }, helper_nodes)
      if expanded[t.id] then
        t:expand()
      end
      table.insert(tbl_nodes, t)
    end

    local schema_node = NuiTree.Node({ id = sch_id, text = sch_name, type = "schema" }, tbl_nodes)
    if expanded[schema_node.id] then
      schema_node:expand()
    end
    table.insert(schema_nodes, schema_node)
  end

  -- history
  local history_nodes = {}
  for _, h in ipairs(history) do
    local history_node = NuiTree.Node {
      id = "history" .. con_details.name .. h,
      master_id = node_id,
      text = h,
      type = "history",
      action = function()
        self.handler:set_active(con_details.id)
        self.handler:history(h, con_details.id)
        self:refresh_connection(node_id)
      end,
    }
    if expanded[history_node.id] then
      history_node:expand()
    end
    table.insert(history_nodes, history_node)
  end

  local children = {
    NuiTree.Node({ id = con_details.name .. "structure", master_id = node_id, text = "structure" }, schema_nodes),
    NuiTree.Node({ id = con_details.name .. "history", master_id = node_id, text = "history" }, history_nodes),
  }
  -- expand nodes from map
  for _, n in ipairs(children) do
    if expanded[n.id] then
      n:expand()
    end
  end

  self.tree:set_nodes(children, node_id)
  self.tree:render()
end

function Drawer:refresh_scratches()
  -- new scratchpad button
  local scratch_nodes = {
    NuiTree.Node {
      id = "scratch:new:-",
      master_id = SCRATCHPAD_NODE_ID,
      text = "New",
      type = "scratch",
      action = function()
        self.editor:new_scratch()
        self.editor:open()
        self:refresh_scratches()
      end,
    },
  }

  -- scratches
  for id, s in ipairs(self.editor:list_scratches()) do
    local scratch_node = NuiTree.Node {
      id = "scratch:" .. id .. ":" .. s,
      master_id = SCRATCHPAD_NODE_ID,
      text = vim.fs.basename(s),
      type = "scratch",
      action = function()
        self.editor:set_active_scratch(id)
        self.editor:open()
      end,
    }
    table.insert(scratch_nodes, scratch_node)
  end
  self.tree:set_nodes(scratch_nodes, SCRATCHPAD_NODE_ID)
  self.tree:render()
end

function Drawer:refresh()
  -- scratches
  self:refresh_scratches()

  -- connections
  local existing_nodes = self.tree:get_nodes()

  local function exists(connection_id)
    for _, n in ipairs(existing_nodes) do
      if n.connection_id == connection_id then
        return true
      end
    end
    return false
  end

  local cons = self.handler:list_connections()

  for _, con in ipairs(cons) do
    -- add connection if it doesn't exist, refresh it if it does
    if not exists(con.id) then
      local db = NuiTree.Node { id = con.name, connection_id = con.id, text = con.name, type = "db" }
      self.tree:add_node(db)
    else
      for _, n in ipairs(existing_nodes) do
        if n.connection_id == con.id then
          self:refresh_connection(n.id)
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

  if bufnr ~= self.ui_opts.bufnr then
    self:map_keys(bufnr)
    self.tree.bufnr = bufnr
    self.ui_opts.bufnr = bufnr
  end

  self.ui_opts.bufnr = bufnr
  self.ui_opts.winid = winid

  self.tree:render()
end

function Drawer:close()
  vim.api.nvim_win_close(self.ui_opts.winid, false)
end

return Drawer
