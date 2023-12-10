local NuiMenu = require("nui.menu")
local NuiInput = require("nui.input")

local M = {}

---@alias menu_select fun(opts?: { title: string, items: string[], callback: fun(selection: string) })
---@alias menu_input fun(opts?: { title: string, default: string, callback: fun(value: string) })

-- Pick items from a list.
---@param relative_winid integer window id
---@param items string[] items to select from
---@param callback fun(item: string) selection callback
---@param title string
function M.select(relative_winid, items, callback, title)
  if not relative_winid or not vim.api.nvim_win_is_valid(relative_winid) then
    error("no window id provided")
  end

  local width = vim.api.nvim_win_get_width(relative_winid)
  local row, _ = unpack(vim.api.nvim_win_get_cursor(relative_winid))

  local popup_options = {
    relative = {
      type = "win",
      winid = relative_winid,
    },
    position = {
      row = row + 1,
      col = 0,
    },
    size = {
      width = width,
    },
    border = {
      style = { "─", "─", "─", "", "─", "─", "─", "" },
      text = {
        top = title,
        top_align = "left",
      },
    },
    win_options = {
      cursorline = true,
    },
  }

  local lines = {}
  for _, item in ipairs(items) do
    table.insert(lines, NuiMenu.item(item))
  end

  local menu = NuiMenu(popup_options, {
    lines = lines,
    keymap = {
      focus_next = { "j", "<Down>", "<Tab>" },
      focus_prev = { "k", "<Up>", "<S-Tab>" },
      close = { "<Esc>", "<C-c>", "q" },
      submit = { "<CR>", "<Space>" },
    },
    on_submit = function(item)
      callback(item.text)
    end,
  })

  menu:mount()
end

-- Ask for input.
---@param relative_winid integer window id
---@param default_value string
---@param callback fun(item: string) selection callback
---@param title string
function M.input(relative_winid, default_value, callback, title)
  if not relative_winid or not vim.api.nvim_win_is_valid(relative_winid) then
    error("no window id provided")
  end

  local width = vim.api.nvim_win_get_width(relative_winid)
  local row, _ = unpack(vim.api.nvim_win_get_cursor(relative_winid))

  local popup_options = {
    relative = {
      type = "win",
      winid = relative_winid,
    },
    position = {
      row = row + 1,
      col = 0,
    },
    size = {
      width = width,
    },
    border = {
      style = { "─", "─", "─", "", "─", "─", "─", "" },
      text = {
        top = title,
        top_align = "left",
      },
    },
    win_options = {
      cursorline = false,
    },
  }

  local input = NuiInput(popup_options, {
    default_value = default_value,
    on_submit = callback,
  })

  for _, key in ipairs { "<Esc>", "<C-c>", "q" } do
    input:map("n", key, function()
      input:unmount()
    end, { noremap = true })
  end

  input:mount()
end

return M
