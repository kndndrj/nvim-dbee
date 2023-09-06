-- This package is used for triggering lua callbacks from go.
-- It uses unique ids to register the callbacks and trigger them.

---@alias __callback fun(stats: call_stats)

local M = {
  ---@type table<string, __callback>
  callbacks = {},
}

---@param id string id to register callback with
---@param cb __callback callback function
function M.register(id, cb)
  M.callbacks[id] = cb
end

---@param id string
---@param stats call_stats
function M.trigger(id, stats)
  stats = stats or {
    success = false,
    time_taken = 0,
  }
  local cb = M.callbacks[id] or function(_) end
  cb(stats)
end

return M
