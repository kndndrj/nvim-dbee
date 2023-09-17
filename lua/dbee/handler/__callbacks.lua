-- This package is used for triggering lua callbacks from go.
-- It uses unique ids to register the callbacks and trigger them.

---@alias event_name "call_state_changed"|"current_conn_changed"
---@alias event_listener fun(data: any)

local M = {
  ---@type table<string, event_listener[]>
  callbacks = {},
}

---@param event event_name event name to register the callback for
---@param cb event_listener callback function - "data" argument type depends on the event
function M.register(event, cb)
  M.callbacks[event] = M.callbacks[event] or {}
  table.insert(M.callbacks[event], cb)
end

---@param event event_name
---@param data any
function M.trigger(event, data)
  local callbacks = M.callbacks[event] or {}
  for _, cb in ipairs(callbacks) do
    cb(data)
  end
end

return M
