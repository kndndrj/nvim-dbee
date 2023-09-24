local NuiLine = require("nui.line")
local NuiTree = require("nui.tree")
local floats = require("dbee.floats")

-- CallLog is a call history
---@class CallLog
---@field private ui Ui
---@field private result Result
---@field private handler Handler
---@field private tree? NuiTree
---@field private candies table<string, Candy> map of eye-candy stuff (icons, highlight)
---@field private current_connection_id? conn_id
---@field private hover_close? fun() function that closes the hover window
local CallLog = {}

---@alias call_log_config { mappings: table<string, mapping>, disable_candies: boolean, candies: table<string, Candy> }

---@param ui Ui
---@param handler Handler
---@param result Result
---@param opts call_log_config
---@return Result
function CallLog:new(ui, handler, result, opts)
  opts = opts or {}

  if not handler then
    error("no Handler passed to CallLog")
  end
  if not result then
    error("no Result passed to CallLog")
  end
  if not ui then
    error("no Ui passed to CallLog")
  end

  local candies = {}
  if not opts.disable_candies then
    candies = opts.candies or {}
  end

  -- class object
  local o = {
    ui = ui,
    handler = handler,
    result = result,
    candies = candies,
    hover_close = function() end,
  }
  setmetatable(o, self)
  self.__index = self

  -- set keymaps
  o.ui:set_keymap(o:generate_keymap(opts.mappings))

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

      local candy = self.candies[call.state]
        or { icon = call_state_initials(call.state), icon_highlight = "", text_highlight = "" }

      local state_preview = candy.icon
      if not state_preview or state_preview == "" then
        state_preview = call_state_initials(call.state)
      end

      line:append(make_length(state_preview, 3), candy.icon_highlight)
      line:append(" ┃ ", "NonText")
      line:append(make_length(call.query, 40), candy.text_highlight)

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
  vim.api.nvim_create_autocmd({ "CursorMoved", "BufEnter" }, {
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
        string.format("query:                %s", call.query),
        string.format("state:                %s", call.state),
        string.format("time_taken [seconds]: %.3f", (call.time_taken_us or 0) / 1000000),
        string.format("timestamp:            %s", tostring(os.date("%c", (call.timestamp_us or 0) / 1000000))),
      }

      self.hover_close = floats.hover(self.ui:window(), call_summary)
    end,
  })

  vim.api.nvim_create_autocmd({ "BufLeave", "QuitPre" }, {
    buffer = bufnr,
    callback = function()
      self.hover_close()
    end,
  })
end

-- Show drawer on screen
function CallLog:open()
  local _, bufnr = self.ui:open()

  -- tree
  if not self.tree then
    self.tree = self:create_tree(bufnr)
  end
  self.tree.bufnr = bufnr

  self:configure_preview(bufnr)

  self:refresh()
end

function CallLog:close()
  self.ui:close()
end

return CallLog
