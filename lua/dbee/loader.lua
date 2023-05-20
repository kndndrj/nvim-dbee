local utils = require("dbee.utils")

local M = {}

---@alias loader_id string

---@class Loader
---@field name fun(self: Loader):string function to return the name of the loader
---@field load fun(self: Loader):connection_details[] function to load connections from external source
---@field save? fun(self: Loader, conns: connection_details[], action: "add"|"delete") function to save connections to external source (optional)
---@field source? fun(self: Loader):string function which returns a source file to edit (optional)

--- File loader
---@class FileLoader: Loader
---@field private path string path to file
M.FileLoader = {}

--- Loads connections from json file
---@param path string path to file
---@return Loader
function M.FileLoader:new(path)
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

---@return string
function M.FileLoader:name()
  return vim.fs.basename(self.path)
end

---@return connection_details[]
function M.FileLoader:load()
  local path = self.path

  ---@type connection_details[]
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
    utils.log("warn", 'Could not parse json file: "' .. path .. '".', "loader")
    return {}
  end

  for _, conn in pairs(data) do
    if type(conn) == "table" then
      table.insert(conns, conn)
    end
  end

  return conns
end

-- saves connection to file
---@param conns connection_details[]
---@param action "add"|"delete"
function M.FileLoader:save(conns, action)
  local path = self.path

  if not conns or vim.tbl_isempty(conns) then
    return
  end

  -- read from file
  local existing = self:load()

  ---@type connection_details[]
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
    utils.log("error", "Could not convert connection list to json", "loader")
    return
  end

  -- overwrite file
  local file = assert(io.open(path, "w+"), "could not open file")
  file:write(json)
  file:close()
end

---@return string
function M.FileLoader:source()
  return self.path
end

--- Environment loader
---@class EnvLoader: Loader
---@field private var string path to file
M.EnvLoader = {}

--- Loads connections from json file
---@param var string env var to load from
---@return Loader
function M.EnvLoader:new(var)
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

---@return string
function M.EnvLoader:name()
  return self.var
end

---@return connection_details[]
function M.EnvLoader:load()
  ---@type connection_details[]
  local conns = {}

  local raw = os.getenv(self.var)
  if not raw then
    return {}
  end

  local ok, data = pcall(vim.fn.json_decode, raw)
  if not ok then
    utils.log("warn", 'Could not parse connections from env: "' .. self.var .. '".', "loader")
    return {}
  end

  for _, conn in pairs(data) do
    if type(conn) == "table" and conn.url and conn.type then
      table.insert(conns, conn)
    end
  end

  return conns
end

--- Environment loader
---@class MemoryLoader: Loader
---@field conns connection_details[]
M.MemoryLoader = {}

--- Loads connections from json file
---@param conns connection_details[]
---@return Loader
function M.MemoryLoader:new(conns)
  local o = {
    conns = conns or {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@return string
function M.MemoryLoader:name()
  return "memory"
end

---@return connection_details[]
function M.MemoryLoader:load()
  return self.conns
end

return M
