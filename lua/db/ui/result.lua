
---@class Result
---@field lines string[] lines to show when window is open
local Result = {}

function Result:new()
  local o = {
    lines = {}
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param result string[] result to show on open
function Result:set(result)
  self.lines = result
end

function Result:show()
  local lines = {}
  for _, l in ipairs(self.lines) do
    l = l:gsub("\n", "")
    table.insert(lines, l )
  end

  vim.api.nvim_command("bo 15new")
  vim.o.buflisted = false
  vim.o.bufhidden = "delete"
  vim.o.buftype = "nofile"
  vim.o.swapfile = false
  vim.wo.wrap = false

  local buf_handle = vim.api.nvim_win_get_buf(0)
  vim.api.nvim_buf_set_option(buf_handle, "modifiable", true)
  vim.api.nvim_buf_set_lines(buf_handle, 0, 0, true, lines)
  vim.api.nvim_buf_set_option(buf_handle, "modifiable", false)
end

return Result
