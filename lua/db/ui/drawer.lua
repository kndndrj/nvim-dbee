local NuiTree = require("nui.tree")
local NuiSplit = require("nui.split")
local NuiLine = require("nui.line")

---@param unexpanded_query string
---@param vars { table: string, schema: string, dbname: string }
---@return string query with expanded vars
local function expand_query(unexpanded_query, vars)
  local ret = unexpanded_query
  for key, val in pairs(vars) do
    ret = ret:gsub("{" .. key .. "}", val)
  end
  return ret
end

---@class Drawer
---@field tree table NuiTree
---@field split table NuiSplit
---@field connections Connection[]
---@field on_result fun(result: string|string[], type: "file"|"lines") callback to call when results are ready
local Drawer = {}

---@param opts? { connections: Connection[], on_result: fun(result: string|string[], type: "file"|"lines") }
function Drawer:new(opts)
  opts = opts or {}

  local split = NuiSplit {
    relative = "win",
    position = "left",
    size = 40,
  }

  local event = require("nui.utils.autocmd").event
  split:on({ event.BufWinLeave }, function()
    vim.schedule(function()
      split:unmount()
    end)
  end, { once = true })

  local tree = NuiTree {
    bufnr = split.bufnr,
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

  if opts.connections then
    for _, c in ipairs(opts.connections) do
      local db = NuiTree.Node { id = c.meta.name, connection = c, text = c.meta.name, type = "db" }
      tree:add_node(db)
    end
  end

  -- class object
  local o = {
    tree = tree,
    split = split,
    connections = opts.connections,
    on_result = opts.on_result or function() end,
  }
  setmetatable(o, self)
  self.__index = self
  return o
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
function Drawer:map_keys()
  local map_options = { noremap = true, nowait = true }

  -- quit
  self.split:map("n", "q", function()
    self.split:unmount()
  end, { noremap = true })

  -- confirm
  self.split:map("n", "<CR>", function()
    local node = self.tree:get_node()
    if type(node.action) == "function" then
      node.action()
    end
  end, map_options)

  -- collapse current node
  self.split:map("n", "i", function()
    local node = self.tree:get_node()
    if not node then
      return
    end

    if node:collapse() then
      self.tree:render()
    end
  end, map_options)

  -- expand current node
  self.split:map("n", "o", function()
    local node = self.tree:get_node()
    if not node then
      return
    end
    -- TODO: clean this up
    local expanded = node:expand()

    self:refresh()
    expanded = node:expand()

    if expanded then
      self.tree:render()
    end
  end, map_options)
end

-- Refresh parent connection tree
---@param node? table currently selected node
function Drawer:refresh(node)
  node = node or self:current_master()
  local connection = node.connection
  local expanded = self:get_expanded_ids() or {}

  -- schemas
  local schemas = connection:schemas()
  -- table helpers
  local table_helpers = connection:table_helpers()
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
            local cb = function(data)
              -- local ui_results = require("db.ui.results")
              -- ui_results.show(data)
              self.on_result(data, "lines")
              self:refresh(node)
            end
            connection:execute_to_result(
              expand_query(helper_query, { table = tbl_name, schema = sch_name, dbname = connection.meta.name }),
              cb,
              "preview"
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
  for i, h in ipairs(history) do
    local history_node = NuiTree.Node {
      id = "history" .. tostring(i),
      text = tostring(i),
      type = "history",
      action = function()
        -- local ui_results = require("db.ui.results")
        -- ui_results.show_file(h.file)
        self.on_result(h.file, "file")
      end,
      -- TODO remove?
      query = h.query,
      file = h.file,
    }
    if expanded[history_node.id] then
      history_node:expand()
    end
    table.insert(history_nodes, history_node)
  end

  local children = {
    NuiTree.Node { id = "new_query", text = "new query" },
    NuiTree.Node({ id = "structure", text = "structure" }, schema_nodes),
    NuiTree.Node({ id = "history", text = "history" }, history_nodes),
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
function Drawer:show()
  self.split:mount()

  self:map_keys()
  self.tree.bufnr = self.split.bufnr

  self.tree:render()
end

-- Hide drawer off screen
function Drawer:hide()
  self.split:unmount()
end

return Drawer
