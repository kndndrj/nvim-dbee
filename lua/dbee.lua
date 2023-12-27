local DrawerTile = require("dbee.tiles.drawer")
local EditorTile = require("dbee.tiles.editor")
local ResultTile = require("dbee.tiles.result")
local CallLogTile = require("dbee.tiles.call_log")
local Handler = require("dbee.handler")
local install = require("dbee.install")
local utils = require("dbee.utils")
local config = require("dbee.config")
local register = require("dbee.__register")

-- TODO:
-- revisit the whole api:
-- decouple api from UI -- don't let people control UI with api.
-- API should be just handler exposed (maybe notes as well).

-- public and private module objects
local M = {}
local m = {}

-- is the ui open?
m.ui_opened = false
-- is core set up?
m.core_loaded = false
-- is ui set up?
m.ui_loaded = false
-- was setup function called?
m.setup_called = false
---@type Config
m.config = {}

local function setup_core()
  if not m.setup_called then
    error("setup() has not been called yet")
  end

  -- register remote plugin
  register()

  -- add install binary to path
  vim.env.PATH = install.path() .. ":" .. vim.env.PATH

  m.handler = Handler:new(m.config.sources)
  m.handler:add_helpers(m.config.extra_helpers)
end

---@return boolean ok was setup successful?
local function pcall_setup_core()
  if m.core_loaded then
    return true
  end

  local ok, mes = pcall(setup_core)
  if not ok then
    utils.log("error", tostring(mes), "setup core")
    return false
  end

  m.core_loaded = true
  return true
end

local function setup_ui()
  if not pcall_setup_core() then
    return
  end

  local switch = function(bufnr)
    m.editor:set_buf(bufnr)
  end

  m.result = ResultTile:new(m.handler, M.close_ui, switch, m.config.result)
  m.call_log = CallLogTile:new(m.handler, m.result, M.close_ui, switch, m.config.call_log)
  m.editor = EditorTile:new(m.handler, m.result, M.close_ui, m.config.editor)
  m.drawer = DrawerTile:new(m.handler, m.editor, m.result, M.close_ui, switch, m.config.drawer)
end

---@return boolean ok was setup successful?
local function pcall_setup_ui()
  if m.ui_loaded then
    return true
  end

  local ok, mes = pcall(setup_ui)
  if not ok then
    utils.log("error", tostring(mes), "setup ui")
    return false
  end

  m.ui_loaded = true
  return true
end

---@param cfg? Config
function M.setup(cfg)
  cfg = cfg or {}
  ---@type Config
  local opts = vim.tbl_deep_extend("force", config.default, cfg)
  -- validate config
  config.validate(opts)

  m.config = opts

  m.setup_called = true
end

function M.toggle_ui()
  if m.ui_opened then
    M.close_ui()
  else
    M.open_ui()
  end
end

function M.open_ui()
  if not pcall_setup_ui() then
    return
  end
  if m.ui_opened then
    return
  end

  m.config.window_layout:open {
    drawer = m.drawer,
    result = m.result,
    editor = m.editor,
    call_log = m.call_log,
  }

  m.ui_opened = true
end

function M.close_ui()
  if not m.ui_opened then
    return
  end

  m.config.window_layout:close()
  m.ui_opened = false
end

---@param command? install_command preffered command
function M.install(command)
  install.exec(command)
end

-- experimental and subject to change!
M.api = m

return M
