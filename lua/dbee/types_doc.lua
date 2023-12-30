---@meta

---@brief [[
--- Overview of public types used in API.
---
--- Please note that type aliases like "call_id" are not expanded in docs.
---@brief ]]

---@tag dbee.types

-- Helpers
---@alias table_helpers table<string, string>

---@class HelperOpts @Options for expanding helper queries.
---@field table string
---@field schema string
---@field materialization "table"|"view"

---@alias call_id string
---@alias call_state "unknown"|"executing"|"executing_failed"|"retrieving"|"retrieving_failed"|"archived"|"archive_failed"|"canceled"

---@class CallDetails @Details and stats of a single call to database.
---@field id call_id
---@field time_taken_us integer: duration (time period) in microseconds
---@field query string
---@field state call_state
---@field timestamp_us integer: time in microseconds

---@alias connection_id string

---@class ConnectionParams @Parameters of a connection
---@field id connection_id
---@field name string
---@field type string
---@field url string

---@class DBStructure @Structure of database.
---@field name string: display name
---@field type ""|"table"|"history"|"database_switch"|"view": type of layout -> this infers action
---@field schema string?: parent schema
---@field children DBStructure[]?: child layout nodes

-- Events
---@alias core_event_name "call_state_changed"|"current_connection_changed"
---@alias event_listener fun(data: any)

---@alias editor_event_name "note_state_changed"|"note_removed"|"note_created"|"current_note_changed"
