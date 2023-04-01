---@class UI
---@field winid integer
---@field bufnr integer
---@field win_cmd string
local UI = {}

---@param opts? { win_cmd: string }
---@return UI
function UI:new(opts)
  opts = opts or {}

  local win_cmd = opts.win_cmd
  if not win_cmd then
    win_cmd  = "bo 15split"
  end

  -- class object
  local o = {
    bufnr = nil,
    winid = nil,
    win_cmd = win_cmd,
  }

  setmetatable(o, self)
  self.__index = self
  return o
end

-- Create new buffer and window and return buffer handle
---@return integer bufnr
function UI:open()
  -- if buffer doesn't exist, create it
  if not self.bufnr or vim.fn.bufwinnr(self.bufnr) < 0 then
    self.bufnr = vim.api.nvim_create_buf(false, true)
  end

  -- if window doesn't exist, create it
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    vim.cmd(self.win_cmd)
    self.winid = vim.api.nvim_get_current_win()
  end

  vim.api.nvim_win_set_buf(self.winid, self.bufnr)

  vim.o.buflisted = false
  vim.o.bufhidden = "delete"
  vim.o.buftype = "nofile"
  vim.o.swapfile = false
  vim.wo.wrap = false

  return self.bufnr
end

function UI:close() end

return UI
