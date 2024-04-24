local floats = require("dbee.ui.common.floats")
local utils = require("dbee.utils")

local M = {}

-- expose floats
M.float_editor = floats.editor
M.float_hover = floats.hover
M.float_prompt = floats.prompt

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
    pcall(vim.api.nvim_buf_set_name, bufnr, name .. "-" .. utils.random_string())
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
---@param actions table<string, fun()>
---@param keymap key_mapping[]
function M.configure_buffer_mappings(bufnr, actions, keymap)
  if not bufnr then
    return
  end
  actions = actions or {}
  keymap = keymap or {}

  local set_fn = vim.keymap.set

  -- keymaps
  local default_opts = { noremap = true, nowait = true }

  for _, km in ipairs(keymap) do
    if km.key and km.mode then
      local action
      if type(km.action) == "string" then
        action = actions[km.action]
      elseif type(km.action) == "function" then
        action = km.action
      end

      if action then
        local map_opts = km.opts or default_opts
        map_opts.buffer = bufnr
        set_fn(km.mode, km.key, action, map_opts)
      end
    end
  end
end

return M
