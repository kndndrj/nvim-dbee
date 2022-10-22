---@class Editor
---@field buffers { integer: boolean } list of buffer handles
---@field last_buffer integer last non-db-editor buffer
local Editor = {}

function Editor:new()
  -- class object
  local o = {
    buffers = {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

function Editor:show()
  local current_buf = vim.api.nvim_get_current_buf()

  local win = vim.api.nvim_get_current_win()
  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_win_set_buf(win, buf)
  vim.api.nvim_buf_set_name(buf, "DB Editor")

  -- update last buffer if not in buffer list
  if not self.buffers[current_buf] then
    self.last_buffer = current_buf
  end
  -- update buffer list
  self.buffers[buf] = true
end

return Editor
