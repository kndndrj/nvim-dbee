local Drawer = require("dbee.drawer")
local Editor = require("dbee.editor")
local Result = require("dbee.result")
local Ui = require("dbee.ui")
local Handler = require("dbee.handler")
local install = require("dbee.install")
local utils = require("dbee.utils")
local default_config = require("dbee.config").default

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

  -- set up UIs
  local result_ui = Ui:new {
    window_command = m.config.ui.window_commands.result,
    window_options = {
      wrap = false,
      winfixheight = true,
      winfixwidth = true,
      number = false,
    },
  }
  local editor_ui = Ui:new {
    window_command = m.config.ui.window_commands.editor,
  }
  local drawer_ui = Ui:new {
    window_command = m.config.ui.window_commands.drawer,
    buffer_options = {
      buflisted = false,
      bufhidden = "delete",
      buftype = "nofile",
      swapfile = false,
    },
    window_options = {
      wrap = false,
      winfixheight = true,
      winfixwidth = true,
      number = false,
    },
  }

  -- set up modules
  m.handler = Handler:new(result_ui, m.config.sources, { fallback_page_size = m.config.page_size })
  m.result = Result:new(result_ui, m.handler, m.config.result)
  m.editor = Editor:new(editor_ui, m.handler, m.config.editor)
  m.drawer = Drawer:new(drawer_ui, m.handler, m.editor, m.config.drawer)

  m.handler:add_helpers(m.config.extra_helpers)
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
    sources = { opts.sources, "table" },
    lazy = { opts.lazy, "boolean" },
    extra_helpers = { opts.extra_helpers, "table" },
    -- submodules
    editor_mappings = { opts.editor.mappings, "table" },
    drawer_disable_candies = { opts.drawer.disable_candies, "boolean" },
    drawer_candies = { opts.drawer.candies, "table" },
    drawer_mappings = { opts.drawer.mappings, "table" },
    -- ui
    ui_window_commands = { opts.ui.window_commands, "table" },
    ui_window_commands_drawer = { opts.ui.window_commands.drawer, { "string", "function" } },
    ui_window_commands_result = { opts.ui.window_commands.result, { "string", "function" } },
    ui_window_commands_editor = { opts.ui.window_commands.editor, { "string", "function" } },
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

---@param params connection_details
---@param source_id source_id id of the source to save connection to
function M.add_connection(params, source_id)
  if not pcall_lazy_setup() then
    return
  end
  m.handler:add_connection(params, source_id)
end

function M.toggle()
  if m.open then
    M.close()
  else
    M.open()
  end
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
    result = m.result,
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

  m.result:close()
  m.drawer:close()
  m.editor:close()

  m.config.ui.post_close_hook()
  m.open = false
end

function M.next()
  if not pcall_lazy_setup() then
    return
  end
  m.handler:current_connection():page_next()
end

function M.prev()
  if not pcall_lazy_setup() then
    return
  end
  m.handler:current_connection():page_prev()
end

---@param query string query to execute on currently selected connection
function M.execute(query)
  if not pcall_lazy_setup() then
    return
  end
  m.handler:current_connection():execute(query)
end

---@param format "csv"|"json" format of the output
---@param file string where to save the results
function M.save(format, file)
  if not pcall_lazy_setup() then
    return
  end
  m.handler:current_connection():save(format, file)
end

---@param command? install_command preffered command
function M.install(command)
  install.exec(command)
end

-- experimental and subject to change!
M.api = m

return M
