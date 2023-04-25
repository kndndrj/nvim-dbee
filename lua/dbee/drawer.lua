local NuiTree = require("nui.tree")
local NuiLine = require("nui.line")

local SCRATCHPAD_NODE_ID = "scratchpad_node"

---@class Icon
---@field icon string
---@field highlight string

---@class Layout
---@field name string display name
---@field schema? string parent schema
---@field database? string parent database
---@field type ""|"table"|"history"|"scratch"|"database" type of layout
---@field action_1? fun(cb: fun()) primary action - takes single arg: callback closure
---@field action_2? fun(cb: fun()) secondary action - takes single arg: callback closure
---@field action_3? fun(cb: fun()) tertiary action - takes single arg: callback closure
---@field children? Layout[] child layout nodes

---@class Node
---@field id string
---@field text string
---@field type string type which infers icon
---@field is_expanded fun(self:Node):boolean
---@field is_master boolean
---@field action_1 fun() function to perform on primary event
---@field action_2 fun() function to perform on secondary event
---@field action_3 fun() function to perform on tertiary event

---@class MasterNode: Node
---@field getter fun():Layout

---@alias drawer_config { disable_icons: boolean, icons: table<string, Icon>, mappings: table<string, string>, window_command: string|fun():integer }

---@class Drawer
---@field private tree table NuiTree
---@field private handler Handler
---@field private editor Editor
---@field private mappings table<string, string>
---@field private bufnr integer
---@field private winid integer
---@field private icons table<string, Icon>
---@field private win_cmd fun():integer function which opens a new window and returns a window id
local Drawer = {}

---@param handler Handler
---@param editor Editor
---@param opts? drawer_config
---@return Drawer
function Drawer:new(handler, editor, opts)
  opts = opts or {}

  if not handler then
    error("no Handler provided to drawer")
  end
  if not editor then
    error("no Editor provided to drawer")
  end

  local win_cmd
  if type(opts.window_command) == "string" then
    win_cmd = function()
      vim.cmd(opts.window_command)
      return vim.api.nvim_get_current_win()
    end
  elseif type(opts.window_command) == "function" then
    win_cmd = opts.window_command
  else
    win_cmd = function()
      vim.cmd("to 40vsplit")
      return vim.api.nvim_get_current_win()
    end
  end

  local icons = {}
  if not opts.disable_icons then
    icons = opts.icons or {}
  end

  -- class object
  local o = {
    tree = nil,
    handler = handler,
    editor = editor,
    mappings = opts.mappings or {},
    icons = icons,
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
    bufnr = bufnr,
    prepare_node = function(node)
      local line = NuiLine()

      line:append(string.rep("  ", node:get_depth() - 1))

      if node:has_children() or not node:get_parent_id() then
        line:append(node:is_expanded() and " " or " ", "NonText")
      else
        line:append("  ")
      end

      ---@type Icon
      local icon
      -- special icons for nodes without type
      if not node.type or node.type == "" then
        if node:has_children() then
          icon = self.icons["none_dir"]
        else
          icon = self.icons["none"]
        end
      else
        icon = self.icons[node.type] or {}
      end
      icon = icon or {}

      if icon.icon then
        line:append(" " .. icon.icon .. " ", icon.highlight)
      end

      -- if connection is the active one, apply a special highlight on the master
      if node.is_master and self.handler:connection_details().id == node.id then
        line:append(node.text, icon.highlight)
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

  -- manual refresh
  local key = self.mappings["refresh"]
  if key then
    vim.keymap.set("n", key, function()
      self:refresh()
    end, map_options)
  end

  -- action_1 (confirm)
  key = self.mappings["action_1"]
  if key then
    vim.keymap.set("n", key, function()
      local node = self.tree:get_node()
      if type(node.action_1) == "function" then
        node.action_1()
      end
    end, map_options)
  end

  -- action_2 (alter)
  key = self.mappings["action_2"]
  if key then
    vim.keymap.set("n", key, function()
      local node = self.tree:get_node()
      if type(node.action_2) == "function" then
        node.action_2()
      end
    end, map_options)
  end

  -- action_3 (remove)
  key = self.mappings["action_3"]
  if key then
    vim.keymap.set("n", key, function()
      local node = self.tree:get_node()
      if type(node.action_3) == "function" then
        node.action_3()
      end
    end, map_options)
  end

  local function collapse_node(node)
    if node:collapse() then
      self.tree:render()
    end
  end

  local function expand_node(node)
    -- expand all children nodes with only one field
    local function expand_all_single(n)
      local children = n:get_child_ids()
      if #children == 1 then
        local nested_node = self.tree:get_node(children[1])
        nested_node:expand()
        expand_all_single(nested_node)
      end
    end

    local expanded = node:is_expanded()

    expand_all_single(node)

    if node.is_master then
      self:refresh_node(node.id)
    end

    node:expand()

    if expanded ~= node:is_expanded() then
      self.tree:render()
    end
  end

  -- collapse current node
  key = self.mappings["collapse"]
  if key then
    vim.keymap.set("n", key, function()
      local node = self.tree:get_node()
      if not node then
        return
      end
      collapse_node(node)
    end, map_options)
  end

  -- expand current node
  key = self.mappings["expand"]
  if key then
    vim.keymap.set("n", key, function()
      local node = self.tree:get_node()
      if not node then
        return
      end
      expand_node(node)
    end, map_options)
  end

  -- toggle collapse/expand
  key = self.mappings["toggle"]
  if key then
    vim.keymap.set("n", key, function()
      local node = self.tree:get_node()
      if not node then
        return
      end
      if node:is_expanded() then
        collapse_node(node)
      else
        expand_node(node)
      end
    end, map_options)
  end
end

---@private
---@param master_node_id string master node id
function Drawer:refresh_node(master_node_id)
  ---@param layouts Layout[]
  ---@param parent_id? string
  ---@return table nodes list of NuiTreeNodes
  local function layout_to_tree_nodes(layouts, parent_id)
    parent_id = parent_id or ""

    if not layouts then
      return {}
    end

    -- sort keys
    table.sort(layouts, function(k1, k2)
      return k1.name < k2.name
    end)

    local nodes = {}
    for _, l in ipairs(layouts) do
      local id = parent_id .. l.name
      local node = NuiTree.Node({
        id = id,
        master_id = master_node_id,
        text = string.gsub(l.name, "\n", " "),
        type = l.type,
        action_1 = function()
          if type(l.action_1) == "function" then
            l.action_1(function()
              self:refresh()
            end)
          end
        end,
        action_2 = function()
          if type(l.action_2) == "function" then
            l.action_2(function()
              self:refresh()
            end)
          end
        end,
        action_3 = function()
          if type(l.action_3) == "function" then
            l.action_3(function()
              self:refresh()
            end)
          end
        end,
        -- recurse children
      }, layout_to_tree_nodes(l.children, id))

      -- get existing node from the current tree and check if it is expanded
      local ex_node = self.tree:get_node(id)
      if ex_node and ex_node:is_expanded() then
        node:expand()
      end

      table.insert(nodes, node)
    end

    return nodes
  end

  ---@type MasterNode
  local master_node = self.tree:get_node(master_node_id)

  local layout = master_node.getter()

  local children = layout_to_tree_nodes(layout, tostring(master_node_id))

  self.tree:set_nodes(children, master_node_id)
  self.tree:render()
end

function Drawer:refresh()
  ---@type MasterNode[]
  local existing_nodes = self.tree:get_nodes()

  ---@param id string
  local function exists(id)
    for _, n in ipairs(existing_nodes) do
      if n.id == id then
        return true
      end
    end
    return false
  end

  -- scratchpads
  if not exists(SCRATCHPAD_NODE_ID) then
    ---@type MasterNode
    local node = NuiTree.Node {
      id = SCRATCHPAD_NODE_ID,
      text = "scratchpads",
      type = "scratch",
      is_master = true,
      getter = function()
        return self.editor:layout()
      end,
    }
    self.tree:add_node(node)
  end

  -- connections
  local cons = self.handler:list_connections()
  for _, con in ipairs(cons) do
    if not exists(con.id) then
      ---@type MasterNode
      local node = NuiTree.Node {
        id = con.id,
        text = con.name,
        type = "database",
        is_master = true,
        -- set connection as active manually
        action_2 = function()
          self.handler:set_active(con.id)
          self:refresh_node(con.id)
        end,
        getter = function()
          return self.handler:layout(con.id)
        end,
      }
      self.tree:add_node(node)
    end
  end

  -- refresh open master nodes
  for _, n in ipairs(existing_nodes) do
    if n:is_expanded() then
      self:refresh_node(n.id)
    end
  end
end

-- Show drawer on screen
function Drawer:open()
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    self.winid = self.win_cmd()
  end

  -- if buffer doesn't exist, create it
  local bufnr = self.bufnr
  if not bufnr or not vim.api.nvim_buf_is_valid(bufnr) then
    bufnr = vim.api.nvim_create_buf(false, true)
  end

  vim.api.nvim_win_set_buf(self.winid, bufnr)
  vim.api.nvim_set_current_win(self.winid)
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
    vim.api.nvim_win_set_option(self.winid, opt, val)
  end

  -- tree
  if not self.tree then
    self.tree = self:create_tree(bufnr)
    self:refresh()
  end

  self:map_keys(bufnr)
  self.tree.bufnr = bufnr

  self.bufnr = bufnr

  self.tree:render()
end

function Drawer:close()
  pcall(vim.api.nvim_win_close, self.winid, false)
end

return Drawer
