local utils = require("dbee.utils")

---@mod dbee.ref.sources Sources
---@brief [[
---Sources can be created by implementing the Source interface.
---Some methods are optional and are related to updating/editing functionality.
---@brief ]]

---ID of a source.
---@alias source_id string

---Action enum for saving connections to source.
---@alias source_save_action
---| '"add"'
---| '"delete"'

---Source interface
---@class Source
---@field name fun(self: Source):string function to return the name of the source
---@field load fun(self: Source):ConnectionParams[] function to load connections from external source
---@field save? fun(self: Source, conns: ConnectionParams[], action: source_save_action) function to save connections to external source (optional)
---@field file? fun(self: Source):string function which returns a source file to edit (optional)

local sources = {}

---@divider -

---Built-In File Source.
---@class FileSource: Source
---@field private path string path to file
sources.FileSource = {}

--- Loads connections from json file
---@param path string path to file
---@return Source
function sources.FileSource:new(path)
  if not path then
    error("no path provided")
  end
  local o = {
    path = path,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@package
---@return string
function sources.FileSource:name()
  return vim.fs.basename(self.path)
end

---@package
---@return ConnectionParams[]
function sources.FileSource:load()
  local path = self.path

  ---@type ConnectionParams[]
  local conns = {}

  if not vim.loop.fs_stat(path) then
    return {}
  end

  local lines = {}
  for line in io.lines(path) do
    if not vim.startswith(vim.trim(line), "//") then
      table.insert(lines, line)
    end
  end

  local contents = table.concat(lines, "\n")
  local ok, data = pcall(vim.fn.json_decode, contents)
  if not ok then
    utils.log("warn", 'Could not parse json file: "' .. path .. '".', "sources")
    return {}
  end

  for _, conn in pairs(data) do
    if type(conn) == "table" then
      table.insert(conns, conn)
    end
  end

  return conns
end

---@package
---@param conns ConnectionParams[]
---@param action "add"|"delete"
function sources.FileSource:save(conns, action)
  local path = self.path

  if not conns or vim.tbl_isempty(conns) then
    return
  end

  -- read from file
  local existing = self:load()

  ---@type ConnectionParams[]
  local new = {}

  if action == "add" then
    for _, to_add in ipairs(conns) do
      local edited = false
      for i, ex_conn in ipairs(existing) do
        if to_add.id == ex_conn.id then
          existing[i] = to_add
          edited = true
        end
      end

      if not edited then
        table.insert(existing, to_add)
      end
    end
    new = existing
  elseif action == "delete" then
    for _, to_remove in ipairs(conns) do
      for i, ex_conn in ipairs(existing) do
        if to_remove.id == ex_conn.id then
          table.remove(existing, i)
        end
      end
    end
    new = existing
  end

  -- write back to file
  local ok, json = pcall(vim.fn.json_encode, new)
  if not ok then
    utils.log("error", "Could not convert connection list to json", "sources")
    return
  end

  -- overwrite file
  local file = assert(io.open(path, "w+"), "could not open file")
  file:write(json)
  file:close()
end

---@package
---@return string
function sources.FileSource:file()
  return self.path
end

---@divider -

---Built-In Env Source.
---Loads connections from json string of env variable.
---@class EnvSource: Source
---@field private var string path to file
sources.EnvSource = {}

---@param var string env var to load connections from
---@return Source
function sources.EnvSource:new(var)
  if not var then
    error("no path provided")
  end
  local o = {
    var = var,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@package
---@return string
function sources.EnvSource:name()
  return self.var
end

---@package
---@return ConnectionParams[]
function sources.EnvSource:load()
  ---@type ConnectionParams[]
  local conns = {}

  local raw = os.getenv(self.var)
  if not raw then
    return {}
  end

  local ok, data = pcall(vim.fn.json_decode, raw)
  if not ok then
    utils.log("warn", 'Could not parse connections from env: "' .. self.var .. '".', "sources")
    return {}
  end

  for _, conn in pairs(data) do
    if type(conn) == "table" and conn.url and conn.type then
      table.insert(conns, conn)
    end
  end

  return conns
end

---@divider -

---Built-In Memory Source.
---Loads connections from lua table.
---@class MemorySource: Source
---@field conns ConnectionParams[]
sources.MemorySource = {}

---@param conns ConnectionParams[] list of connections
---@return Source
function sources.MemorySource:new(conns)
  local o = {
    conns = conns or {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@package
---@return string
function sources.MemorySource:name()
  return "memory"
end

---@package
---@return ConnectionParams[]
function sources.MemorySource:load()
  return self.conns
end

return sources
