local NuiTree = require("nui.tree")
local NuiLine = require("nui.line")

---@class Icon
---@field icon string
---@field highlight string

---@class Layout
---@field id string unique identifier
---@field name string display name
---@field type ""|"table"|"history"|"scratch"|"database" type of layout
---@field schema? string parent schema
---@field database? string parent database
---@field action_1? fun(cb: fun()) primary action - takes single arg: callback closure
---@field action_2? fun(cb: fun()) secondary action - takes single arg: callback closure
---@field action_3? fun(cb: fun()) tertiary action - takes single arg: callback closure
---@field children? Layout[]|fun():Layout[] child layout nodes

-- node is Layout converted to NuiTreeNode
---@class Node: Layout
---@field getter fun():Layout

---@alias drawer_config { disable_icons: boolean, icons: table<string, Icon>, mappings: table<string, mapping>, window_command: string|fun():integer }

---@class Drawer
---@field private tree table NuiTree
---@field private handler Handler
---@field private editor Editor
---@field private mappings table<string, mapping>
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
        local icon = self.icons["node_closed"] or { icon = ">", highlight = "NonText" }
        if node:is_expanded() then
          icon = self.icons["node_expanded"] or { icon = "v", highlight = "NonText" }
        end
        line:append(icon.icon .. " ", icon.highlight)
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
      local active = self.handler:connection_details()
      if active and active.id == node.id then
        line:append(node.name, icon.highlight)
      else
        line:append(node.name)
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

---@return table<string, fun()>
function Drawer:actions()
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

    -- if function for getting layout exist, call it
    if not expanded and type(node.getter) == "function" then
      node.getter()
    end

    node:expand()

    if expanded ~= node:is_expanded() then
      self.tree:render()
    end
  end

  return {
    refresh = function()
      self:refresh()
    end,
    action_1 = function()
      local node = self.tree:get_node()
      if type(node.action_1) == "function" then
        node.action_1(function()
          self:refresh()
        end)
      end
    end,
    action_2 = function()
      local node = self.tree:get_node()
      if type(node.action_2) == "function" then
        node.action_2(function()
          self:refresh()
        end)
      end
    end,
    action_3 = function()
      local node = self.tree:get_node()
      if type(node.action_3) == "function" then
        node.action_3(function()
          self:refresh()
        end)
      end
    end,
    collapse = function()
      local node = self.tree:get_node()
      if not node then
        return
      end
      collapse_node(node)
    end,
    expand = function()
      local node = self.tree:get_node()
      if not node then
        return
      end
      expand_node(node)
    end,
    toggle = function()
      local node = self.tree:get_node()
      if not node then
        return
      end
      if node:is_expanded() then
        collapse_node(node)
      else
        expand_node(node)
      end
    end,
  }
end

-- Map keybindings to split window
---@private
---@param bufnr integer which buffer to map the keys in
function Drawer:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }

  local actions = self:actions()

  for act, map in pairs(self.mappings) do
    local action = actions[act]
    if action and type(action) == "function" then
      vim.keymap.set(map.mode, map.key, action, map_options)
    end
  end
end

-- sets layout to tree
---@private
---@param layout Layout[] layout to add to tree
---@param node_id? string layout is set as children to this id or root
function Drawer:set_layout(layout, node_id)
  --- recursed over Layout[] and sets it to the tree
  ---@param layouts Layout[]
  ---@return Node[] nodes list of NuiTreeNodes
  local function to_node(layouts)
    if not layouts then
      return {}
    end

    local nodes = {}
    for _, l in ipairs(layouts) do

      -- get children or set getter
      local getter
      local children
      if type(l.children) == "function" then
        getter = function()
          local exists = self.tree:get_node(l.id)
          if exists then
            self.tree:set_nodes(to_node(l.children()), l.id)
          end
        end
      else
        children = l.children
      end

      -- all other fields stay the same
      local n = vim.fn.copy(l)
      n.name = string.gsub(l.name, "\n", " ")
      n.getter = getter

      -- get existing node from the current tree and check if it is expanded
      local expanded = false
      local ex_node = self.tree:get_node(l.id)
      if ex_node and ex_node:is_expanded() then
        expanded = true
        -- if getter exists, and node is expanded, we call it
        if getter then
          children = l.children()
        end
      end
      -- recurse children
      local node = NuiTree.Node(n, to_node(children --[[@as Layout[] ]]))
      if expanded then
        node:expand()
      end

      table.insert(nodes, node)
    end

    return nodes
  end

  -- recurse layout
  self.tree:set_nodes(to_node(layout), node_id)
end

function Drawer:refresh()
  ---@type Layout[]
  local layouts = { unpack(self.editor:layout()), unpack(self.handler:layout()) }

  self:set_layout(layouts)

  self.tree:render()
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
