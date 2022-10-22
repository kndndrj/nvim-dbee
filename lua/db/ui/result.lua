
---@class Result
---@field current_result string[]|string current result - lines or file
---@field current_type string type of result - lines or file
local Result = {}

function Result:new()
  local o = {
    current_result = nil,
    current_type = "",
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param result string[]|string result to show on open
---@param type "lines"|"file"
function Result:set(result, type)
  self.current_result = result
  self.current_type = type
end

function Result:_show_lines()
  local lines = {}
  for _, l in ipairs(self.current_result) do
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

function Result:_show_file()
  local file_path = self.current_result

  vim.api.nvim_command("bo 15split " .. file_path)
  vim.o.buflisted = false
  -- vim.o.buftype = "nofile"
  vim.o.swapfile = false
  vim.wo.wrap = false

  local buf_handle = vim.api.nvim_win_get_buf(0)
  vim.api.nvim_buf_set_option(buf_handle, "modifiable", false)
end

function Result:show()
  if self.current_type == "lines" then
    self:_show_lines()
  elseif self.current_type == "file" then
    self:_show_file()
  end
end


return Result
