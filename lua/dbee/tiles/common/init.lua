local utils = require("dbee.utils")
local floats = require("dbee.tiles.common.floats")

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
---@param actions table<string, fun()>
---@param keymap key_mapping[]
---@param delete? boolean if set, remove mappings instad of adding them
function M.configure_buffer_mappings(bufnr, actions, keymap, delete)
  if not bufnr then
    return
  end
  actions = actions or {}
  keymap = keymap or {}

  local set_fn = vim.keymap.set
  if delete then
    set_fn = function(mode, lhs, _, _)
      vim.keymap.del(mode, lhs, { buffer = bufnr })
    end
  end

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

-- Sets mappings to the window.
---@param winid integer
---@param actions table<string, fun()>
---@param keymap key_mapping[]
function M.configure_window_mappings(winid, actions, keymap)
  if not winid then
    return
  end

  -- add mappings when buffer enters the window
  utils.create_singleton_autocmd({ "BufWinEnter" }, {
    window = winid,
    callback = function(event)
      M.configure_buffer_mappings(event.buf, actions, keymap)
    end,
  })

  -- remove mappings when buffer leaves the window
  utils.create_singleton_autocmd({ "BufWinLeave" }, {
    window = winid,
    callback = function(event)
      pcall(M.configure_buffer_mappings, event.buf, actions, keymap, true)
    end,
  })
end

-- Configures quit handle for buffer
---@param bufnr integer
---@param handle fun()
function M.configure_buffer_quit_handle(bufnr, handle)
  if not bufnr then
    return
  end
  handle = handle or function() end

  utils.create_singleton_autocmd({ "QuitPre" }, {
    buffer = bufnr,
    callback = handle,
  })
end

-- Configured quit handle for window.
---@param winid integer
---@param handle fun()
function M.configure_window_quit_handle(winid, handle)
  if not winid then
    return
  end
  handle = handle or function() end

  utils.create_singleton_autocmd({ "QuitPre" }, {
    window = winid,
    callback = handle,
  })
end

-- Configures immutablity of the window (e.g. only the provided buffer can be opened in
-- the window).
---@param winid integer
---@param bufnr integer
---@param switch? fun(bufnr: integer) optional function that recieves the number of buffer which tried to be opened in the window.
function M.configure_window_immutable_buffer(winid, bufnr, switch)
  if not winid or not bufnr then
    return
  end

  utils.create_singleton_autocmd({ "BufWinEnter", "BufReadPost", "BufNewFile" }, {
    window = winid,
    callback = function(event)
      if event.buf == bufnr then
        return
      end

      pcall(vim.api.nvim_win_set_buf, winid, bufnr)

      if type(switch) == "function" then
        switch(event.buf)
      end
    end,
  })
end

return M
