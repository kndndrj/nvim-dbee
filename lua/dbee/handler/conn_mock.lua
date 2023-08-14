-- this package is intended to be used as a dummy connection in case no connections are available
local MockConn = {}

---@param params? connection_details
---@return Conn
function MockConn:new(params)
  local defaults = {
    id = "mocked_connection_id",
    name = "mocked connection",
    type = "",
    url = "...",
    page_size = 444,
  }
  params = params or defaults
  -- class object
  local o = {
    id = params.id or defaults.id,
    name = params.name or defaults.name,
    type = params.type or defaults.type,
    url = params.url or defaults.url,
    page_size = params.page_size or defaults.page_size,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

function MockConn:close() end

---@return connection_details
function MockConn:details()
  return {
    id = self.id,
    name = self.name,
    url = self.url,
    type = self.type,
    page_size = self.page_size,
  }
end

---@return connection_details
function MockConn:original_details()
  return self:details()
end

function MockConn:execute(query, cb)
  cb = cb or function() end
  print("trying to execute query on a mocked connection: " .. query)
  cb()
end

function MockConn:history(history_id, cb)
  cb = cb or function() end
  print("trying to get history on a mocked connection: " .. history_id)
  cb()
end

function MockConn:page_next()
  self.on_exec()
end

function MockConn:page_prev()
  self.on_exec()
end

function MockConn:store(format, output, opts)
  print("trying to store on a mocked connection: ", format, output, tostring(opts))
end

function MockConn:layout()
  return {
    id = self.id,
    name = self.name,
    type = "",
  }
end

return MockConn
