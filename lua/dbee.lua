local Drawer = require("dbee.drawer")
local Editor = require("dbee.editor")
local Handler = require("dbee.handler")
local layout = require("dbee.layout")
local install = require("dbee.install")

-- public and private module objects
local M = {}
local m = {}

---@alias setup_opts { connections: { name: string, type: string, url: string }, lazy: boolean }

---@class Ui
---@field open fun(self: Ui, winid?: integer)
---@field close fun(self: Ui, )

-- is the plugin loaded?
m.loaded = false
---@type setup_opts
m.setup_opts = {}

local function lazy_setup()
  local opts = m.setup_opts

  -- add install binary to path
  vim.env.PATH = install.path() .. ":" .. vim.env.PATH

  m.handler = Handler:new { connections = opts.connections, win_cmd = "bo 15split" }
  if not m.handler then
    print("error in handler setup")
    return
  end

  m.editor = Editor:new { handler = m.handler, win_cmd = "vsplit" }
  if not m.editor then
    print("error in editor setup")
    return
  end

  m.drawer = Drawer:new { handler = m.handler, editor = m.editor, win_cmd = "to 40vsplit" }
  if not m.drawer then
    print("error in drawer setup")
    return
  end

  m.loaded = true
end

---@param opts setup_opts
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
