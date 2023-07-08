-- Helpers to execute resources only once

-- private variable with registered onces
---@type table<string, boolean>
local onces = {}

---@class Once<T>
---@field private id string
local Once = {}

---@param id string unique id of this resource
---@return Once
function Once:new(id)
  id = id or ""

  -- register as active if not used
  if onces[id] == nil then
    onces[id] = true
  end

  -- class object
  local o = {
    id = id,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@return boolean # false - has been used ... true - wasn't yet used
function Once:poke()
  local state = onces[self.id] or false
  onces[self.id] = false
  return state
end

return Once
