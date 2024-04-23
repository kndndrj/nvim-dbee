---@alias _layout { type: string, winid: integer, bufnr: integer, win_opts: { string: any}, children: _layout[] }

---@alias layout_egg { layout: _layout, restore: string }

-- vim.fn.winlayout() example structure:
-- { "row", { { "leaf", winid }, { "col", { { "leaf", winid }, { "leaf", winid } } } } }

local M = {}

-- list all non-floating windows from the current tabpage
---@return integer[] # list of non floating window ids
local function list_non_floating_wins()
  return vim.fn.filter(vim.api.nvim_tabpage_list_wins(vim.api.nvim_get_current_tabpage()), function(_, v)
    return vim.api.nvim_win_get_config(v).relative == ""
  end)
end

-- makes the window the only one on screen
-- same as ":only" except ignores floating windows
---@param winid integer
function M.make_only(winid)
  if not winid or winid == 0 then
    winid = vim.api.nvim_get_current_win()
  end

  for _, wid in ipairs(list_non_floating_wins()) do
    if wid ~= winid then
      local winnr = vim.fn.win_id2win(wid)
      vim.cmd(winnr .. "wincmd c")
    end
  end
end

-- exact clone of the builtin "winrestcmd()" with exclusion of floating windows
-- https://github.com/neovim/neovim/blob/fcf3519c65a2d6736de437f686e788684a6c8564/src/nvim/eval/window.c#L770
---@return string
local function winrestcmd()
  local cmd = ""

  -- Do this twice to handle some window layouts properly.
  for _ = 1, 2 do
    local winnr = 1
    for _, winid in ipairs(list_non_floating_wins()) do
      cmd = string.format("%s%dresize %d|", cmd, winnr, vim.api.nvim_win_get_height(winid))
      cmd = string.format("%svert %dresize %d|", cmd, winnr, vim.api.nvim_win_get_width(winid))
      winnr = winnr + 1
    end
  end

  return cmd
end

-- add bufnr to leaf
local function add_details(layout)
  if layout[1] == "leaf" then
    local win = layout[2]

    -- window options
    local all_options = vim.api.nvim_get_all_options_info()
    local v = vim.wo[win]
    local options = {}
    for key, val in pairs(all_options) do
      if val.global_local == false and val.scope == "win" then
        options[key] = v[key]
      end
    end

    -- create dict structure with added buffer and window opts
    ---@type _layout
    local l = {
      type = layout[1],
      winid = win,
      bufnr = vim.fn.winbufnr(win),
      win_opts = options,
    }
    return l
  else
    local children = {}
    for _, child_layout in ipairs(layout[2]) do
      table.insert(children, add_details(child_layout))
    end
    return { type = layout[1], children = children }
  end
end

---@return layout_egg layout egg (use with restore())
function M.save()
  local layout = vim.fn.winlayout()
  local restore_cmd = winrestcmd()

  layout = add_details(layout)

  return { layout = layout, restore = restore_cmd }
end

---@param layout _layout
local function apply_layout(layout)
  if layout.type == "leaf" then
    -- open the previous buffer
    if vim.fn.bufexists(layout.bufnr) == 1 then
      vim.cmd("b " .. layout.bufnr)
    end
    -- apply window options
    for opt, val in pairs(layout.win_opts) do
      if val ~= nil then
        vim.wo[opt] = val
      end
    end
  else
    -- split cols or rows, split n-1 times
    local split_method = "rightbelow vsplit"
    if layout.type == "col" then
      split_method = "rightbelow split"
    end

    local wins = { vim.fn.win_getid() }

    for i in ipairs(layout.children) do
      if i ~= 1 then
        vim.cmd(split_method)
        table.insert(wins, vim.fn.win_getid())
      end
    end

    -- recursive into child windows
    for index, win in ipairs(wins) do
      vim.fn.win_gotoid(win)
      apply_layout(layout.children[index])
    end
  end
end

---@param egg layout_egg layout to restore
function M.restore(egg)
  egg = egg or {}

  if not egg.layout or not egg.restore then
    return
  end

  -- make a new window and set it as the only one
  vim.cmd("new")
  M.make_only(0)
  local tmp_buf = vim.api.nvim_get_current_buf()

  -- apply layout and perform resize_cmd
  apply_layout(egg.layout)
  vim.cmd(egg.restore)

  -- delete temporary buffer
  vim.cmd("bd " .. tmp_buf)
end

return M
