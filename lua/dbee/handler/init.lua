local Helpers = require("dbee.handler.helpers")
local event_bus = require("dbee.handler.__callbacks")

-- structure of the database
---@class DBStructure
---@field name string display name
---@field type ""|"table"|"history"|"database_switch"|"view" type of layout -> this infers action
---@field schema? string parent schema
---@field children? DBStructure[] child layout nodes
---@field pick_items?  string[] pick items

---@alias handler_config { fallback_page_size: integer, progress: progress_config }

-- Handler is an aggregator of connections
---@class Handler
---@field private ui Ui ui for results
---@field private sources table<source_id, Source>
---@field private source_conn_lookup table<string, conn_id[]>
---@field private helpers Helpers query helpers
---@field private opts handler_config
local Handler = {}

---@param ui Ui ui for displaying results
---@param sources? Source[]
---@param opts? handler_config
---@return Handler
function Handler:new(ui, sources, opts)
  if not ui then
    error("no results Ui passed to Handler")
  end

  -- class object
  local o = {
    ui = ui,
    sources = {},
    source_conn_lookup = {},
    helpers = Helpers:new(),
    opts = opts or {},
  }
  setmetatable(o, self)
  self.__index = self

  -- initialize the sources
  sources = sources or {}
  for _, source in ipairs(sources) do
    pcall(o.add_source, o, source)
  end

  return o
end

---@param event event_name
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
    local conn_id = vim.fn.DbeeCreateConnection(spec)
    table.insert(self.source_conn_lookup[id], conn_id)
  end
end

---@param id source_id
---@param details connection_details[]
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
---@param details connection_details[]
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
---@return connection_details[]
function Handler:source_get_connections(id)
  local conn_ids = self.source_conn_lookup[id] or {}
  if #conn_ids < 1 then
    return {}
  end

  ---@type connection_details[]?
  local ret = vim.fn.DbeeGetConnections { ids = conn_ids }
  if not ret or ret == vim.NIL then
    return {}
  end

  table.sort(ret, function(k1, k2)
    return k1.name < k2.name
  end)

  return ret
end

---@param helpers table<string, table_helpers> extra helpers per type
function Handler:helpers_add(helpers)
  self.helpers:add(helpers)
end

---@param type string
---@param vars helper_vars
---@return table_helpers helpers list of table helpers
function Handler:helpers_get(type, vars)
  return self.helpers:get(type, vars)
end

---@return connection_details?
function Handler:get_current_connection()
  local ret = vim.fn.DbeeGetCurrentConnection()
  if ret == vim.NIL then
    return
  end
  return ret
end

---@param id conn_id
function Handler:set_current_connection(id)
  vim.fn.DbeeSetCurrentConnection { id = id }
end

---@param id conn_id
---@param query string
---@return call_details
function Handler:connection_execute(id, query)
  return vim.fn.DbeeConnExecute { id = id, query = query }
end

---@param id conn_id
---@return Layout[]
function Handler:connection_get_structure(id)
  local ret = vim.fn.DbeeConnGetStructure { id = id }
  if not ret or ret == vim.NIL then
    return {}
  end
  return ret
end

---@param id conn_id
---@return connection_details?
function Handler:connection_get_params(id)
  local ret = vim.fn.DbeeConnGetParams { id = id }
  if not ret or ret == vim.NIL then
    return
  end
  return ret
end

---@param id conn_id
---@return string current_db
---@return string[] available_dbs
function Handler:connection_list_databases(id)
  return unpack(vim.fn.DbeeConnListDatabases { id = id })
end

---@param id conn_id
---@param database string
function Handler:connection_select_database(id, database)
  vim.fn.DbeeConnSelectDatabase { id = id, database = database }
end

---@param id conn_id
---@return call_details[]
function Handler:connection_get_calls(id)
  local ret = vim.fn.DbeeConnGetCalls { id = id }
  if not ret or ret == vim.NIL then
    return {}
  end
  return ret
end

---@param id call_id
function Handler:call_cancel(id)
  vim.fn.DbeeCallCancel { id = id }
end

---@param id call_id
---@param bufnr integer
---@param from integer
---@param to integer
---@return integer # total number of rows
function Handler:call_display_result(id, bufnr, from, to)
  return vim.fn.DbeeCallDisplayResult { id = id, buffer = bufnr, from = from, to = to }
end

---@param id call_id
---@param format "csv"|"json"|"table" format of the output
---@param output "file"|"yank"|"buffer" where to pipe the results
---@param opts { from: integer, to: integer, extra_arg: any }
function Handler:call_store_result(id, format, output, opts)
  opts = opts or {}

  local from = opts.from or 0
  local to = opts.to or -1

  local path
  if output == "file" then
    path = opts.extra_arg
  end
  local bufnr
  if output == "buffer" then
    bufnr = opts.extra_arg
  end

  vim.fn.DbeeCallStoreResult {
    id = id,
    format = format,
    output = output,
    from = from,
    to = to,
    path = path,
    bufnr = bufnr,
  }
end

return Handler
