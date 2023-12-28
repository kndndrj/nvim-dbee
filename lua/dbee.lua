local entry = require("dbee.entry")
local install = require("dbee.install")

local M = {
  api = require("dbee.api"),
}

---@param cfg? Config
function M.setup(cfg)
  entry.setup(cfg)
end

function M.toggle()
  entry.toggle_ui()
end

function M.open()
  entry.open_ui()
end

function M.close()
  entry.close_ui()
end

---@param command? install_command preffered command
function M.install(command)
  install.exec(command)
end

return M
