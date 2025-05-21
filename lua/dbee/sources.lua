local utils = require("dbee.utils")

---@mod dbee.ref.sources Sources
---@brief [[
---Sources can be created by implementing the Source interface.
---Some methods are optional and are related to updating/editing functionality.
---@brief ]]

---ID of a source.
---@alias source_id string

---Source interface
---"name" and "load" methods are mandatory for basic functionality.
---"create", "update" and "delete" methods are optional and provide interactive CRUD.
---"file" method is used for providing optional manual edits of the source's file.
---A source is also in charge of managing ids of connections. A connection parameter without
---a unique id results in an error or undefined behavior.
---@class Source
---@field name fun(self: Source):string function to return the name of the source
---@field load fun(self: Source):ConnectionParams[] function to load connections from external source
---@field create? fun(self: Source, details: ConnectionParams):connection_id create a connection and return its id (optional)
---@field delete? fun(self: Source, id: connection_id) delete a connection from its id (optional)
---@field update? fun(self: Source, id: connection_id, details: ConnectionParams) update provided connection (optional)
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
    error('Could not parse json file: "' .. path .. '".')
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
---@param conn ConnectionParams
---@return connection_id
function sources.FileSource:create(conn)
  local path = self.path

  if not conn or vim.tbl_isempty(conn) then
    error("cannot create an empty connection")
  end

  -- read from file
  local existing = self:load()

  conn.id = "file_source_/" .. utils.random_string()
  table.insert(existing, conn)

  -- write back to file
  local ok, json = pcall(vim.fn.json_encode, existing)
  if not ok then
    error("could not convert connection list to json")
  end

  -- overwrite file
  local file = assert(io.open(path, "w+"), "could not open file")
  file:write(json)
  file:close()

  return conn.id
end

---@package
---@param id connection_id
function sources.FileSource:delete(id)
  local path = self.path

  if not id or id == "" then
    error("no id passed to delete function")
  end

  -- read from file
  local existing = self:load()

  local new = {}
  for _, ex in ipairs(existing) do
    if ex.id ~= id then
      table.insert(new, ex)
    end
  end

  -- write back to file
  local ok, json = pcall(vim.fn.json_encode, new)
  if not ok then
    error("could not convert connection list to json")
    return
  end

  -- overwrite file
  local file = assert(io.open(path, "w+"), "could not open file")
  file:write(json)
  file:close()
end

---@package
---@param id connection_id
---@param details ConnectionParams
function sources.FileSource:update(id, details)
  local path = self.path

  if not id or id == "" then
    error("no id passed to update function")
  end

  if not details or vim.tbl_isempty(details) then
    error("cannot create an empty connection")
  end

  -- read from file
  local existing = self:load()

  for _, ex in ipairs(existing) do
    if ex.id == id then
      ex.name = details.name
      ex.url = details.url
      ex.type = details.type
    end
  end

  -- write back to file
  local ok, json = pcall(vim.fn.json_encode, existing)
  if not ok then
    error("could not convert connection list to json")
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
    error('Could not parse connections from env: "' .. self.var .. '".')
    return {}
  end

  for i, conn in pairs(data) do
    if type(conn) == "table" and conn.url and conn.type then
      conn.id = conn.id or ("environment_source_" .. self.var .. "_" .. i)
      table.insert(conns, conn)
    end
  end

  return conns
end

---@divider -

---Built-In Memory Source.
---Loads connections from lua table.
---@class MemorySource: Source
---@field private conns ConnectionParams[]
---@field private display_name string
sources.MemorySource = {}

---@param conns ConnectionParams[] list of connections
---@param name? string optional display name
---@return Source
function sources.MemorySource:new(conns, name)
  name = name or "memory"

  local parsed = {}
  for i, conn in pairs(conns or {}) do
    if type(conn) == "table" and conn.url and conn.type then
      conn.id = "memory_source_" .. name .. i
      table.insert(parsed, conn)
    end
  end

  local o = {
    conns = parsed,
    display_name = name,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@package
---@return string
function sources.MemorySource:name()
  return self.display_name
end

---@package
---@return ConnectionParams[]
function sources.MemorySource:load()
  return self.conns
end

return sources
