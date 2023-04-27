local Output = require("projector.contract.output")
local dbee = require("dbee")
local dbee_helpers = require("dbee.helpers")

---@type Output
local DbeeOutput = Output:new()

---@param configuration Configuration
---@diagnostic disable-next-line: unused-local
function DbeeOutput:init(configuration)
  if not configuration then
    self:done(false)
    return
  end

  for setting, config in pairs(configuration) do
    if setting == "databases" then
      for _, db in ipairs(config) do
        dbee.add_connection(db)
      end
    elseif setting == "queries" then
      dbee_helpers.add(config)
    end
  end

  self.status = "inactive"
  self:show()

  self:done(true)
end

function DbeeOutput:show()
  dbee.open()
  self.status = "visible"
end

function DbeeOutput:hide()
  dbee.close()
  self.status = "hidden"
end

function DbeeOutput:kill()
  self:hide()
  self.status = "inactive"
end

---@return Action[]|nil
function DbeeOutput:list_actions()
  if not dbee.api.editor:is_current_window() then
    return
  end

  local actions = {}

  for name, action in pairs(dbee.api.editor:actions()) do
    -- prettier format for labels
    local label = name:gsub("_", " ")
    label = string.gsub(" " .. label, "%W%l", string.upper):sub(2)
    table.insert(actions, {
      label = label,
      action = action,
    })
  end

  table.sort(actions, function(k1, k2)
    return k1.label < k2.label
  end)

  return actions
end

return DbeeOutput
