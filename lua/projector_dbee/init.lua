local dbee = require("dbee")
local ProjectorDbeeSource = require("projector_dbee.source")
local ProjectorOutput = require("projector_dbee.output")

local M = {}

---@class ProjectorOutputBuilder: OutputBuilder
---@field private source ProjectorDbeeSource
---@field private source_added boolean
---@field private name string
M.OutputBuilder = {}

-- new builder
---@return ProjectorOutputBuilder
function M.OutputBuilder:new()
  local o = {
    source = ProjectorDbeeSource:new(),
    source_added = false,
    name = "Dbee",
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

-- build a new output
---@return ProjectorOutput
function M.OutputBuilder:build()
  return ProjectorOutput:new()
end

---@return task_mode mode
function M.OutputBuilder:mode_name()
  return "dbee"
end

---@param configuration task_configuration
---@return boolean
function M.OutputBuilder:validate(configuration)
  if configuration and configuration.name == self.name and configuration.evaluate == self:mode_name() then
    return true
  end
  return false
end

---@param configurations task_configuration[]
---@return task_configuration[]
function M.OutputBuilder:preprocess(configurations)
  -- get databases from all configs
  local connections = {}

  ---@param cfgs task_configuration[]
  local function parse(cfgs)
    for _, cfg in pairs(cfgs) do
      if vim.tbl_islist(cfg.databases) then
        for _, db in ipairs(cfg.databases) do
          if db.name and db.type and db.url then
            table.insert(connections, db)
          end
        end
      end

      if vim.tbl_islist(cfg.children) then
        parse(cfg.children)
      end
    end
  end

  parse(configurations)

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
  return {
    {
      id = "__dbee_task_unique_id__",
      name = "Dbee",
      evaluate = self:mode_name(),
    },
  }
end

return M
