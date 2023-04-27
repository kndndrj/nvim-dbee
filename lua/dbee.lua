local Drawer = require("dbee.drawer")
local Editor = require("dbee.editor")
local Handler = require("dbee.handler")
local install = require("dbee.install")
local utils = require("dbee.utils")
local default_config = require("dbee.config").default
local helpers = require("dbee.helpers")

-- public and private module objects
local M = {}
local m = {}

-- is the ui open?
m.open = false
-- is the plugin loaded?
m.loaded = false
---@type Config
m.config = {}

local function lazy_setup()
  -- add install binary to path
  vim.env.PATH = install.path() .. ":" .. vim.env.PATH

  -- set up modules
  m.handler = Handler:new(m.config.connections, m.config.result)
  m.editor = Editor:new(m.handler, m.config.editor)
  m.drawer = Drawer:new(m.handler, m.editor, m.config.drawer)

  helpers.add(m.config.extra_helpers)
end

---@return boolean ok was setup successful?
local function pcall_lazy_setup()
  if m.loaded then
    return true
  end

  local ok, mes = pcall(lazy_setup)
  if not ok then
    utils.log("error", tostring(mes), "init")
    return false
  end

  m.loaded = true
  return true
end

---@param o Config
function M.setup(o)
  o = o or {}
  ---@type Config
  local opts = vim.tbl_deep_extend("force", default_config, o)
  -- validate config
  vim.validate {
    connections = { opts.connections, "table" },
    lazy = { opts.lazy, "boolean" },
    extra_helpers = { opts.extra_helpers, "table" },
    -- submodules
    result_window_command = { opts.result.window_command, { "string", "function" } },
    editor_window_command = { opts.editor.window_command, { "string", "function" } },
    editor_mappings = { opts.editor.mappings, "table" },
    drawer_window_command = { opts.drawer.window_command, { "string", "function" } },
    drawer_disable_icons = { opts.drawer.disable_icons, "boolean" },
    drawer_icons = { opts.drawer.icons, "table" },
    drawer_mappings = { opts.drawer.mappings, "table" },
    -- ui
    ui_window_open_order = { opts.ui.window_open_order, "table" },
    ui_pre_open_hook = { opts.ui.pre_open_hook, "function" },
    ui_post_open_hook = { opts.ui.post_open_hook, "function" },
    ui_pre_close_hook = { opts.ui.pre_close_hook, "function" },
    ui_post_close_hook = { opts.ui.post_close_hook, "function" },
  }

  m.config = opts

  if m.config.lazy then
    return
  end
  pcall_lazy_setup()
end

---@param connection connection_details
function M.add_connection(connection)
  if not pcall_lazy_setup() then
    return
  end
  m.handler:add_connection(connection)
end

function M.open()
  if not pcall_lazy_setup() then
    return
  end
  if m.open then
    utils.log("warn", "already open")
    return
  end

  m.config.ui.pre_open_hook()

  local order_map = {
    drawer = m.drawer,
    result = m.handler,
    editor = m.editor,
  }

  for _, u in ipairs(m.config.ui.window_open_order) do
    local ui = order_map[u]
    if ui then
      ui:open()
    end
  end

  m.config.ui.post_open_hook()
  m.open = true
end

function M.close()
  if not pcall_lazy_setup() then
    return
  end

  m.config.ui.pre_close_hook()

  m.handler:close()
  m.drawer:close()
  m.editor:close()

  m.config.ui.post_close_hook()
  m.open = false
end

function M.next()
  if not pcall_lazy_setup() then
    return
  end
  m.handler:page_next()
end

function M.prev()
  if not pcall_lazy_setup() then
    return
  end
  m.handler:page_prev()
end

---@param query string query to execute on currently selected connection
function M.execute(query)
  if not pcall_lazy_setup() then
    return
  end
  m.handler:execute(query)
end

---@param format "csv"|"json" format of the output
---@param file string where to save the results
function M.save(format, file)
  if not pcall_lazy_setup() then
    return
  end
  m.handler:save(format, file)
end

---@param command? "wget"|"curl"|"bitsadmin"|"go" preffered command
function M.install(command)
  install.exec(command)
end

-- experimental and subject to change!
M.api = m

return M
