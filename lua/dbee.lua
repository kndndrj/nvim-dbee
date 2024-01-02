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
