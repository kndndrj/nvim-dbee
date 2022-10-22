local Connection = require("db.connection")
local UI = require"db.ui"
local M = {}

---@alias grid { header: string[], rows: string[][] }

local ui

M.data = {}

M.connections = {}

function M.setup()
  for _, d in ipairs(M.data) do
    table.insert(M.connections, Connection:new { name = d.name, type = d.type, url = d.url })
  end
end

function M.open_ui()
  if not ui then
    ui = UI:new()
  end
  ui:open()
end

function M.close_ui()
  ui:close()
end

return M
