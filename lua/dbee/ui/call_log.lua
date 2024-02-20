local NuiLine = require("nui.line")
local NuiTree = require("nui.tree")
local utils = require("dbee.utils")
local common = require("dbee.ui.common")

-- CallLogTile is a call history.
---@class CallLogUI
---@field private result ResultUI
---@field private handler Handler
---@field private tree NuiTree
---@field private winid? integer
---@field private bufnr integer
---@field private candies table<string, Candy> map of eye-candy stuff (icons, highlight)
---@field private current_connection_id? connection_id
---@field private hover_close? fun() function that closes the hover window
---@field private switch_handle fun(bufnr: integer)
local CallLogTile = {}

---@param handler Handler
---@param result ResultUI
---@param quit_handle? fun()
---@param switch_handle? fun(bufnr: integer)
---@param opts call_log_config
---@return CallLogUI
function CallLogTile:new(handler, result, quit_handle, switch_handle, opts)
  opts = opts or {}
  quit_handle = quit_handle or function() end

  if not handler then
    error("no Handler passed to CallLogTile")
  end
  if not result then
    error("no ResultTile passed to CallLogTile")
  end

  local candies = {}
  if not opts.disable_candies then
    candies = opts.candies or {}
  end

  ---@type CallLogUI
  local o = {
    handler = handler,
    result = result,
    candies = candies,
    hover_close = function() end,
    switch_handle = switch_handle or function() end,
  }
  setmetatable(o, self)
  self.__index = self

  -- create a buffer for drawer and configure it
  o.bufnr = common.create_blank_buffer("dbee-call-log", {
    buflisted = false,
    bufhidden = "delete",
    buftype = "nofile",
    swapfile = false,
  })
  common.configure_buffer_mappings(o.bufnr, o:get_actions(), opts.mappings)
  common.configure_buffer_quit_handle(o.bufnr, quit_handle)

  -- create the tree
  o.tree = o:create_tree(o.bufnr)

  handler:register_event_listener("call_state_changed", function(data)
    ---@diagnostic disable-next-line
    o:on_call_state_changed(data)
  end)
  handler:register_event_listener("current_connection_changed", function(data)
    ---@diagnostic disable-next-line
    o:on_current_connection_changed(data)
  end)

  return o
end

-- event listener for new calls
---@private
---@param _ { call: CallDetails }
function CallLogTile:on_call_state_changed(_)
  self:refresh()
end

-- event listener for current connection change
---@private
---@param data { conn_id: connection_id }
function CallLogTile:on_current_connection_changed(data)
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
function CallLogTile:create_tree(bufnr)
  return NuiTree {
    bufnr = bufnr,
    prepare_node = function(node)
      ---@type CallDetails
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
---@return table<string, fun()>
function CallLogTile:get_actions()
  return {
    show_result = function()
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
    cancel_call = function()
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
  }
end

function CallLogTile:refresh()
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
function CallLogTile:configure_preview(bufnr)
  utils.create_singleton_autocmd({ "CursorMoved", "BufEnter" }, {
    buffer = bufnr,
    callback = function()
      self.hover_close()

      local node = self.tree:get_node()
      if not node then
        return
      end
      ---@type CallDetails?
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

      if call.error and call.error ~= "" then
        table.insert(call_summary, string.format("error:                %s", string.gsub(call.error, "\n", " ")))
      end

      self.hover_close = common.float_hover(self.winid, call_summary)
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
function CallLogTile:show(winid)
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

  -- configure auto preview
  self:configure_preview(self.bufnr)

  -- set buffer to window
  vim.api.nvim_win_set_buf(self.winid, self.bufnr)

  self:refresh()
end

return CallLogTile
