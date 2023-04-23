-- save and restore window layout

---@alias _layout { type: string, winid: integer, bufnr: integer, win_opts: { string: any}, children: _layout[] }

---@alias layoutEgg { layout: _layout, restore: any }

-- vim.fn.winlayout() example structure:
-- { "row", { { "leaf", winid }, { "col", { { "leaf", winid }, { "leaf", winid } } } } }

local M = {}

-- closes all currently open floating windows
local function close_all_floating()
  local closed_windows = {}
  for _, win in ipairs(vim.api.nvim_list_wins()) do
    local config = vim.api.nvim_win_get_config(win)
    if config.relative ~= "" then
      vim.api.nvim_win_close(win, false)
      table.insert(closed_windows, win)
    end
  end
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

---@return layoutEgg layout egg (use with restore())
function M.save()
  -- close any floating windows first
  close_all_floating()

  local layout = vim.fn.winlayout()
  local restore_cmd = vim.fn.winrestcmd()

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

---@param egg layoutEgg layout to restore
function M.restore(egg)
  egg = egg or {}

  if not egg.layout or not egg.restore then
    return
  end

  -- make a new window and set it as the only one
  vim.cmd("new")
  vim.cmd("only!")
  local tmp_buf = vim.api.nvim_get_current_buf()

  -- apply layout and perform resize_cmd
  apply_layout(egg.layout)
  vim.cmd(egg.restore)

  -- delete temporary buffer
  vim.cmd("bd " .. tmp_buf)
end

return M
