local Drawer = require("db.drawer")
local Editor = require("db.editor")
local Handler = require("db.handler")

-- public and private module objects
local M = {}
local m = {}

---@alias setup_opts { connections: { name: string, type: string, url: string }, lazy: boolean }

---@class Ui
---@field open fun()
---@field close fun()

-- is the plugin loaded?
m.loaded = false
---@type setup_opts
m.setup_opts = {}

local function lazy_setup()
  local opts = m.setup_opts

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
  m.drawer:open()
end

function M.close()
  if not m.loaded then
    lazy_setup()
  end
  m.drawer:close()
  m.handler:close()
end

function M.handler()
  if not m.loaded then
    lazy_setup()
  end
  return m.handler
end

function M.editor()
  if not m.loaded then
    lazy_setup()
  end
  return m.editor
end

return M
