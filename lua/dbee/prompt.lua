local M = {}

---@param prompt string[] list of lines to display as prompt
---@param opts? { width: integer, height: integer, title: string, border: string|string[], callback: fun(result: table<string, string>) } optional parameters
function M.open(prompt, opts)
  opts = opts or {}

  -- add colons to prompt
  for i, p in ipairs(prompt) do
    if not p:find(":$") then
      prompt[i] = p .. ":"
    end
  end

  local win_width = opts.width or 100
  local win_height = opts.height or #prompt
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
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, true, prompt)
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
        local key = p:gsub("(.*):", "%1")
        kv[key] = ""

        for _, l in ipairs(lines) do
          -- if line has prompt prefix, get the value and strip whitespace
          if l:find("^%s*" .. p) then
            local val = l:gsub("^%s*" .. p .. "%s*(.-)%s*$", "%1")
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

return M
