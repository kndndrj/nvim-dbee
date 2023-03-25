local Connection = require("db.connection")
local Drawer = require("db.drawer")
local UI = require("db.ui")
local M = {}

---@alias grid { header: string[], rows: string[][] }

---@alias setup_opts { connections: { name: string, type: string, url: string }, lazy: boolean }

-- is the plugin loaded?
local loaded = false
---@type setup_opts
local setup_opts = {}

local function lazy_setup()
  local opts = setup_opts

  local ui_drawer = UI:new { win_cmd = "to 40vsplit" }
  local ui_result = UI:new { win_cmd = "bo 15split" }

  local connections = {}
  for _, d in ipairs(opts.connections) do
    table.insert(connections, Connection:new { name = d.name, type = d.type, url = d.url, ui = ui_result })
  end

  M.drawer = Drawer:new {
    connections = connections,
    ui = ui_drawer,
  }

  loaded = true
end

---@param opts setup_opts
function M.setup(opts)
  setup_opts = opts or {}
  if setup_opts.lazy then
    return
  end
  lazy_setup()
end

function M.open_ui()
  if not loaded then
    lazy_setup()
  end
  M.drawer:render()
end

function M.close_ui()
  if not loaded then
    lazy_setup()
  end
  M.drawer:close()
end

return M
