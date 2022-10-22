local M = {}

--@param lines string[]
function M.show(lines)
  local ls = {}
  for _, l in ipairs(lines) do
    l = l:gsub("\n", "")
    table.insert(ls, l )
  end
  lines = ls

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

---@param file_path string
function M.show_file(file_path)

  vim.api.nvim_command("bo 15split " .. file_path)
  vim.o.buflisted = false
  -- vim.o.buftype = "nofile"
  vim.o.swapfile = false
  vim.wo.wrap = false

  local buf_handle = vim.api.nvim_win_get_buf(0)
  vim.api.nvim_buf_set_option(buf_handle, "modifiable", false)
end

return M
