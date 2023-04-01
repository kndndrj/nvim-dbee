-- save and restore window layout

---@alias layoutEgg { layout: any, restore: any }

-- layout structure:
-- { "row", { { "leaf", winid, bufnr }, { "col", { { "leaf", winid, bufnr }, { "leaf", winid, bufnr } } } } }

local M = {}

-- add bufnr to leaf
local function add_buf_to_layout(layout)
  if layout[1] == "leaf" then
    table.insert(layout, vim.fn.winbufnr(layout[2]))
    return layout
  else
    local children = {}
    for _, child_layout in ipairs(layout[2]) do
      table.insert(children, add_buf_to_layout(child_layout))
    end
    return { layout[1], children }
  end
end

---@return layoutEgg layout egg (use with restore())
function M.save()
  local layout = vim.fn.winlayout()
  local restore_cmd = vim.fn.winrestcmd()

  local buf_layout = add_buf_to_layout(layout)

  return { layout = buf_layout, restore = restore_cmd }
end

local function apply_layout(layout)
  if layout[1] == "leaf" then
    if vim.fn.bufexists(layout[3]) == 1 then
      vim.cmd("b " .. layout[3])
    end
  else
    -- split cols or rows, split n-1 times
    local split_method = "rightbelow vsplit"
    if layout[1] == "col" then
      split_method = "rightbelow split"
    end

    local wins = { vim.fn.win_getid() }

    for i in ipairs(layout[2]) do
      if i ~= 1 then
        vim.cmd(split_method)
        table.insert(wins, vim.fn.win_getid())
      end
    end

    -- recursive into child windows
    for index, win in ipairs(wins) do
      vim.fn.win_gotoid(win)
      apply_layout(layout[2][index])
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
  vim.cmd("wincmd o")
  local tmp_buf = vim.api.nvim_get_current_buf()

  -- apply layout and perform resize_cmd
  apply_layout(egg.layout)
  vim.cmd(egg.restore)

  -- delete temporary buffer
  vim.cmd("bd " .. tmp_buf)
end

return M
