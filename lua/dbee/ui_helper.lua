local utils = require("dbee.utils")

local M = {}

-- Creates a blank hidden buffer.
---@param name string
---@param opts? table<string, any> buffer options
---@return integer bufnr
function M.create_blank_buffer(name, opts)
  opts = opts or {}

  local bufnr = vim.api.nvim_create_buf(false, true)
  -- try setting buffer name - fallback to random string
  local ok = pcall(vim.api.nvim_buf_set_name, bufnr, name)
  if not ok then
    pcall(vim.api.nvim_buf_set_name, bufnr, name .. "-" .. tostring(os.clock()))
  end

  M.configure_buffer_options(bufnr, opts)

  return bufnr
end

---@param bufnr integer
---@param opts? table<string, any> buffer options
function M.configure_buffer_options(bufnr, opts)
  if not bufnr then
    return
  end

  opts = opts or {}

  for opt, val in pairs(opts) do
    vim.api.nvim_buf_set_option(bufnr, opt, val)
  end
end

---@param winid integer
---@param opts? table<string, any> window options
function M.configure_window_options(winid, opts)
  if not winid then
    return
  end
  opts = opts or {}

  for opt, val in pairs(opts) do
    vim.api.nvim_win_set_option(winid, opt, val)
  end
end

-- Sets mappings to the buffer.
---@param bufnr integer
---@param keymap keymap[]
---@param opts? { delete: boolean } if delete is set, remove mappings instad of adding them
function M.configure_buffer_mappings(bufnr, keymap, opts)
  if not bufnr then
    return
  end
  keymap = keymap or {}
  opts = opts or {}

  local set_fn = vim.keymap.set
  if opts.delete then
    set_fn = function(mode, lhs, _, _)
      vim.keymap.del(mode, lhs, { buffer = bufnr })
    end
  end

  -- keymaps
  local default_opts = { noremap = true, nowait = true }

  for _, km in ipairs(keymap) do
    if km.action and type(km.action) == "function" and km.mapping then
      if not vim.tbl_islist(km.mapping) then
        ---@diagnostic disable-next-line
        km.mapping = { km.mapping }
      end

      for _, map in ipairs(km.mapping) do
        if map.key and map.mode then
          local map_opts = map.opts or default_opts
          map_opts.buffer = bufnr
          set_fn(map.mode, map.key, km.action, map_opts)
        end
      end
    end
  end
end

---@type table<integer, boolean> which windows have mapping autocmds already configured
local configured_window_mappings = {}

-- Sets mappings to the window.
---@param winid integer
---@param keymap keymap[]
function M.configure_window_mappings(winid, keymap)
  if not winid then
    return
  end

  if configured_window_mappings[winid] then
    -- autocommands already configured
    return
  end

  -- add mappings when buffer enters the window
  utils.create_window_autocmd({ "BufWinEnter" }, winid, function(event)
    M.configure_buffer_mappings(event.buf, keymap)
  end)

  -- remove mappings when buffer leaves the window
  utils.create_window_autocmd({ "BufWinLeave" }, winid, function(event)
    pcall(M.configure_buffer_mappings, event.buf, keymap, { delete = true })
  end)
end

---@type table<integer, boolean> which buffers have quit autocmds already configured
local configured_quit_buffers = {}

-- Configures quit handle for buffer
---@param bufnr integer
---@param handle fun()
function M.configure_buffer_quit_handle(bufnr, handle)
  if not bufnr then
    return
  end
  handle = handle or function() end

  if configured_quit_buffers[bufnr] then
    -- autocommands already configured
    return
  end

  vim.api.nvim_create_autocmd({ "QuitPre" }, {
    buffer = bufnr,
    callback = handle,
  })

  -- set buffers which have already been mapped
  configured_quit_buffers[bufnr] = true
end

---@type table<integer, boolean> which windows have quit autocmds already configured
local configured_quit_windows = {}

-- Configured quit handle for window.
---@param winid integer
---@param handle fun()
function M.configure_window_quit_handle(winid, handle)
  if not winid then
    return
  end
  handle = handle or function() end

  if configured_quit_windows[winid] then
    -- autocommands already configured
    return
  end

  utils.create_window_autocmd({ "QuitPre" }, winid, handle)

  -- set windows which have already been mapped
  configured_quit_windows[winid] = true
end

return M
