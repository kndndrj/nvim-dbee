
---@class Result
---@field bufnr integer number of buffer
---@field current_result string[]|string current result - lines or file
---@field current_type string type of result - lines or file
local Result = {}

function Result:new()
  local o = {
    bufnr = nil,
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

  -- TODO: move this somewhere else
  vim.o.buflisted = false
  vim.o.bufhidden = "delete"
  vim.o.buftype = "nofile"
  vim.o.swapfile = false
  vim.wo.wrap = false

  local bufnr = self.bufnr
  vim.api.nvim_buf_set_option(bufnr, "modifiable", true)
  vim.api.nvim_buf_set_lines(bufnr, 0, 0, true, lines)
  vim.api.nvim_buf_set_option(bufnr, "modifiable", false)
end

function Result:_show_file()
  local file_path = self.current_result

  vim.api.nvim_command("e " .. file_path)

  -- TODO: move this somewhere else
  vim.o.buflisted = false
  vim.o.swapfile = false
  vim.wo.wrap = false

  vim.api.nvim_buf_set_option(self.bufnr, "modifiable", false)
end

-- Show results on screen
---@param winid integer window id to display the results in - 0 for current
function Result:render(winid)
  -- if buffer doesn't exist, create it
  if not self.bufnr or vim.fn.bufwinnr(self.bufnr) < 0 then
    self.bufnr = vim.api.nvim_create_buf(false, true)
  end

  vim.api.nvim_win_set_buf(winid, self.bufnr)

  if self.current_type == "lines" then
    self:_show_lines()
  elseif self.current_type == "file" then
    self:_show_file()
  end
end


return Result
