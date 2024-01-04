-- This package is used for triggering lua callbacks from go.
-- It uses unique ids to register the callbacks and trigger them.
local M = {}

---@type table<core_event_name, event_listener[]>
local callbacks = {}

---@param event core_event_name event name to register the callback for
---@param cb event_listener callback function - "data" argument type depends on the event
function M.register(event, cb)
  callbacks[event] = callbacks[event] or {}
  table.insert(callbacks[event], cb)
end

---@param event core_event_name
---@param data any
function M.trigger(event, data)
  vim.schedule(function()
    local cbs = callbacks[event] or {}
    for _, cb in ipairs(cbs) do
      cb(data)
    end
  end)
end

return M
