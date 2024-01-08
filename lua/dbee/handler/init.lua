local event_bus = require("dbee.handler.__events")
local utils = require("dbee.utils")

-- Handler is an aggregator of connections
---@class Handler
---@field private sources table<source_id, Source>
---@field private source_conn_lookup table<string, connection_id[]>
local Handler = {}

---@param sources? Source[]
---@return Handler
function Handler:new(sources)
  -- class object
  local o = {
    sources = {},
    source_conn_lookup = {},
  }
  setmetatable(o, self)
  self.__index = self

  -- initialize the sources
  sources = sources or {}
  for _, source in ipairs(sources) do
    local ok, mes = pcall(o.add_source, o, source)
    if not ok then
      utils.log("error", "failed registering source: " .. mes, "core")
    end
  end

  return o
end

---@param event core_event_name
---@param listener event_listener
function Handler:register_event_listener(event, listener)
  event_bus.register(event, listener)
end

-- add new source and load connections from it
---@param source Source
function Handler:add_source(source)
  local id = source:name()

  -- keep the old source if present
  self.sources[id] = self.sources[id] or source

  self:source_reload(id)
end

---@return Source[]
function Handler:get_sources()
  local sources = vim.tbl_values(self.sources)
  table.sort(sources, function(k1, k2)
    return k1:name() < k2:name()
  end)
  return sources
end

---@param id source_id
function Handler:source_reload(id)
  local source = self.sources[id]
  if not source then
    return
  end

  -- add new connections
  self.source_conn_lookup[id] = {}
  for _, spec in ipairs(source:load()) do
    spec.id = spec.id or spec.type .. spec.name

    local conn_id = vim.fn.DbeeCreateConnection(spec)
    table.insert(self.source_conn_lookup[id], conn_id)
  end
end

---@param id source_id
---@param details ConnectionParams[]
function Handler:source_add_connections(id, details)
  if not details then
    return
  end

  local source = self.sources[id]
  if not source then
    return
  end

  if type(source.save) == "function" then
    source:save(details, "add")
  end

  self:source_reload(id)
end

---@param id source_id
---@param details ConnectionParams[]
function Handler:source_remove_connections(id, details)
  if not details then
    return
  end

  local source = self.sources[id]
  if not source then
    return
  end

  if type(source.save) == "function" then
    source:save(details, "delete")
  end

  self:source_reload(id)
end

---@param id source_id
---@return ConnectionParams[]
function Handler:source_get_connections(id)
  local conn_ids = self.source_conn_lookup[id] or {}
  if #conn_ids < 1 then
    return {}
  end

  ---@type ConnectionParams[]?
  local ret = vim.fn.DbeeGetConnections(conn_ids)
  if not ret or ret == vim.NIL then
    return {}
  end

  table.sort(ret, function(k1, k2)
    return k1.name < k2.name
  end)

  return ret
end

---@param helpers table<string, table_helpers> extra helpers per type
function Handler:add_helpers(helpers)
  for type, help in pairs(helpers) do
    vim.fn.DbeeAddHelpers(type, help)
  end
end

---@param id connection_id
---@param opts TableOpts
---@return table_helpers helpers list of table helpers
function Handler:connection_get_helpers(id, opts)
  local helpers = vim.fn.DbeeConnectionGetHelpers(id, {
    table = opts.table,
    schema = opts.schema,
    materialization = opts.materialization,
  })
  if not helpers or helpers == vim.NIL then
    return {}
  end

  return helpers
end

---@return ConnectionParams?
function Handler:get_current_connection()
  local ok, ret = pcall(vim.fn.DbeeGetCurrentConnection)
  if not ok or ret == vim.NIL then
    return
  end
  return ret
end

---@param id connection_id
function Handler:set_current_connection(id)
  vim.fn.DbeeSetCurrentConnection(id)
end

---@param id connection_id
---@param query string
---@return CallDetails
function Handler:connection_execute(id, query)
  return vim.fn.DbeeConnectionExecute(id, query)
end

---@param id connection_id
---@return DBStructure[]
function Handler:connection_get_structure(id)
  local ret = vim.fn.DbeeConnectionGetStructure(id)
  if not ret or ret == vim.NIL then
    return {}
  end
  return ret
end

---@param id connection_id
---@param opts { table: string, schema: string, materialization: string }
---@return Column[]
function Handler:connection_get_columns(id, opts)
  local out = vim.fn.DbeeConnectionGetColumns(id, opts)
  if not out or out == vim.NIL then
    return {}
  end

  return out
end

---@param id connection_id
---@return ConnectionParams?
function Handler:connection_get_params(id)
  local ret = vim.fn.DbeeConnectionGetParams(id)
  if not ret or ret == vim.NIL then
    return
  end
  return ret
end

---@param id connection_id
---@return string current_db
---@return string[] available_dbs
function Handler:connection_list_databases(id)
  local ret = vim.fn.DbeeConnectionListDatabases(id)
  if not ret or ret == vim.NIL then
    return "", {}
  end

  return unpack(ret)
end

---@param id connection_id
---@param database string
function Handler:connection_select_database(id, database)
  vim.fn.DbeeConnectionSelectDatabase(id, database)
end

---@param id connection_id
---@return CallDetails[]
function Handler:connection_get_calls(id)
  local ret = vim.fn.DbeeConnectionGetCalls(id)
  if not ret or ret == vim.NIL then
    return {}
  end
  return ret
end

---@param id call_id
function Handler:call_cancel(id)
  vim.fn.DbeeCallCancel(id)
end

---@param id call_id
---@param bufnr integer
---@param from integer
---@param to integer
---@return integer # total number of rows
function Handler:call_display_result(id, bufnr, from, to)
  local length = vim.fn.DbeeCallDisplayResult(id, { buffer = bufnr, from = from, to = to })
  if not length or length == vim.NIL then
    return 0
  end
  return length
end

---@alias store_format "csv"|"json"|"table"
---@alias store_output "file"|"yank"|"buffer"

---@param id call_id
---@param format store_format format of the output
---@param output store_output where to pipe the results
---@param opts { from: integer, to: integer, extra_arg: any }
function Handler:call_store_result(id, format, output, opts)
  opts = opts or {}

  local from = opts.from or 0
  local to = opts.to or -1

  vim.fn.DbeeCallStoreResult(id, format, output, {
    from = from,
    to = to,
    extra_arg = opts.extra_arg,
  })
end

return Handler
