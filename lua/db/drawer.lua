local NuiTree = require("nui.tree")
local NuiLine = require("nui.line")
local helpers = require("db.helpers")

---@class Drawer
---@field private bufnr integer number of buffer to display the tree in
---@field private tree table NuiTree
---@field private connections Connection[]
---@field private ui UI
---@field private last_bufnr integer last used buffer
local Drawer = {}

---@param opts? { connections: Connection[], ui: UI }
function Drawer:new(opts)
  opts = opts or {}

  local connections = opts.connections or {}

  if opts.ui == nil then
    print("no UI provided to drawer")
    return
  end

  -- class object
  local o = {
    tree = nil,
    connections = connections,
    ui = opts.ui,
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

  for _, c in ipairs(self.connections) do
    local db = NuiTree.Node { id = c.meta.name, connection = c, text = c.meta.name, type = "db" }
    tree:add_node(db)
  end

  return tree
end

---@param connection Connection
function Drawer:add_connection(connection)
  local db =
    NuiTree.Node { id = connection.meta.name, connection = connection, text = connection.meta.name, type = "db" }

  local existing_nodes = self.tree:get_nodes()
  for _, n in ipairs(existing_nodes) do
    if n.text == connection.meta.name then
      return
    end
  end

  self.tree:add_node(db)
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

---@private
---@return table node current master node
function Drawer:current_master()
  local node = self.tree:get_node()

  local function process(n)
    if not n:get_parent_id() then
      return n
    end
    local parent = self.tree:get_node(n:get_parent_id())
    return process(parent)
  end

  return process(node)
end

-- Map keybindings to split window
---@private
---@param bufnr integer which buffer to map the keys in
function Drawer:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }

  -- quit
  vim.keymap.set("n", "q", function()
    vim.api.nvim_win_close(0, false)
  end, { noremap = true, buffer = bufnr })

  -- confirm
  vim.keymap.set("n", "<CR>", function()
    local node = self.tree:get_node()
    if type(node.action) == "function" then
      node.action()
    end
  end, map_options)

  -- collapse current node
  vim.keymap.set("n", "i", function()
    local node = self.tree:get_node()
    if not node then
      return
    end

    if node:collapse() then
      self.tree:render()
    end
  end, map_options)

  -- expand all children nodes with only one field
  local function expand_all_single(node)
    local children = node:get_child_ids()
    if #children == 1 then
      local nested_node = self.tree:get_node(children[1])
      nested_node:expand()
      expand_all_single(nested_node)
    end
  end

  -- expand current node
  vim.keymap.set("n", "o", function()
    local node = self.tree:get_node()
    if not node then
      return
    end
    -- TODO: clean this up
    local expanded = node:expand()

    expand_all_single(node)

    self:refresh()
    expanded = node:expand()

    if expanded then
      self.tree:render()
    end
  end, map_options)
end

-- Refresh parent connection tree
---@private
---@param node? table currently selected node
function Drawer:refresh(node)
  node = node or self:current_master()
  local connection = node.connection
  local expanded = self:get_expanded_ids() or {}

  -- schemas
  local schemas = connection:schemas()
  -- table helpers
  local table_helpers = helpers.get(connection.type)
  if not table_helpers then
    print("no table_helpers")
    return
  end
  -- history
  local history = connection.history

  -- structure
  local schema_nodes = {}
  for sch_name, tbls in pairs(schemas) do
    local sch_id = connection.meta.name .. sch_name
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
          text = helper_name,
          type = "query",
          action = function()
            local cb = function()
              self:refresh(node)
            end
            connection:execute(
              helpers.expand_query(helper_query, { table = tbl_name, schema = sch_name, dbname = connection.meta.name }),
              "preview",
              cb
            )
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
  for i, _ in ipairs(history) do
    local history_node = NuiTree.Node {
      id = "history" .. tostring(i),
      text = tostring(i),
      type = "history",
      action = function()
        connection:display_history(i)
      end,
    }
    if expanded[history_node.id] then
      history_node:expand()
    end
    table.insert(history_nodes, history_node)
  end

  local children = {
    NuiTree.Node { id = connection.meta.name .. "new_query", text = "new query" },
    NuiTree.Node({ id = connection.meta.name .. "structure", text = "structure" }, schema_nodes),
    NuiTree.Node({ id = connection.meta.name .. "history", text = "history" }, history_nodes),
  }
  -- expand nodes from map
  for _, n in ipairs(children) do
    if expanded[n.id] then
      n:expand()
    end
  end

  self.tree:set_nodes(children, node:get_id())
  self.tree:render()
end

-- Show drawer on screen
function Drawer:render()
  local bufnr = self.ui:open()

  if not self.tree then
    self.tree = self:create_tree(bufnr)
  end

  if bufnr ~= self.last_bufnr then
    self:map_keys(bufnr)
    self.tree.bufnr = bufnr
    self.last_bufnr = bufnr
  end

  self.tree:render()
end

return Drawer
