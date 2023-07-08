local utils = require("dbee.utils")

-- Lookup is a "dumb" storage for sources and connections
-- and their relations
---@class Lookup
---@field private connections table<conn_id, Conn>
---@field private sources table<source_id, { source: Source, connections: conn_id[] }>
---@field private conn_lookup table<conn_id, source_id>
---@field private active_connection conn_id
local Lookup = {}

---@return Lookup
function Lookup:new()
  local o = {
    connections = {},
    sources = {},
    conn_lookup = {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param source Source
function Lookup:add_source(source)
  local id = source:name()

  if self.sources[id] then
    error("source already exists: " .. id)
  end

  self.sources[id] = {
    source = source,
    connections = {},
    active_connection = "",
  }
end

---@param id source_id
function Lookup:remove_source(id)
  if not self.sources[id] then
    return
  end

  for _, conn_id in ipairs(self.sources[id].connections) do
    local conn = self.connections[conn_id]
    if conn then
      pcall(conn.close, conn)
    end
    self.connections[conn_id] = nil
    self.conn_lookup[conn_id] = nil
  end

  self.sources[id] = nil
end

---@param connection Conn
---@param source_id source_id
function Lookup:add_connection(connection, source_id)
  if not source_id then
    error("source_id not set")
  end

  local id = connection:details().id

  local old = self.connections[id]
  if old then
    pcall(old.close, old)
  end

  self.connections[id] = connection
  self.conn_lookup[id] = source_id
  table.insert(self.sources[source_id].connections, id)

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
  local source_id = self.conn_lookup[id]
  if self.sources[source_id] and self.sources[source_id].connections then
    for i, c_id in ipairs(self.sources[source_id].connections) do
      if id == c_id then
        table.remove(self.sources[source_id].connections, i)
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

---@param source_id? source_id # id of the source or all
---@return Conn[] connections
function Lookup:get_connections(source_id)
  local conns = {}
  -- get connections of a source
  -- or get all connections
  if source_id then
    local l = self.sources[source_id]
    if not l then
      error("unknown source: " .. source_id)
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

---@return Conn|nil
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
---@return Source[] sources
function Lookup:get_sources(conn_id)
  local sources = {}
  -- get source of a connection
  -- or get all sources
  if conn_id then
    local l_id = self.conn_lookup[conn_id]
    if not l_id then
      return {}
    end
    table.insert(sources, self.sources[l_id].source)
  else
    for _, l in pairs(self.sources) do
      table.insert(sources, l.source)
    end
  end

  -- sort keys
  table.sort(sources, function(k1, k2)
    return k1:name() < k2:name()
  end)

  return sources
end

---@param id source_id
---@return Source|nil source
function Lookup:get_source(id)
  local l = self.sources[id]
  if not l then
    return
  end
  return l.source
end

return Lookup
