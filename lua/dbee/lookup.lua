local utils = require("dbee.utils")

-- Lookup is a "dumb" storage for loaders and connections
-- and their relations
---@class Lookup
---@field private connections table<conn_id, Conn>
---@field private loaders table<loader_id, { loader: Loader, connections: conn_id[] }>
---@field private conn_lookup table<conn_id, loader_id>
---@field private active_connection conn_id
local Lookup = {}

---@return Lookup
function Lookup:new()
  local o = {
    connections = {},
    loaders = {},
    conn_lookup = {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param loader Loader
function Lookup:add_loader(loader)
  local id = loader:name()

  if self.loaders[id] then
    error("loader already exists: " .. id)
  end

  self.loaders[id] = {
    loader = loader,
    connections = {},
    active_connection = "",
  }
end

---@param id loader_id
function Lookup:remove_loader(id)
  if not self.loaders[id] then
    return
  end

  for _, conn_id in ipairs(self.loaders[id].connections) do
    local conn = self.connections[conn_id]
    if conn then
      pcall(conn.close, conn)
    end
    self.connections[conn_id] = nil
    self.conn_lookup[conn_id] = nil
  end

  self.loaders[id] = nil
end

---@param connection Conn
---@param loader_id loader_id
function Lookup:add_connection(connection, loader_id)
  if not loader_id then
    error("loader_id not set")
  end

  local id = connection:details().id

  self:remove_connection(id)

  self.connections[id] = connection
  self.conn_lookup[loader_id] = id
  table.insert(self.loaders[loader_id].connections, id)

  self.active_connection = id
end

---@param id conn_id
function Lookup:remove_connection(id)
  local conn = self.connections[id]
  if not conn then
    return
  end

  -- close the connection
  pcall(conn.close, conn)

  -- remove the connection from all lookups
  local loader_id = self.conn_lookup[id]
  if self.loaders[loader_id] and self.loaders[loader_id].connections then
    for i, c_id in ipairs(self.loaders[loader_id].connections) do
      if id == c_id then
        table.remove(self.loaders[loader_id].connections, i)
      end
    end
  end
  self.conn_lookup[id] = nil
  self.connections[id] = nil

  -- set random connection as active
  if self.active_connection == id then
    self.active_connection = utils.random_key(self.connections)
  end
end

---@param loader_id? loader_id # id of the loader or all
---@return Conn[] connections
function Lookup:get_connections(loader_id)
  local conns = {}
  -- get connections of a loader
  -- or get all connections
  if loader_id then
    local l = self.loaders[loader_id]
    if not l then
      error("unknown loader: " .. loader_id)
    end
    for _, c_id in ipairs(l.connections) do
      table.insert(conns, self.connections[c_id])
    end
  else
    for _, conn in pairs(self.connections) do
      table.insert(conns, conn)
    end
  end

  -- sort keys
  table.sort(conns, function(k1, k2)
    return k1:details().name < k2:details().name
  end)

  return conns
end

---@param id conn_id
---@return Conn|nil connection
function Lookup:get_connection(id)
  return self.connections[id]
end

---@return Conn
function Lookup:get_active_connection()
  return self.connections[self.active_connection]
end

---@param id conn_id
function Lookup:set_active_connection(id)
  if self.connections[id] then
    self.active_connection = id
  end
end

---@param conn_id? conn_id id of the connection or all
---@return Loader[] loaders
function Lookup:get_loaders(conn_id)
  local loaders = {}
  -- get loader of a connection
  -- or get all loaders
  if conn_id then
    local l_id = self.conn_lookup[conn_id]
    if not l_id then
      return {}
    end
    table.insert(loaders, self.loaders[l_id].loader)
  else
    for _, l in pairs(self.loaders) do
      table.insert(loaders, l.loader)
    end
  end

  -- sort keys
  table.sort(loaders, function(k1, k2)
    return k1:name() < k2:name()
  end)

  return loaders
end

---@param id loader_id
---@return Loader|nil loader
function Lookup:get_loader(id)
  local l = self.loaders[id]
  if not l then
    return
  end
  return l.loader
end

return Lookup
