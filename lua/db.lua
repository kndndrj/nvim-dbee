local Connection = require("db.connection")
local Drawer = require("db.ui.drawer")
local M = {}

---@alias grid { header: string[], rows: string[][] }

local drawer

M.data = {}

M.connections = {}

function M.setup()
  for _, d in ipairs(M.data) do
    table.insert(M.connections, Connection:new { name = d.name, type = d.type, url = d.url })
  end
end

function M.open_ui()
  if not drawer then
    drawer = Drawer:new { connections = M.connections }
  end
  drawer:show()
end

function M.close_ui()
  drawer:hide()
end

return M
