local DrawerUI = require("dbee.ui.drawer")
local EditorUI = require("dbee.ui.editor")
local ResultUI = require("dbee.ui.result")
local CallLogUI = require("dbee.ui.call_log")
local Handler = require("dbee.handler")
local install = require("dbee.install")
local config = require("dbee.config")
local register = require("dbee.entry.__register")
local utils = require("dbee.utils")

-- public and private module objects
local M = {}
local m = {}

-- is ui open?
m.ui_opened = false
-- is core set up?
m.core_loaded = false
-- is ui set up?
m.ui_loaded = false
-- was setup function called?
m.setup_called = false
---@type Config
m.config = {}

---TODO: remove
---@param cfg Config
local function check_old_config(cfg)
  local deprecated_opts = {}
  if cfg.page_size then
    table.insert(deprecated_opts, "config.page_size")
  end

  if cfg.progress_bar then
    table.insert(deprecated_opts, "config.progress_bar")
  end

  if cfg.ui then
    table.insert(deprecated_opts, "config.ui")
  end

  if #deprecated_opts > 0 then
    utils.log(
      "warn",
      "deprecated setup options detected: ["
        .. table.concat(deprecated_opts, ", ")
        .. "]\n see: github.com/kndndrj/nvim-dbee/issues/19 for breaking changes."
    )
  end
end

local function setup_handler()
  if m.core_loaded then
    return
  end
  check_old_config(m.config)

  if not m.setup_called then
    error("setup() has not been called yet")
  end

  -- register remote plugin
  register()

  -- add install binary to path
  vim.env.PATH = install.dir() .. ":" .. vim.env.PATH

  m.handler = Handler:new(m.config.sources)
  m.handler:add_helpers(m.config.extra_helpers)

  m.core_loaded = true
end

local function setup_ui()
  if m.ui_loaded then
    return true
  end

  setup_handler()

  local switch = function(bufnr)
    m.editor:set_buf(bufnr)
  end

  m.result = ResultUI:new(m.handler, M.close_ui, switch, m.config.result)
  m.call_log = CallLogUI:new(m.handler, m.result, M.close_ui, switch, m.config.call_log)
  m.editor = EditorUI:new(m.handler, m.result, M.close_ui, m.config.editor)
  m.drawer = DrawerUI:new(m.handler, m.editor, m.result, M.close_ui, switch, m.config.drawer)

  m.ui_loaded = true
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

---@return boolean
function M.is_ui_open()
  return m.ui_opened
end

---@return boolean
function M.is_core_loaded()
  return m.core_loaded
end

---@return boolean
function M.is_ui_loaded()
  return m.core_loaded
end

function M.toggle_ui()
  if m.ui_opened then
    M.close_ui()
  else
    M.open_ui()
  end
end

function M.open_ui()
  setup_ui()

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

---@return Handler
function M.get_handler()
  setup_handler()

  return m.handler
end

---@return layout_uis
function M.get_ui()
  setup_ui()

  return {
    drawer = m.drawer,
    result = m.result,
    editor = m.editor,
    call_log = m.call_log,
  }
end

---@return Config
function M.get_config()
  if not m.setup_called then
    error("setup() has not been called yet")
  end

  return m.config
end

return M
