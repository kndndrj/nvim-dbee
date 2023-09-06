local spinners = require("dbee.progress.spinners")

local M = {}

M.spinners = spinners

---@alias progress_config { text_prefix: string, spinner: spinner }

--- Display an updated progress loader in the specified buffer
---@param bufnr integer -- buffer to display the progres in
---@param opts? progress_config
---@return fun() # cancel function
function M.display(bufnr, opts)
  if not bufnr then
    return function() end
  end
  opts = opts or {}
  local text_prefix = opts.text_prefix or "Loading..."
  local spinner = opts.spinner or spinners.dots

  local icon_index = 1
  local start_time = vim.fn.reltimefloat(vim.fn.reltime())

  local function update()
    local passed_time = vim.fn.reltimefloat(vim.fn.reltime()) - start_time
    icon_index = (icon_index % #spinner) + 1

    local line = string.format("%s %.3f seconds %s ", text_prefix, passed_time, spinner[icon_index])
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, { line })
  end

  local timer = vim.fn.timer_start(100, update, { ["repeat"] = -1 })
  return function()
    vim.fn.timer_stop(timer)
  end
end

return M
