local utils = require("db.utils")
local layout = require("db.layout")

---@class Editor
---@field private handler Handler
---@field private ui_opts { win_cmd: string, bufnr: integer, winid: integer}
---@field private egg layoutEgg
local Editor = {}

---@param opts? { handler: Handler, win_cmd: string }
---@return Editor|nil
function Editor:new(opts)
  opts = opts or {}

  if opts.handler == nil then
    print("no Handler provided to editor")
    return
  end

  -- class object
  local o = {
    handler = opts.handler,
    ui_opts = {
      win_cmd = opts.win_cmd or "vsplit",
      bufnr = nil,
      winid = nil,
    },
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

-- TODO
function Editor:execute_selection()
  local srow, scol, erow, ecol = utils.visual_selection()

  local selection = vim.api.nvim_buf_get_text(0, srow, scol, erow, ecol, {})
  local query = table.concat(selection, "\n")

  self.handler:execute(query)
end

function Editor:open()

  -- save layout before doing anything
  self.egg = layout.save()

  -- if buffer doesn't exist, create it
  local bufnr = self.ui_opts.bufnr
  if not bufnr or not vim.api.nvim_buf_is_valid(bufnr) then
    bufnr = vim.api.nvim_create_buf(false, true)
  end

  -- if window doesn't exist, create it
  local winid = self.ui_opts.winid
  if not winid or not vim.api.nvim_win_is_valid(winid) then
    vim.cmd(self.ui_opts.win_cmd)
    winid = vim.api.nvim_get_current_win()
  end

  vim.api.nvim_win_set_buf(winid, bufnr)
  vim.api.nvim_set_current_win(winid)

  self.ui_opts.bufnr = bufnr
  self.ui_opts.winid = winid

  vim.o.buflisted = false
  vim.o.bufhidden = "delete"
  vim.o.buftype = "nofile"
  vim.o.swapfile = false
  vim.wo.winfixheight = true
  vim.wo.winfixwidth = true
end

function Editor:close()
  layout.restore(self.egg)
  self.egg = nil
end

return Editor
