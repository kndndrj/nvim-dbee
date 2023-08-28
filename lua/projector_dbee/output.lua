local dbee = require("dbee")

---@class ProjectorOutput: Output
---@field private state output_status
local ProjectorOutput = {}

---@return ProjectorOutput
function ProjectorOutput:new()
  local o = {}
  setmetatable(o, self)
  self.__index = self
  return o
end

---@return output_status
function ProjectorOutput:status()
  if dbee.api.open then
    return "visible"
  else
    return "hidden"
  end
end

---@param _ task_configuration
---@param callback fun(success: boolean)
function ProjectorOutput:init(_, callback)
  -- due to evaluation specification in the
  -- output builder, we don't have to do anything
  callback(true)
end

function ProjectorOutput:show()
  dbee.open()
end

function ProjectorOutput:hide()
  dbee.close()
end

function ProjectorOutput:kill()
  self:hide()
end

return ProjectorOutput
