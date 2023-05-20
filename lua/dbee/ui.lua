---@alias ui_config { buffer_options: table<string, any>, window_options: table<string, any>, window_command: string|fun():integer }

---@class Ui
---@field private winid integer
---@field private bufnr integer
---@field private window_options table<string, any>
---@field private buffer_options table<string, any>
---@field private window_command fun():integer function which opens a new window and returns a window id
local Ui = {}

---@param opts? ui_config
---@return Ui
function Ui:new(opts)
  opts = opts or {}

  local win_cmd
  if type(opts.window_command) == "string" then
    win_cmd = function()
      vim.cmd(opts.window_command)
      return vim.api.nvim_get_current_win()
    end
  elseif type(opts.window_command) == "function" then
    win_cmd = opts.window_command
  else
    win_cmd = function()
      vim.cmd("vsplit")
      return vim.api.nvim_get_current_win()
    end
  end

  -- class object
  local o = {
    winid = nil,
    bufnr = nil,
    window_command = win_cmd,
    window_options = opts.window_options or {},
    buffer_options = opts.buffer_options or {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@return integer winid
function Ui:window()
  return self.winid
end

---@return integer bufnr
function Ui:buffer()
  return self.bufnr
end

---@return integer winid
---@return integer bufnr
function Ui:open()
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    self.winid = self.window_command()
  end

  -- if buffer doesn't exist, create it
  if not self.bufnr or not vim.api.nvim_buf_is_valid(self.bufnr) then
    self.bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(self.bufnr, "dbee-" .. tostring(os.clock()))
  end
  vim.api.nvim_win_set_buf(self.winid, self.bufnr)
  vim.api.nvim_set_current_win(self.winid)

  -- set options
  for opt, val in pairs(self.buffer_options) do
    vim.api.nvim_buf_set_option(self.bufnr, opt, val)
  end
  for opt, val in pairs(self.window_options) do
    vim.api.nvim_win_set_option(self.winid, opt, val)
  end

  return self.winid, self.bufnr
end

function Ui:close()
  pcall(vim.api.nvim_win_close, self.winid, false)
end

return Ui
