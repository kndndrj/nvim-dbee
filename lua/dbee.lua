local Drawer = require("dbee.drawer")
local Editor = require("dbee.editor")
local Handler = require("dbee.handler")
local layout = require("dbee.layout")
local install = require("dbee.install")

-- public and private module objects
local M = {}
local m = {}

-- configuration object
---@class Config
---@field connections { name: string, type: string, url: string }[] list of configured database connections
---@field lazy boolean lazy load the plugin or not?
---@field drawer drawer_config
---@field editor editor_config
---@field result handler_config

-- default configuration
---@type Config
local default_config = {
  connections = {},
  lazy = false,
  drawer = {
    fallback_window_command = "to 40vsplit",
    disable_icons = false,
    icons = {
      history = {
        icon = "",
        highlight = "Constant",
      },
      scratch = {
        icon = "",
        highlight = "Character",
      },
      database = {
        icon = "",
        highlight = "SpecialChar",
      },
      table = {
        icon = "",
        highlight = "Conditional",
      },

      -- if there is no type
      -- use this for normal nodes...
      none = {
        icon = " ",
      },
      -- ...and use this for nodes with children
      none_dir = {
        icon = "",
        highlight = "NonText",
      },
    },
  },
  result = {
    fallback_window_command = "bo 15split",
  },
  editor = {
    fallback_window_command = function()
      return vim.api.nvim_get_current_win()
    end,
  },
}

-- is the plugin loaded?
m.loaded = false
---@type Config
m.setup_opts = {}

local function lazy_setup()
  ---@type Config
  local opts = vim.tbl_deep_extend("force", default_config, m.setup_opts)

  -- add install binary to path
  vim.env.PATH = install.path() .. ":" .. vim.env.PATH

  m.handler = Handler:new(opts.connections, opts.result)
  if not m.handler then
    print("error in handler setup")
    return
  end

  m.editor = Editor:new(m.handler, opts.editor)
  if not m.editor then
    print("error in editor setup")
    return
  end

  m.drawer = Drawer:new(m.handler, m.editor, opts.drawer)
  if not m.drawer then
    print("error in drawer setup")
    return
  end

  m.loaded = true
end

---@param opts Config
function M.setup(opts)
  m.setup_opts = opts or {}
  if m.setup_opts.lazy then
    return
  end
  lazy_setup()
end

function M.open()
  if not m.loaded then
    lazy_setup()
  end
  if m.egg then
    print("already open")
    return
  end
  -- save layout before doing anything
  m.egg = layout.save()

  -- create a new layout
  vim.cmd("new")
  vim.cmd("only")
  local editor_win = vim.api.nvim_get_current_win()
  local tmp_buf = vim.api.nvim_get_current_buf()

  -- open windows
  m.editor:open(editor_win)
  m.drawer:open()
  m.handler:open()

  vim.cmd("bd " .. tmp_buf)
end

function M.close()
  if not m.loaded then
    lazy_setup()
  end

  layout.restore(m.egg)
  m.egg = nil
end

function M.next()
  if not m.loaded then
    lazy_setup()
  end
  m.handler:page_next()
end

function M.prev()
  if not m.loaded then
    lazy_setup()
  end
  m.handler:page_prev()
end

---@param command? "wget"|"curl"|"bitsadmin"|"go" preffered command
function M.install(command)
  install.exec(command)
end

return M
