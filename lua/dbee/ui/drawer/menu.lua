local NuiMenu = require("nui.menu")
local NuiInput = require("nui.input")

local M = {}

---@alias menu_select fun(opts?: { title: string, items: string[], on_confirm: fun(selection: string), on_yank: fun(selection: string) })
---@alias menu_input fun(opts?: { title: string, default: string, on_confirm: fun(value: string) })

-- Pick items from a list.
---@param opts { relative_winid: integer, items: string[], on_confirm: fun(item: string), on_yank: fun(item:string), title: string, mappings: key_mapping[] }
function M.select(opts)
  opts = opts or {}
  if not opts.relative_winid or not vim.api.nvim_win_is_valid(opts.relative_winid) then
    error("no window id provided")
  end

  local width = vim.api.nvim_win_get_width(opts.relative_winid)
  local row, _ = unpack(vim.api.nvim_win_get_cursor(opts.relative_winid))

  local popup_options = {
    relative = {
      type = "win",
      winid = opts.relative_winid,
    },
    position = {
      row = row + 1,
      col = 0,
    },
    size = {
      width = width,
    },
    zindex = 160,
    border = {
      style = { "─", "─", "─", "", "─", "─", "─", "" },
      text = {
        top = opts.title or "",
        top_align = "left",
      },
    },
    win_options = {
      cursorline = true,
    },
  }

  local lines = {}
  for _, item in ipairs(opts.items or {}) do
    table.insert(lines, NuiMenu.item(item))
  end

  local menu = NuiMenu(popup_options, {
    lines = lines,
    keymap = {
      focus_next = { "j", "<Down>", "<Tab>" },
      focus_prev = { "k", "<Up>", "<S-Tab>" },
      close = {},
      submit = {},
    },
    on_submit = function() end,
  })

  -- configure mappings
  for _, km in ipairs(opts.mappings or {}) do
    local action
    if km.action == "menu_confirm" then
      action = opts.on_confirm
    elseif km.action == "menu_yank" then
      action = opts.on_yank
    elseif km.action == "menu_close" then
      action = function() end
    end

    local map_opts = km.opts or { noremap = true, nowait = true }

    if action then
      menu:map(km.mode, km.key, function()
        local item = menu.tree:get_node()
        menu:unmount()
        if item then
          action(item.text)
        end
      end, map_opts)
    end
  end

  menu:mount()
end

-- Ask for input.
---@param opts { relative_winid: integer, default_value: string, on_confirm: fun(item: string), title: string, mappings: key_mapping[] }
function M.input(opts)
  if not opts.relative_winid or not vim.api.nvim_win_is_valid(opts.relative_winid) then
    error("no window id provided")
  end

  local width = vim.api.nvim_win_get_width(opts.relative_winid)
  local row, _ = unpack(vim.api.nvim_win_get_cursor(opts.relative_winid))

  local popup_options = {
    relative = {
      type = "win",
      winid = opts.relative_winid,
    },
    position = {
      row = row + 1,
      col = 0,
    },
    size = {
      width = width,
    },
    zindex = 160,
    border = {
      style = { "─", "─", "─", "", "─", "─", "─", "" },
      text = {
        top = opts.title or "",
        top_align = "left",
      },
    },
    win_options = {
      cursorline = false,
    },
  }

  local input = NuiInput(popup_options, {
    default_value = opts.default_value,
    on_submit = opts.on_confirm,
  })

  -- configure mappings
  for _, km in ipairs(opts.mappings or {}) do
    local action
    if km.action == "menu_confirm" then
      action = opts.on_confirm
    elseif km.action == "menu_close" then
      action = function() end
    end

    local map_opts = km.opts or { noremap = true, nowait = true }

    if action then
      input:map(km.mode, km.key, function()
        local line = vim.api.nvim_buf_get_lines(input.bufnr, 0, 1, false)[1]
        input:unmount()
        action(line)
      end, map_opts)
    end
  end

  input:mount()
end

return M
