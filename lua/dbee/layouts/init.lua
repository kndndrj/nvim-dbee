local tools = require("dbee.layouts.tools")

---@alias window_layout_tiles { drawer: Drawer, editor: Editor, result: Result, call_log: CallLog }

-- Window layout defines how windows are opened.
---@class WindowLayout
---@field open fun(self: WindowLayout, tiles: window_layout_tiles) function to open ui.
---@field close fun(self: WindowLayout) function to close ui.

local M = {}

-- Default layout uses a helper to save the existing window layout before opening any windows,
-- then makes a new empty window for the editor and then opens result and drawer.
-- When later calling close(), the previously saved layout is restored.
---@class DefaultWindowLayout: WindowLayout
---@field private egg? layout_egg
---@field private windows integer[]
M.Default = {}

--- Loads connections from json file
---@return WindowLayout
function M.Default:new()
  local o = {
    egg = nil,
    windows = {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param tiles window_layout_tiles
function M.Default:open(tiles)
  -- save layout before opening ui
  self.egg = tools.save()

  self.windows = {}

  -- editor
  tools.make_only(0)
  local editor_win = vim.api.nvim_get_current_win()
  table.insert(self.windows, editor_win)
  tiles.editor:show(editor_win)

  -- result
  vim.cmd("bo 15split")
  local win = vim.api.nvim_get_current_win()
  table.insert(self.windows, win)
  tiles.result:show(win)

  -- drawer
  vim.cmd("to 40vsplit")
  win = vim.api.nvim_get_current_win()
  table.insert(self.windows, win)
  tiles.drawer:show(win)

  -- call log
  vim.cmd("belowright 15split")
  win = vim.api.nvim_get_current_win()
  table.insert(self.windows, win)
  tiles.call_log:show(win)

  -- set cursor to drawer
  vim.api.nvim_set_current_win(editor_win)
end

function M.Default:close()
  -- close all windows
  for _, win in ipairs(self.windows) do
    pcall(vim.api.nvim_win_close, win, false)
  end

  -- restore layout
  tools.restore(self.egg)
  self.egg = nil
end

return M
