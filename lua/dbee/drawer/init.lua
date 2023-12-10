local NuiTree = require("nui.tree")
local NuiLine = require("nui.line")
local ui_helper = require("dbee.ui_helper")
local menu = require("dbee.drawer.menu")
local convert = require("dbee.drawer.convert")
local expansion = require("dbee.drawer.expansion")

---@class Candy
---@field icon string
---@field icon_highlight string
---@field text_highlight string

-- action function of drawer nodes
---@alias drawer_node_action fun(cb: fun(), pick: fun(opts?: { title: string, items: string[], on_select: fun(selection: string) }))

-- A single line in drawer tree
---@class DrawerNode: NuiTree.Node
---@field id string unique identifier
---@field name string display name
---@field type ""|"table"|"history"|"note"|"connection"|"database_switch"|"add"|"edit"|"remove"|"help"|"source"|"view" type of node
---@field action_1? drawer_node_action primary action if function takes a second selection parameter, pick_items get picked before the call
---@field action_2? drawer_node_action secondary action if function takes a second selection parameter, pick_items get picked before the call
---@field action_3? drawer_node_action tertiary action if function takes a second selection parameter, pick_items get picked before the call
---@field lazy_children? fun():DrawerNode[] lazy loaded child nodes

---@alias drawer_config { disable_candies: boolean, candies: table<string, Candy>, mappings: table<string, mapping>, disable_help: boolean }

---@class Drawer
---@field private tree NuiTree
---@field private handler Handler
---@field private editor Editor
---@field private result Result
---@field private mappings table<string, mapping>
---@field private candies table<string, Candy> map of eye-candy stuff (icons, highlight)
---@field private disable_help boolean show help or not
---@field private winid? integer
---@field private bufnr integer
---@field private quit_handle fun() function that closes the whole ui
---@field private current_conn_id? conn_id current active connection
---@field private current_note_id? note_id current active note
local Drawer = {}

---@param handler Handler
---@param editor Editor
---@param result Result
---@param quit_handle? fun()
---@param opts? drawer_config
---@return Drawer
function Drawer:new(handler, editor, result, quit_handle, opts)
  opts = opts or {}

  if not handler then
    error("no Handler provided to Drawer")
  end
  if not editor then
    error("no Editor provided to Drawer")
  end
  if not result then
    error("no Result provided to Drawer")
  end

  local candies = {}
  if not opts.disable_candies then
    candies = opts.candies or {}
  end

  local current_conn = handler:get_current_connection() or {}
  local current_note = editor:get_current_note() or {}

  -- class object
  local o = {
    handler = handler,
    editor = editor,
    result = result,
    mappings = opts.mappings or {},
    candies = candies,
    disable_help = opts.disable_help or false,
    quit_handle = quit_handle or function() end,
    current_conn_id = current_conn.id,
    current_note_id = current_note.id,
  }
  setmetatable(o, self)
  self.__index = self

  -- create a buffer for drawer and configure it
  o.bufnr = ui_helper.create_blank_buffer("dbee-drawer", {
    buflisted = false,
    bufhidden = "delete",
    buftype = "nofile",
    swapfile = false,
  })
  ui_helper.configure_buffer_mappings(o.bufnr, o:generate_keymap(opts.mappings))
  ui_helper.configure_buffer_quit_handle(o.bufnr, o.quit_handle)

  -- create tree
  o.tree = o:create_tree(o.bufnr)

  -- listen to events
  handler:register_event_listener("current_connection_changed", function(data)
    o:on_current_connection_changed(data)
  end)

  editor:register_event_listener("current_note_changed", function(data)
    o:on_current_note_changed(data)
  end)

  return o
end

-- event listener for current connection change
---@private
---@param data { conn_id: conn_id }
function Drawer:on_current_connection_changed(data)
  if self.current_conn_id == data.conn_id then
    return
  end
  self.current_conn_id = data.conn_id
  self:refresh()
end

-- event listener for current note change
---@private
---@param data { note_id: note_id }
function Drawer:on_current_note_changed(data)
  if self.current_note_id == data.note_id then
    return
  end
  self.current_note_id = data.note_id
  self:refresh()
end

---@private
---@param bufnr integer
---@return NuiTree tree
function Drawer:create_tree(bufnr)
  return NuiTree {
    bufnr = bufnr,
    prepare_node = function(node)
      local line = NuiLine()

      line:append(string.rep("  ", node:get_depth() - 1))

      if node:has_children() or node.lazy_children then
        local candy = self.candies["node_closed"] or { icon = ">", icon_highlight = "NonText" }
        if node:is_expanded() then
          candy = self.candies["node_expanded"] or { icon = "v", icon_highlight = "NonText" }
        end
        line:append(candy.icon .. " ", candy.icon_highlight)
      else
        line:append("  ")
      end

      ---@type Candy
      local candy
      -- special icons for nodes without type
      if not node.type or node.type == "" then
        if node:has_children() then
          candy = self.candies["none_dir"]
        else
          candy = self.candies["none"]
        end
      else
        candy = self.candies[node.type] or {}
      end
      candy = candy or {}

      if candy.icon then
        line:append(" " .. candy.icon .. " ", candy.icon_highlight)
      end

      -- apply a special highlight for active connection and active note
      if node.id == self.current_conn_id or self.current_note_id == node.id then
        line:append(string.gsub(node.name, "\n", " "), candy.icon_highlight)
      else
        line:append(string.gsub(node.name, "\n", " "), candy.text_highlight)
      end

      return line
    end,
    get_node_id = function(node)
      if node.id then
        return node.id
      end
      return tostring(math.random())
    end,
  }
end

---@private
---@param mappings table<string, mapping>
---@return keymap[]
function Drawer:generate_keymap(mappings)
  mappings = mappings or {}

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
        if not nested_node then
          return
        end
        nested_node:expand()
        expand_all_single(nested_node)
      end
    end

    local expanded = node:is_expanded()

    expand_all_single(node)

    -- if function for getting layout exist, call it
    if not expanded and type(node.lazy_children) == "function" then
      self.tree:set_nodes(node.lazy_children(), node.id)
    end

    node:expand()

    self.tree:render()
  end

  -- wrapper for actions (e.g. action_1, action_2, action_3)
  ---@param action drawer_node_action
  local function perform_action(action)
    if type(action) ~= "function" then
      return
    end

    action(function()
      self:refresh()
    end, function(opts)
      opts = opts or {}
      menu.open(self.winid, opts.items or {}, opts.on_select or function() end, opts.title or "")
    end)
  end

  return {
    {
      action = self.quit_handle,
      mapping = mappings["quit"],
    },
    {
      action = function()
        self:refresh()
      end,
      mapping = mappings["refresh"],
    },
    {
      action = function()
        local node = self.tree:get_node() --[[@as DrawerNode]]
        if not node then
          return
        end
        perform_action(node.action_1)
      end,
      mapping = mappings["action_1"],
    },
    {
      action = function()
        local node = self.tree:get_node() --[[@as DrawerNode]]
        if not node then
          return
        end
        perform_action(node.action_2)
      end,
      mapping = mappings["action_2"],
    },
    {
      action = function()
        local node = self.tree:get_node() --[[@as DrawerNode]]
        if not node then
          return
        end
        perform_action(node.action_3)
      end,
      mapping = mappings["action_3"],
    },
    {
      action = function()
        local node = self.tree:get_node()
        if not node then
          return
        end
        collapse_node(node)
      end,
      mapping = mappings["collapse"],
    },
    {
      action = function()
        local node = self.tree:get_node()
        if not node then
          return
        end
        expand_node(node)
      end,
      mapping = mappings["expand"],
    },
    {
      action = function()
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
      mapping = mappings["toggle"],
    },
  }
end

function Drawer:refresh()
  -- assemble tree layout
  ---@type DrawerNode[]
  local nodes = {}
  local editor_nodes = convert.editor_nodes(self.editor, self.current_conn_id, function()
    self:refresh()
  end)
  for _, ly in ipairs(editor_nodes) do
    table.insert(nodes, ly)
  end
  table.insert(nodes, convert.separator_node())
  for _, ly in ipairs(convert.handler_nodes(self.handler, self.result)) do
    table.insert(nodes, ly)
  end

  if not self.disable_help then
    table.insert(nodes, convert.separator_node())
    table.insert(nodes, convert.help_node(self.mappings))
  end

  local exp = expansion.get(self.tree)
  self.tree:set_nodes(nodes)
  expansion.set(self.tree, exp)

  self.tree:render()
end

---@param winid integer
function Drawer:show(winid)
  self.winid = winid

  -- configure window options
  ui_helper.configure_window_options(self.winid, {
    wrap = false,
    winfixheight = true,
    winfixwidth = true,
    number = false,
  })

  -- set buffer to window
  vim.api.nvim_win_set_buf(self.winid, self.bufnr)

  self:refresh()
end

return Drawer