local floats = require("dbee.ui.common.floats")
local DrawerUI = require("dbee.ui.drawer")
local EditorUI = require("dbee.ui.editor")
local ResultUI = require("dbee.ui.result")
local CallLogUI = require("dbee.ui.call_log")
local Handler = require("dbee.handler")
local install = require("dbee.install")
local register = require("dbee.api.__register")

-- public and private module objects
local M = {}
local m = {}

-- is core set up?
m.core_loaded = false
-- is ui set up?
m.ui_loaded = false
-- was setup function called?
m.setup_called = false
---@type Config
m.config = {}

local function setup_handler()
  if m.core_loaded then
    return
  end

  if not m.setup_called then
    error("setup() has not been called yet")
  end

  -- register remote plugin
  register()

  -- add install binary to path
  local pathsep = ":"
  if vim.fn.has("win32") == 1 then
    pathsep = ";"
  end
  vim.env.PATH = install.dir() .. pathsep .. vim.env.PATH

  m.handler = Handler:new(m.config.sources)
  m.handler:add_helpers(m.config.extra_helpers)

  -- activate default connection if present
  if m.config.default_connection then
    pcall(m.handler.set_current_connection, m.handler, m.config.default_connection)
  end

  m.core_loaded = true
end

local function setup_ui()
  if m.ui_loaded then
    return
  end

  setup_handler()

  -- configure options for floating windows
  floats.configure(m.config.float_options)

  -- initiate all UI elements
  m.result = ResultUI:new(m.handler, m.config.result)
  m.call_log = CallLogUI:new(m.handler, m.result, m.config.call_log)
  m.editor = EditorUI:new(m.handler, m.result, m.config.editor)
  m.drawer = DrawerUI:new(m.handler, m.editor, m.result, m.config.drawer)

  m.ui_loaded = true
end

---@param cfg Config
function M.setup(cfg)
  if m.setup_called then
    error("setup() can only be called once")
  end
  m.config = cfg

  m.setup_called = true
end

---@return boolean
function M.is_core_loaded()
  return m.core_loaded
end

---@return boolean
function M.is_ui_loaded()
  return m.ui_loaded
end

---@return Handler
function M.handler()
  setup_handler()
  return m.handler
end

---@return EditorUI
function M.editor()
  setup_ui()
  return m.editor
end

---@return CallLogUI
function M.call_log()
  setup_ui()
  return m.call_log
end

---@return DrawerUI
function M.drawer()
  setup_ui()
  return m.drawer
end

---@return ResultUI
function M.result()
  setup_ui()
  return m.result
end

---@return Config
function M.config()
  return m.config
end

return M
