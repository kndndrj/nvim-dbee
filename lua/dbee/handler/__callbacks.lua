-- This package is used for triggering lua callbacks from go.
-- It uses unique ids to register the callbacks and trigger them.

local M = {
  ---@type table<string, fun(success: boolean)>
  callbacks = {},
}

---@param id string id to register callback with
---@param cb fun(success: boolean) callback function
function M.register(id, cb)
  M.callbacks[id] = cb
end

---@param id string
---@param success boolean
function M.trigger(id, success)
  success = success or false
  local cb = M.callbacks[id] or function(_) end
  cb(success)
end

return M
