local dbee = require("dbee")
local ProjectorDbeeSource = require("projector_dbee.source")
local ProjectorOutput = require("projector_dbee.output")

---@class ProjectorOutputBuilder: OutputBuilder
---@field private source ProjectorDbeeSource
---@field private source_added boolean
local ProjectorOutputBuilder = {}

-- new builder
---@return ProjectorOutputBuilder
function ProjectorOutputBuilder:new()
  local o = {
    source = ProjectorDbeeSource:new(),
    source_added = false,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

-- build a new output
---@return ProjectorOutput
function ProjectorOutputBuilder:build()
  return ProjectorOutput:new()
end

---@return task_mode mode
function ProjectorOutputBuilder:mode_name()
  return "dbee"
end

---@param selection configuraiton_picks
---@return configuraiton_picks # picked configs
function ProjectorOutputBuilder:preprocess(selection)
  -- get databases from all configs
  local connections = {}
  for _, config in pairs(selection) do
    if vim.tbl_islist(config.databases) then
      for _, db in ipairs(config.databases) do
        if db.name and db.type and db.url then
          table.insert(connections, db)
        end
      end
    end
  end

  -- add connections to source
  self.source:set_conns(connections)

  -- add source to dbee (once)
  if not self.source_added then
    dbee.add_source(self.source)
    self.source_added = true
  elseif dbee.api.loaded then
    dbee.api.handler:source_reload("projector")
    dbee.api.drawer:refresh()
  end

  -- return a single manufactured task capable of running in DbeeOutput
  ---@type configuraiton_picks
  return {
    ["__dbee_output_builder_task_id__"] = {
      scope = "global",
      group = "db",
      name = "Dbee",
      evaluate = self:mode_name(),
    },
  }
end

return ProjectorOutputBuilder
