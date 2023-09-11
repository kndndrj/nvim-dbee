-- this package contains various floating window utilities such as floating editor and an input prompt

local M = {
  call_log = require("dbee.floats.call_log").call_log,
}

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
---@param opts? { width: integer, height: integer, title: string, border: string|string[], callback: fun(result: table<string, string>) } optional parameters
function M.prompt(prompt, opts)
  opts = opts or {}

  -- create lines to display
  ---@type string[]
  local display_prompt = {}
  for _, p in ipairs(prompt) do
    table.insert(display_prompt, p.name .. ": " .. (p.default or ""))
  end

  local win_width = opts.width or 100
  local win_height = opts.height or #display_prompt
  local ui_spec = vim.api.nvim_list_uis()[1]
  local x = math.floor((ui_spec["width"] - win_width) / 2)
  local y = math.floor((ui_spec["height"] - win_height) / 2)

  -- create new buffer
  local bufnr = vim.api.nvim_create_buf(false, false)
  local name = opts.title or tostring(os.clock())
  vim.api.nvim_buf_set_name(bufnr, name)
  vim.api.nvim_buf_set_option(bufnr, "filetype", "dbee")
  vim.api.nvim_buf_set_option(bufnr, "buftype", "acwrite")
  vim.api.nvim_buf_set_option(bufnr, "bufhidden", "delete")

  -- fill buffer contents
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, true, display_prompt)
  vim.api.nvim_buf_set_option(bufnr, "modified", false)

  -- open window
  local winid = vim.api.nvim_open_win(bufnr, true, {
    relative = "editor",
    width = win_width,
    height = win_height,
    col = x,
    row = y,
    border = opts.border or "rounded",
    title = opts.title or "",
    title_pos = "center",
    style = "minimal",
  })
  -- apply the highlighting of keys to window
  highlight_prompt(prompt, winid)

  local callback = opts.callback or function() end

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
---@param opts? { width: integer, height: integer, title: string, border: string|string[], callback: fun() } optional parameters
function M.editor(file, opts)
  opts = opts or {}

  local ui_spec = vim.api.nvim_list_uis()[1]
  local win_width = opts.width or (ui_spec["width"] - 50)
  local win_height = opts.height or (ui_spec["height"] - 10)
  local x = math.floor((ui_spec["width"] - win_width) / 2)
  local y = math.floor((ui_spec["height"] - win_height) / 2)

  -- create new dummy buffer
  local tmp_buf = vim.api.nvim_create_buf(false, true)

  -- open window
  local winid = vim.api.nvim_open_win(tmp_buf, true, {
    relative = "editor",
    width = win_width,
    height = win_height,
    col = x,
    row = y,
    border = opts.border or "rounded",
    title = opts.title or "",
    title_pos = "center",
    style = "minimal",
  })

  -- open the file
  vim.cmd("e " .. file)
  local bufnr = vim.api.nvim_get_current_buf()
  vim.api.nvim_buf_set_option(bufnr, "bufhidden", "delete")

  local callback = opts.callback or function() end

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

return M
