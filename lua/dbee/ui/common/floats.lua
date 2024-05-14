-- this package contains various floating window utilities such as floating editor and an input prompt
local utils = require("dbee.utils")

local M = {}

---User defined options for floating windows
---@type table<string, any>
local OPTS = {}

---Set up custom floating window parameters
---@param opts? table<string, any>
function M.configure(opts)
  OPTS = vim.tbl_extend("force", {
    border = "rounded",
    title_pos = "center",
    style = "minimal",
    title = "",
    zindex = 150,
  }, opts or {})
end

---Merges user defined options with provided spec.
---@param spec table<string, any>
local function enrich_float_opts(spec)
  return vim.tbl_extend("keep", spec, OPTS)
end

---@alias prompt { name: string, default: string }[] list of lines with optional defaults to display as prompt

--- highlight the prompt keys
---@param prompt prompt
---@param winid integer window to apply the highlight to
local function highlight_prompt(prompt, winid)
  -- assemble the command
  ---@type string[]
  local patterns = {}
  for _, p in ipairs(prompt) do
    table.insert(patterns, string.format([[^\s*%s]], p.name))
  end

  local cmd = string.format("match Question /%s/", table.concat(patterns, [[\|]]))

  local current_win = vim.api.nvim_get_current_win()
  -- just apply the highlight if we apply the highlight to current win
  if not winid or winid == 0 or winid == current_win then
    vim.cmd(cmd)
    return
  end

  -- switch to provided window, apply hightlight and jump back
  vim.api.nvim_set_current_win(winid)
  vim.cmd(cmd)
  vim.api.nvim_set_current_win(current_win)
end

---@param prompt prompt
---@param spec? { title: string, callback: fun(result: table<string, string>) }
function M.prompt(prompt, spec)
  spec = spec or {}

  -- create lines to display
  ---@type string[]
  local display_prompt = {}
  for _, p in ipairs(prompt) do
    table.insert(display_prompt, p.name .. ": " .. (p.default or ""))
  end

  local win_width = 100
  local win_height = #display_prompt
  local ui_spec = vim.api.nvim_list_uis()[1]
  local x = math.floor((ui_spec["width"] - win_width) / 2)
  local y = math.floor((ui_spec["height"] - win_height) / 2)

  -- create new buffer
  local bufnr = vim.api.nvim_create_buf(false, false)
  local name = spec.title or utils.random_string()
  vim.api.nvim_buf_set_name(bufnr, name)
  vim.api.nvim_buf_set_option(bufnr, "filetype", "dbee")
  vim.api.nvim_buf_set_option(bufnr, "buftype", "acwrite")
  vim.api.nvim_buf_set_option(bufnr, "bufhidden", "delete")

  -- fill buffer contents
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, true, display_prompt)
  vim.api.nvim_buf_set_option(bufnr, "modified", false)

  -- open window
  local winid = vim.api.nvim_open_win(
    bufnr,
    true,
    enrich_float_opts {
      relative = "editor",
      width = win_width,
      height = win_height,
      col = x,
      row = y,
      title = spec.title or "",
    }
  )
  -- apply the highlighting of keys to window
  highlight_prompt(prompt, winid)

  local callback = spec.callback or function() end

  -- set callbacks
  vim.api.nvim_create_autocmd("BufWriteCmd", {
    buffer = bufnr,
    callback = function()
      -- reset modified flag
      vim.api.nvim_buf_set_option(bufnr, "modified", false)

      -- get lines
      local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)

      -- close the window if not using "wq" already
      local cmd_hist = vim.api.nvim_exec2(":history cmd -1", { output = true })
      local last_cmd = cmd_hist.output:gsub(".*\n>%s*%d+%s*(.*)%s*", "%1")
      if not last_cmd:find("^wq") then
        vim.api.nvim_win_close(winid, true)
      end

      -- create key-value from prompt and values and trigger callback
      local kv = {}
      for _, p in ipairs(prompt) do
        -- get key from prompt and store it as empty string by default
        local key = p.name
        kv[key] = ""

        for _, l in ipairs(lines) do
          -- if line has prompt prefix, get the value and strip whitespace
          if l:find("^%s*" .. p.name .. ":") then
            local val = l:gsub("^%s*" .. p.name .. ":%s*(.-)%s*$", "%1")
            kv[key] = val
          end
        end
      end

      callback(kv)
    end,
  })

  vim.api.nvim_create_autocmd({ "BufLeave", "BufWritePost" }, {
    buffer = bufnr,
    callback = function()
      vim.api.nvim_win_close(winid, true)
    end,
  })

  -- set keymaps
  vim.keymap.set("n", "q", function()
    vim.api.nvim_win_close(winid, true)
  end, { silent = true, buffer = bufnr })

  vim.keymap.set("i", "<CR>", function()
    -- write and return to normal mode
    vim.cmd(":w")
    vim.api.nvim_input("<C-\\><C-N>")
  end, { silent = true, buffer = bufnr })
end

---@param file string file to edit
---@param spec? { title: string, callback: fun() } required parameters for float.
function M.editor(file, spec)
  spec = spec or {}

  local ui_spec = vim.api.nvim_list_uis()[1]
  local win_width = ui_spec["width"] - 50
  local win_height = ui_spec["height"] - 10
  local x = math.floor((ui_spec["width"] - win_width) / 2)
  local y = math.floor((ui_spec["height"] - win_height) / 2)

  -- create new dummy buffer
  local tmp_buf = vim.api.nvim_create_buf(false, true)

  -- open window
  local winid = vim.api.nvim_open_win(
    tmp_buf,
    true,
    enrich_float_opts {
      title = spec.title or "",
      relative = "editor",
      width = win_width,
      height = win_height,
      col = x,
      row = y,
    }
  )

  -- open the file
  vim.cmd("e " .. file)
  local bufnr = vim.api.nvim_get_current_buf()
  vim.api.nvim_buf_set_option(bufnr, "bufhidden", "delete")

  local callback = spec.callback or function() end

  -- set callbacks
  vim.api.nvim_create_autocmd("BufWritePost", {
    buffer = bufnr,
    callback = callback,
  })

  vim.api.nvim_create_autocmd({ "BufLeave", "BufWritePost" }, {
    buffer = bufnr,
    callback = function()
      -- close the window if not using "wq" already
      local cmd_hist = vim.api.nvim_exec2(":history cmd -1", { output = true })
      local last_cmd = cmd_hist.output:gsub(".*\n>%s*%d+%s*(.*)%s*", "%1")
      if not last_cmd:find("^wq") then
        pcall(vim.api.nvim_win_close, winid, true)
        pcall(vim.api.nvim_buf_delete, bufnr, {})
      end
    end,
  })

  -- set keymaps
  vim.keymap.set("n", "q", function()
    vim.api.nvim_win_close(winid, true)
  end, { silent = true, buffer = bufnr })
end

---@param winid integer window to chech the neighbors of
---@return boolean # true if window has a right neighbor
local function has_neighbor_right(winid)
  local right_winid = vim.fn.win_getid(vim.fn.winnr("l"))
  if right_winid == 0 then
    return false
  end

  return winid ~= right_winid
end

---@param winid integer window to chech the neighbors of
---@return boolean # true if window has a left neighbor
local function has_neighbor_left(winid)
  local left_winid = vim.fn.win_getid(vim.fn.winnr("h"))
  if left_winid == 0 then
    return false
  end

  return winid ~= left_winid
end

-- hover window with custom content
---@param relative_winid? integer window to set the hover relative to
---@param contents string[] file to edit
---@return fun() # close handle
function M.hover(relative_winid, contents)
  if not contents or #contents < 1 or not relative_winid or not vim.api.nvim_win_is_valid(relative_winid) then
    return function() end
  end

  local win_width = 1
  for _, line in ipairs(contents) do
    if #line > win_width then
      win_width = #line
    end
  end

  local win_height = #contents

  -- create new buffer with contents
  local bufnr = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, contents)
  vim.api.nvim_buf_set_option(bufnr, "filetype", "dbee")
  vim.api.nvim_buf_set_option(bufnr, "bufhidden", "delete")

  -- row is relative to cursor in the "parent" window
  local cursor_row, _ = unpack(vim.api.nvim_win_get_cursor(relative_winid))

  -- open to left/right based on window position
  local col = 0
  local anchor = "NW"
  if has_neighbor_right(relative_winid) then
    col = vim.api.nvim_win_get_width(relative_winid)
  elseif has_neighbor_left(relative_winid) then
    anchor = "NE"
  end

  -- open window

  local winid = vim.api.nvim_open_win(
    bufnr,
    false,
    enrich_float_opts {
      relative = "win",
      win = relative_winid,
      width = win_width,
      height = win_height,
      col = col,
      row = cursor_row - 1,
      anchor = anchor,
    }
  )

  return function()
    pcall(vim.api.nvim_win_close, winid, true)
    pcall(vim.api.nvim_buf_delete, bufnr, {})
  end
end

return M
