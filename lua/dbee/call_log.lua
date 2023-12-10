local NuiLine = require("nui.line")
local NuiTree = require("nui.tree")
local floats = require("dbee.floats")
local utils = require("dbee.utils")
local ui_helper = require("dbee.ui_helper")

-- CallLog is a call history
---@class CallLog
---@field private result Result
---@field private handler Handler
---@field private tree NuiTree
---@field private winid? integer
---@field private bufnr integer
---@field private candies table<string, Candy> map of eye-candy stuff (icons, highlight)
---@field private current_connection_id? conn_id
---@field private hover_close? fun() function that closes the hover window
local CallLog = {}

---@alias call_log_config { mappings: table<string, mapping>, disable_candies: boolean, candies: table<string, Candy> }

---@param handler Handler
---@param result Result
---@param quit_handle? fun()
---@param opts call_log_config
---@return CallLog
function CallLog:new(handler, result, quit_handle, opts)
  opts = opts or {}
  quit_handle = quit_handle or function() end

  if not handler then
    error("no Handler passed to CallLog")
  end
  if not result then
    error("no Result passed to CallLog")
  end

  local candies = {}
  if not opts.disable_candies then
    candies = opts.candies or {}
  end

  ---@type CallLog
  local o = {
    handler = handler,
    result = result,
    candies = candies,
    hover_close = function() end,
  }
  setmetatable(o, self)
  self.__index = self

  -- create a buffer for drawer and configure it
  o.bufnr = ui_helper.create_blank_buffer("dbee-call-log", {
    buflisted = false,
    bufhidden = "delete",
    buftype = "nofile",
    swapfile = false,
  })
  ui_helper.configure_buffer_mappings(o.bufnr, o:generate_keymap(opts.mappings))
  ui_helper.configure_buffer_quit_handle(o.bufnr, quit_handle)

  -- create the tree
  o.tree = o:create_tree(o.bufnr)

  handler:register_event_listener("call_state_changed", function(data)
    o:on_call_state_changed(data)
  end)
  handler:register_event_listener("current_connection_changed", function(data)
    o:on_current_connection_changed(data)
  end)

  return o
end

-- event listener for new calls
---@private
---@param _ { call: call_details }
function CallLog:on_call_state_changed(_)
  self:refresh()
end

-- event listener for current connection change
---@private
---@param data { conn_id: conn_id }
function CallLog:on_current_connection_changed(data)
  self.current_connection_id = data.conn_id
  self:refresh()
end

---@param str string
---@param len integer
---@return string # string of length
local function make_length(str, len)
  local orig_len = vim.fn.strchars(str)
  if orig_len > len then
    return str:sub(1, len - 1) .. "…"
  elseif orig_len < len then
    return str .. string.rep(" ", len - orig_len)
  end

  -- same length
  return str
end

-- returns the initials of the call state
---@param state call_state
---@return string # string of length
local function call_state_initials(state)
  if not state then
    return "  "
  end

  local initials = ""
  for word in string.gmatch(state, "([^_]+)") do
    initials = initials .. word:sub(1, 1)
  end

  if #initials < 2 then
    initials = initials .. string.rep(" ", 2 - #initials)
  end

  return initials
end

---@private
---@param bufnr integer
---@return NuiTree
function CallLog:create_tree(bufnr)
  return NuiTree {
    bufnr = bufnr,
    prepare_node = function(node)
      ---@type call_details
      local call = node.call
      local line = NuiLine()
      if not call then
        if node.text then
          line:append(node.text, "NonText")
        end
        return line
      end

      local candy = self.candies[call.state]
        or { icon = call_state_initials(call.state), icon_highlight = "", text_highlight = "" }

      local state_preview = candy.icon
      if not state_preview or state_preview == "" then
        state_preview = call_state_initials(call.state)
      end

      line:append(make_length(state_preview, 3), candy.icon_highlight)
      line:append(" ┃ ", "NonText")
      line:append(make_length(string.gsub(call.query, "\n", " "), 40), candy.text_highlight)

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
function CallLog:generate_keymap(mappings)
  mappings = mappings or {}

  return {
    {
      action = function()
        local node = self.tree:get_node()
        if not node then
          return
        end
        local call = node.call
        if not call then
          return
        end

        if call.state == "archived" or call.state == "retrieving" then
          self.result:set_call(call)
          self.result:page_current()
        end
      end,
      mapping = mappings["show_result"],
    },
    {
      action = function()
        local node = self.tree:get_node()
        if not node then
          return
        end
        local call = node.call
        if not call then
          return
        end

        self.handler:call_cancel(call.id)
      end,
      mapping = mappings["cancel"],
    },
  }
end

---@private
function CallLog:refresh()
  if not self.current_connection_id then
    return
  end
  local calls = self.handler:connection_get_calls(self.current_connection_id)

  -- dummy node if no calls
  if vim.tbl_isempty(calls) then
    self.tree:set_nodes { NuiTree.Node { id = tostring(math.random()), text = "Call log will be displayed here!" } }
    self.tree:render()
    return
  end

  table.sort(calls, function(k1, k2)
    return k1.timestamp_us > k2.timestamp_us
  end)

  local nodes = {}
  for _, c in ipairs(calls) do
    table.insert(nodes, NuiTree.Node { id = tostring(math.random()), call = c })
  end

  self.tree:set_nodes(nodes)
  self.tree:render()
end

---@private
---@param bufnr integer
function CallLog:configure_preview(bufnr)
  utils.create_singleton_autocmd({ "CursorMoved", "BufEnter" }, {
    buffer = bufnr,
    callback = function()
      self.hover_close()

      local node = self.tree:get_node()
      if not node then
        return
      end
      ---@type call_details?
      local call = node.call
      if not call then
        return
      end

      local call_summary = {
        string.format("id:                   %s", call.id),
        string.format("query:                %s", string.gsub(call.query, "\n", " ")),
        string.format("state:                %s", call.state),
        string.format("time_taken [seconds]: %.3f", (call.time_taken_us or 0) / 1000000),
        string.format("timestamp:            %s", tostring(os.date("%c", (call.timestamp_us or 0) / 1000000))),
      }

      self.hover_close = floats.hover(self.winid, call_summary)
    end,
  })

  utils.create_singleton_autocmd({ "BufLeave", "QuitPre" }, {
    buffer = bufnr,
    callback = function()
      self.hover_close()
    end,
  })
end

---@param winid integer
function CallLog:show(winid)
  self.winid = winid

  -- configure window options
  ui_helper.configure_window_options(self.winid, {
    wrap = false,
    winfixheight = true,
    winfixwidth = true,
    number = false,
  })

  -- configure auto preview
  self:configure_preview(self.bufnr)

  -- set buffer to window
  vim.api.nvim_win_set_buf(self.winid, self.bufnr)

  self:refresh()
end

return CallLog
