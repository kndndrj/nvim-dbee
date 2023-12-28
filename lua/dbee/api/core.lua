local entry = require("dbee.entry")

local M = {}

---@param event handler_event_name
---@param listener handler_event_listener
function M.register_event_listener(event, listener)
  entry.get_handler():register_event_listener(event, listener)
end

-- add new source and load connections from it
---@param source Source
function M.add_source(source)
  entry.get_handler():add_source(source)
end

---@return Source[]
function M.get_sources()
  return entry.get_handler():get_sources()
end

---@param id source_id
function M.source_reload(id)
  entry.get_handler():source_reload(id)
end

---@param id source_id
---@param details connection_details[]
function M.source_add_connections(id, details)
  entry.get_handler():source_add_connections(id, details)
end

---@param id source_id
---@param details connection_details[]
function M.source_remove_connections(id, details)
  entry.get_handler():source_remove_connections(id, details)
end

---@param id source_id
---@return connection_details[]
function M.source_get_connections(id)
  return entry.get_handler():source_get_connections(id)
end

---@param helpers table<string, table_helpers> extra helpers per type
function M.add_helpers(helpers)
  entry.get_handler():add_helpers(helpers)
end

---@param id conn_id
---@param opts helper_opts
---@return table_helpers helpers list of table helpers
function M.connection_get_helpers(id, opts)
  return entry.get_handler():connection_get_helpers(id, opts)
end

---@return connection_details?
function M.get_current_connection()
  return entry.get_handler():get_current_connection()
end

---@param id conn_id
function M.set_current_connection(id)
  entry.get_handler():set_current_connection(id)
end

---@param id conn_id
---@param query string
---@return call_details
function M.connection_execute(id, query)
  return entry.get_handler():connection_execute(id, query)
end

---@param id conn_id
---@return DBStructure[]
function M.connection_get_structure(id)
  return entry.get_handler():connection_get_structure(id)
end

---@param id conn_id
---@return connection_details?
function M.connection_get_params(id)
  return entry.get_handler():connection_get_params(id)
end

---@param id conn_id
---@return string current_db
---@return string[] available_dbs
function M.connection_list_databases(id)
  return entry.get_handler():connection_list_databases(id)
end

---@param id conn_id
---@param database string
function M.connection_select_database(id, database)
  entry.get_handler():connection_select_database(id, database)
end

---@param id conn_id
---@return call_details[]
function M.connection_get_calls(id)
  return entry.get_handler():connection_get_calls(id)
end

---@param id call_id
function M.call_cancel(id)
  entry.get_handler():call_cancel(id)
end

---@param id call_id
---@param bufnr integer
---@param from integer
---@param to integer
---@return integer # total number of rows
function M.call_display_result(id, bufnr, from, to)
  return entry.get_handler():call_display_result(id, bufnr, from, to)
end

---@param id call_id
---@param format "csv"|"json"|"table" format of the output
---@param output "file"|"yank"|"buffer" where to pipe the results
---@param opts { from: integer, to: integer, extra_arg: any }
function M.call_store_result(id, format, output, opts)
  entry.get_handler():call_store_result(id, format, output, opts)
end

return M
