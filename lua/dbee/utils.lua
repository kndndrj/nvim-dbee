local M = {}

-- private variable with registered onces
---@type table<string, boolean>
local used_onces = {}

---@param id string unique id of this singleton bool
---@return boolean
function M.once(id)
  id = id or ""

  if used_onces[id] then
    return false
  end

  used_onces[id] = true

  return true
end

-- Get cursor range of current selection
---@return integer start row
---@return integer start column
---@return integer end row
---@return integer end column
function M.visual_selection()
  -- return to normal mode ('< and '> become available only after you exit visual mode)
  local key = vim.api.nvim_replace_termcodes("<esc>", true, false, true)
  vim.api.nvim_feedkeys(key, "x", false)

  local _, srow, scol, _ = unpack(vim.fn.getpos("'<"))
  local _, erow, ecol, _ = unpack(vim.fn.getpos("'>"))
  if ecol > 200000 then
    ecol = 20000
  end
  if srow < erow or (srow == erow and scol <= ecol) then
    return srow - 1, scol - 1, erow - 1, ecol
  else
    return erow - 1, ecol - 1, srow - 1, scol
  end
end

---@param level "info"|"warn"|"error"
---@param message string
---@param subtitle? string
function M.log(level, message, subtitle)
  -- log level
  local l = vim.log.levels.OFF
  if level == "info" then
    l = vim.log.levels.INFO
  elseif level == "warn" then
    l = vim.log.levels.WARN
  elseif level == "error" then
    l = vim.log.levels.ERROR
  end

  -- subtitle
  if subtitle then
    subtitle = "[" .. subtitle .. "]:"
  else
    subtitle = ""
  end
  vim.notify(subtitle .. " " .. message, l, { title = "nvim-dbee" })
end

-- Gets keys of a map and sorts them by name
---@param obj table<string, any> map-like table
---@return string[]
function M.sorted_keys(obj)
  local keys = {}
  for k, _ in pairs(obj) do
    table.insert(keys, k)
  end
  table.sort(keys)
  return keys
end

-- create an autocmd that is associated with a window rather than a buffer.
---@param events string[]
---@param winid integer
---@param opts table<string, any>
local function create_window_autocmd(events, winid, opts)
  opts = opts or {}
  if not events or not winid or not opts.callback then
    return
  end

  local cb = opts.callback

  opts.callback = function(event)
    -- remove autocmd if window is closed
    if not vim.api.nvim_win_is_valid(winid) then
      vim.api.nvim_del_autocmd(event.id)
      return
    end

    local wid = vim.fn.bufwinid(event.buf or -1)
    if wid ~= winid then
      return
    end
    cb(event)
  end

  vim.api.nvim_create_autocmd(events, opts)
end

-- create an autocmd just once in a single place in code.
-- If opts hold a "window" key, autocmd is defined per window rather than a buffer.
-- If window and buffer are provided, this results in an error.
---@param events string[] events list as defined in nvim api
---@param opts table<string, any> options as in api
function M.create_singleton_autocmd(events, opts)
  if opts.window and opts.buffer then
    error("cannot register autocmd for buffer and window at the same time")
  end

  local caller_info = debug.getinfo(2)
  if not caller_info or not caller_info.name or not caller_info.currentline then
    error("could not determine function caller")
  end

  if
    not M.once(
      "autocmd_singleton_"
        .. caller_info.name
        .. caller_info.currentline
        .. tostring(opts.window)
        .. tostring(opts.buffer)
    )
  then
    -- already configured
    return
  end

  if opts.window then
    local window = opts.window
    opts.window = nil
    create_window_autocmd(events, window, opts)
    return
  end

  vim.api.nvim_create_autocmd(events, opts)
end

local random_charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

--- Generate a random string
---@return string _ random string of 10 characters
function M.random_string()
  local function r(length)
    if length < 1 then
      return ""
    end

    local i = math.random(1, #random_charset)
    return r(length - 1) .. random_charset:sub(i, i)
  end

  return r(10)
end

return M
