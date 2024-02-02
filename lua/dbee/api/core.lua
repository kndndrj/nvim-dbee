---@mod dbee.ref.api.core Dbee Core API
---@brief [[
---This module contains functions to operate on the backend side.
---
---Access it like this:
--->
---require("dbee").api.core.func()
---<
---@brief ]]

local entry = require("dbee.entry")

local core = {}

---Registers an event handler for core events.
---@param event core_event_name
---@param listener event_listener
function core.register_event_listener(event, listener)
  entry.get_handler():register_event_listener(event, listener)
end

---Add new source and load connections from it.
---@param source Source
function core.add_source(source)
  entry.get_handler():add_source(source)
end

---Get a list of registered sources.
---@return Source[]
function core.get_sources()
  return entry.get_handler():get_sources()
end

---Reload a source by id.
---@param id source_id
function core.source_reload(id)
  entry.get_handler():source_reload(id)
end

---Reload all sources.
function core.reload_sources()
  entry.get_handler():reload_sources()
end

---Add connections to the source.
---@param id source_id
---@param details ConnectionParams[]
function core.source_add_connections(id, details)
  entry.get_handler():source_add_connections(id, details)
end

---Remove a connection from the source.
---If source can edit connections, it also removes the
---connection permanently.
---@param id source_id
---@param details ConnectionParams[]
function core.source_remove_connections(id, details)
  entry.get_handler():source_remove_connections(id, details)
end

--- Get a list of connections from source.
---@param id source_id
---@return ConnectionParams[]
function core.source_get_connections(id)
  return entry.get_handler():source_get_connections(id)
end

---Register helper queries per database type.
---every helper value is a go-template with values set for
---"Table", "Schema" and "Materialization".
---@param helpers table<string, table<string, string>> extra helpers per type
---@see table_helpers
---@usage lua [[
---{
---  ["postgres"] = {
---    ["List All"] = "SELECT * FROM {{ .Table }}",
---  }
---}
---@usage ]]
function core.add_helpers(helpers)
  entry.get_handler():add_helpers(helpers)
end

---Get helper queries for a specific connection.
---@param id connection_id
---@param opts TableOpts
---@return table<string, string> _ list of table helpers
---@see table_helpers
function core.connection_get_helpers(id, opts)
  return entry.get_handler():connection_get_helpers(id, opts)
end

---Get the currently active connection.
---@return ConnectionParams|nil
function core.get_current_connection()
  return entry.get_handler():get_current_connection()
end

---Set a currently active connection.
---@param id connection_id
function core.set_current_connection(id)
  entry.get_handler():set_current_connection(id)
end

---Execute a query on a connection.
---@param id connection_id
---@param query string
---@return CallDetails
function core.connection_execute(id, query)
  return entry.get_handler():connection_execute(id, query)
end

---Get database structure of a connection.
---@param id connection_id
---@return DBStructure[]
function core.connection_get_structure(id)
  return entry.get_handler():connection_get_structure(id)
end

---Get columns of a table
---@param id connection_id
---@param opts { table: string, schema: string, materialization: string }
---@return Column[]
function core.connection_get_columns(id, opts)
  return entry.get_handler():connection_get_columns(id, opts)
end

---Get parameters that define the connection.
---@param id connection_id
---@return ConnectionParams|nil
function core.connection_get_params(id)
  return entry.get_handler():connection_get_params(id)
end

---List databases of a connection.
---Some databases might not support this - in that case, a call to this
---function returns an error.
---@param id connection_id
---@return string currently selected database
---@return string[] other available databases
function core.connection_list_databases(id)
  return entry.get_handler():connection_list_databases(id)
end

---Select an active database of a connection.
---Some databases might not support this - in that case, a call to this
---function returns an error.
---@param id connection_id
---@param database string
function core.connection_select_database(id, database)
  entry.get_handler():connection_select_database(id, database)
end

---Get a list of past calls of a connection.
---@param id connection_id
---@return CallDetails[]
function core.connection_get_calls(id)
  return entry.get_handler():connection_get_calls(id)
end

---Cancel call execution.
---If call is finished, nothing happens.
---@param id call_id
function core.call_cancel(id)
  entry.get_handler():call_cancel(id)
end

---Display the result of a call formatted as a table in a buffer.
---@param id call_id id of the call
---@param bufnr integer
---@param from integer
---@param to integer
---@return integer total number of rows
function core.call_display_result(id, bufnr, from, to)
  return entry.get_handler():call_display_result(id, bufnr, from, to)
end

---Store the result of a call.
---@param id call_id
---@param format string format of the output -> "csv"|"json"|"table"
---@param output string where to pipe the results -> "file"|"yank"|"buffer"
---@param opts { from: integer, to: integer, extra_arg: any }
function core.call_store_result(id, format, output, opts)
  entry.get_handler():call_store_result(id, format, output, opts)
end

return core
