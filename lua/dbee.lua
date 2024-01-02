local entry = require("dbee.entry")
local install = require("dbee.install")

---@toc dbee.ref.contents

---@mod dbee.ref Dbee Reference
---@brief [[
---Database Client for NeoVim.
---@brief ]]

local dbee = {
  api = require("dbee.api"),
}

---Setup function.
---Needs to be called before calling any other function.
---@param cfg? Config
function dbee.setup(cfg)
  entry.setup(cfg)
end

---Toggle dbee UI.
function dbee.toggle()
  entry.toggle_ui()
end

---Open dbee UI.
function dbee.open()
  entry.open_ui()
end

---Close dbee UI.
function dbee.close()
  entry.close_ui()
end

---Check if dbee UI is open or not.
---@return boolean
function dbee.is_open()
  return entry.is_ui_open()
end

---Check if dbee core has been loaded.
---@return boolean
function dbee.is_core_loaded()
  return entry.is_core_loaded()
end

---Check if dbee UI has been loaded.
---@return boolean
function dbee.is_ui_loaded()
  return entry.is_ui_loaded()
end

---Execute a query on current connection.
---Convenience wrapper around some api functions that executes a query on
---current connection and pipes the output to result UI.
---@param query string
function dbee.execute(query)
  local handler = entry.get_handler()
  local result = entry.get_tiles().result

  local conn = handler:get_current_connection()
  if not conn then
    error("no connection currently selected")
  end

  local call = handler:connection_execute(conn.id, query)
  result:set_call(call)

  entry.open_ui()
end

---Store currently displayed result.
---Convenience wrapper around some api functions.
---@param format string format of the output -> "csv"|"json"|"table"
---@param output string where to pipe the results -> "file"|"yank"|"buffer"
---@param opts { from: integer, to: integer, extra_arg: any }
function dbee.store(format, output, opts)
  local result = entry.get_tiles().result
  local handler = entry.get_handler()

  local call = result:get_call()
  if not call then
    error("no current call to store")
  end

  handler:call_store_result(call.id, format, output, opts)
end

---Supported install commands.
---@alias install_command
---| '"wget"'
---| '"curl"'
---| '"bitsadmin"'
---| '"go"'
---| '"cgo"'

---Install dbee backend binary.
---@param command? install_command Preffered install command
---@see install_command
function dbee.install(command)
  install.exec(command)
end

return dbee
