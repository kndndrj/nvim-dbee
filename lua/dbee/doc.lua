---@mod dbee.ref.types Types
---@brief [[
---Overview of types used in DBee API.
---@brief ]]

---@divider -
---@tag dbee.ref.types.table
---@brief [[
---Table related types
---@brief ]]

---Table column
---@class Column
---@field name string name of the column
---@field type string database type of the column

---Table Materialization.
---@alias materialization
---| '"table"'
---| '"view"'

---Options for gathering table specific info.
---@class TableOpts
---@field table string
---@field schema string
---@field materialization materialization

---Table helpers queries by name.
---@alias table_helpers table<string, string>

---@divider -
---@tag dbee.ref.types.call
---@brief [[
---Call related types.
---@brief ]]

---ID of a call.
---@alias call_id string

---State of a call.
---@alias call_state
---| '"unknown"'
---| '"executing"'
---| '"executing_failed"'
---| '"retrieving"'
---| '"retrieving_failed"'
---| '"archived"'
---| '"archive_failed"'
---| '"canceled"'

---Details and stats of a single call to database.
---@class CallDetails
---@field id call_id
---@field time_taken_us integer duration (time period) in microseconds
---@field query string
---@field state call_state
---@field timestamp_us integer time in microseconds
---@field error? string error message in case of error

---@divider -
---@tag dbee.ref.types.connection
---@brief [[
---Connection related types.
---@brief ]]

---ID of a connection.
---@alias connection_id string

---Parameters of a connection.
---@class ConnectionParams
---@field id connection_id
---@field name string
---@field type string
---@field url string

---@divider -
---@tag dbee.ref.types.structure
---@brief [[
---Database structure related types.
---@brief ]]

---Type of node in database structure.
---@alias structure_type
---| '""'
---| '"table"'
---| '"history"'
---| '"database_switch"'
---| '"view"'

---Structure of database.
---@class DBStructure
---@field name string display name
---@field type structure_type type of node in structure
---@field schema string? parent schema
---@field children DBStructure[]? child layout nodes

---@divider -
---@tag dbee.ref.types.events
---@brief [[
---Event related types.
---@brief ]]

---Avaliable core events.
---@alias core_event_name
---| '"call_state_changed"'
---| '"current_connection_changed"'
---| '"current_database_changed"'

---Available editor events.
---@alias editor_event_name
---| '"note_state_changed"'
---| '"note_removed"'
---| '"note_created"'
---| '"current_note_changed"'

---Event handler function.
---@alias event_listener fun(data: any)

local M = {}
return M
