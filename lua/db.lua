local Connection = require("db.connection")
local Drawer = require("db.drawer")
local UI = require("db.ui")
local M = {}

---@alias grid { header: string[], rows: string[][] }

M.data = {}

function M.setup()
  local ui_drawer = UI:new { win_cmd = "to 40vsplit" }
  local ui_result = UI:new { win_cmd = "bo 15split" }

  local connections = {}
  for _, d in ipairs(M.data) do
    table.insert(connections, Connection:new { name = d.name, type = d.type, url = d.url, ui = ui_result })
  end

  M.drawer = Drawer:new {
    connections = connections,
    ui = ui_drawer,
  }
end

function M.open_ui()
  M.drawer:render()
end

function M.close_ui()
  M.drawer:close()
end

return M
