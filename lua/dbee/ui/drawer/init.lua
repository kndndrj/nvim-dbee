local NuiTree = require("nui.tree")
local NuiLine = require("nui.line")
local common = require("dbee.ui.common")
local menu = require("dbee.ui.drawer.menu")
local convert = require("dbee.ui.drawer.convert")
local expansion = require("dbee.ui.drawer.expansion")

-- action function of drawer nodes
---@alias drawer_node_action fun(cb: fun(), select: menu_select, input: menu_input)

-- A single line in drawer tree
---@class DrawerUINode: NuiTree.Node
---@field id string unique identifier
---@field name string display name
---@field type ""|"table"|"view"|"column"|"history"|"note"|"connection"|"database_switch"|"add"|"edit"|"remove"|"help"|"source" type of node
---@field action_1? drawer_node_action primary action if function takes a second selection parameter, pick_items get picked before the call
---@field action_2? drawer_node_action secondary action if function takes a second selection parameter, pick_items get picked before the call
---@field action_3? drawer_node_action tertiary action if function takes a second selection parameter, pick_items get picked before the call
---@field lazy_children? fun():DrawerUINode[] lazy loaded child nodes

---@class DrawerUI
---@field private tree NuiTree
---@field private handler Handler
---@field private editor EditorUI
---@field private result ResultUI
---@field private mappings key_mapping[]
---@field private candies table<string, Candy> map of eye-candy stuff (icons, highlight)
---@field private disable_help boolean show help or not
---@field private winid? integer
---@field private bufnr integer
---@field private quit_handle fun() function that closes the whole ui
---@field private switch_handle fun(bufnr: integer)
---@field private current_conn_id? connection_id current active connection
---@field private current_note_id? note_id current active note
local DrawerUI = {}

---@param handler Handler
---@param editor EditorUI
---@param result ResultUI
---@param quit_handle? fun()
---@param switch_handle? fun(bufnr: integer)
---@param opts? drawer_config
---@return DrawerUI
function DrawerUI:new(handler, editor, result, quit_handle, switch_handle, opts)
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
    switch_handle = switch_handle or function() end,
    current_conn_id = current_conn.id,
    current_note_id = current_note.id,
  }
  setmetatable(o, self)
  self.__index = self

  -- create a buffer for drawer and configure it
  o.bufnr = common.create_blank_buffer("dbee-drawer", {
    buflisted = false,
    bufhidden = "delete",
    buftype = "nofile",
    swapfile = false,
  })
  common.configure_buffer_mappings(o.bufnr, o:get_actions(), opts.mappings)
  common.configure_buffer_quit_handle(o.bufnr, o.quit_handle)

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
---@param data { conn_id: connection_id }
function DrawerUI:on_current_connection_changed(data)
  if self.current_conn_id == data.conn_id then
    return
  end
  self.current_conn_id = data.conn_id
  self:refresh()
end

-- event listener for current note change
---@private
---@param data { note_id: note_id }
function DrawerUI:on_current_note_changed(data)
  if self.current_note_id == data.note_id then
    return
  end
  self.current_note_id = data.note_id
  self:refresh()
end

---@private
---@param bufnr integer
---@return NuiTree tree
function DrawerUI:create_tree(bufnr)
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
---@return table<string, fun()>
function DrawerUI:get_actions()
  local function collapse_node(node)
    if node:collapse() then
      self.tree:render()
    end
  end

  local function expand_node(node)
    local expanded = node:is_expanded()

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
      menu.select {
        relative_winid = self.winid,
        title = opts.title or "",
        mappings = self.mappings,
        items = opts.items or {},
        on_confirm = opts.on_confirm,
        on_yank = opts.on_yank,
      }
    end, function(opts)
      menu.input {
        relative_winid = self.winid,
        title = opts.title or "",
        mappings = self.mappings,
        default_value = opts.default or "",
        on_confirm = opts.on_confirm,
      }
    end)
  end

  return {
    quit = self.quit_handle,
    refresh = function()
      self:refresh()
    end,
    action_1 = function()
      local node = self.tree:get_node() --[[@as DrawerUINode]]
      if not node then
        return
      end
      perform_action(node.action_1)
    end,
    action_2 = function()
      local node = self.tree:get_node() --[[@as DrawerUINode]]
      if not node then
        return
      end
      perform_action(node.action_2)
    end,
    action_3 = function()
      local node = self.tree:get_node() --[[@as DrawerUINode]]
      if not node then
        return
      end
      perform_action(node.action_3)
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

function DrawerUI:refresh()
  -- assemble tree layout
  ---@type DrawerUINode[]
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
function DrawerUI:show(winid)
  self.winid = winid

  -- configure window options
  common.configure_window_options(self.winid, {
    wrap = false,
    winfixheight = true,
    winfixwidth = true,
    number = false,
    relativenumber = false,
    spell = false,
  })

  -- configure window immutablity
  common.configure_window_immutable_buffer(self.winid, self.bufnr, self.switch_handle)

  -- set buffer to window
  vim.api.nvim_win_set_buf(self.winid, self.bufnr)

  self:refresh()
end

return DrawerUI
