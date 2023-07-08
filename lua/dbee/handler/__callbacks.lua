-- This package is used for triggering lua callbacks from go.
-- It uses unique ids to register the callbacks and trigger them.

local M = {
  ---@type table<string, fun()>
  callbacks = {},
}

---@param id string id to register callback with
---@param cb fun() callback function
function M.register(id, cb)
  M.callbacks[id] = cb
end

function M.trigger(id)
  local cb = M.callbacks[id] or function() end
  cb()
end

return M
