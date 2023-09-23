local NuiLine = require("nui.line")
local NuiTree = require("nui.tree")

local M = {}

local m = {}

---@param str string
---@param len integer
---@return string # string of length
local function make_length(str, len)
  if #str > len then
    return str:sub(1, len - 1) .. "…"
  elseif #str < len then
    return str .. string.rep(" ", len - #str)
  end

  -- same length
  return str
end

---@param state call_state
---@return string # highlight group
local function call_state_highlight(state)
  if state == "unknown" then
    return "NonText"
  elseif state == "executing" then
    return "WarningMsg"
  elseif state == "cached" then
    return "String"
  elseif state == "archived" then
    return "Title"
  elseif state == "failed" then
    return "Error"
  end

  return ""
end

---@param bufnr integer
---@return table tree
local function create_tree(bufnr)
  m.tree = m.tree
    or NuiTree {
      bufnr = bufnr,
      prepare_node = function(node)
        ---@type call_details
        local call = node.call

        local line = NuiLine()
        line:append(make_length(call.state, 15), call_state_highlight(call.state))
        line:append(" | ", "NonText")
        line:append(make_length(call.query, 40))
        line:append(" | ", "NonText")
        line:append(tostring(os.date("%c", (call.timestamp_us or 0) / 1000000)))
        return line
      end,
      get_node_id = function(node)
        if node.id then
          return node.id
        end
        return tostring(math.random())
      end,
    }

  m.tree.bufnr = bufnr

  return m.tree
end

---@param call_getter fun():call_details[]
---@param opts? { width: integer, height: integer, title: string, border: string|string[], on_select: fun(call: call_details), on_cancel: fun(call: call_details) } optional parameters
function M.call_log(call_getter, opts)
  opts = opts or {}

  local on_select = opts.on_select or function(_) end
  local on_cancel = opts.on_cancel or function(_) end

  local ui_spec = vim.api.nvim_list_uis()[1]
  local win_width = opts.width or 100
  local win_height = opts.height or 20
  local x = math.floor((ui_spec["width"] - win_width) / 2)
  local y = math.floor((ui_spec["height"] - win_height) / 2)

  -- create new buffer
  local bufnr = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_option(bufnr, "bufhidden", "delete")

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
  vim.api.nvim_win_set_option(winid, "cursorline", true)

  local tree = create_tree(bufnr)

  -- this function is called by a scheduled task
  local function refresher()
    -- create nodes from calls
    local nodes = {}
    local calls = call_getter()
    table.sort(calls, function(k1, k2)
      return k1.timestamp_us > k2.timestamp_us
    end)
    for _, c in ipairs(calls) do
      table.insert(nodes, NuiTree.Node { id = tostring(math.random()), call = c })
    end

    tree:set_nodes(nodes)
    tree:render()
  end

  -- manual call for the first time
  refresher()

  -- call the function every n milliseconds
  local timer = vim.fn.timer_start(500, refresher, { ["repeat"] = -1 })
  local timer_stop = function()
    pcall(vim.fn.timer_stop, timer)
  end

  vim.api.nvim_create_autocmd({ "BufLeave" }, {
    buffer = bufnr,
    callback = function()
      timer_stop()
      pcall(vim.api.nvim_win_close, winid, true)
      pcall(vim.api.nvim_buf_delete, bufnr, {})
    end,
  })

  -- set keymaps
  vim.keymap.set("n", "q", function()
    timer_stop()
    pcall(vim.api.nvim_win_close, winid, true)
  end, { silent = true, buffer = bufnr })

  vim.keymap.set("n", "<CR>", function()
    local node = tree:get_node()
    if not node then
      return
    end
    on_select(node.call)
  end, { silent = true, buffer = bufnr })

  vim.keymap.set("n", "d", function()
    local node = tree:get_node()
    if not node then
      return
    end
    on_cancel(node.call)
  end, { silent = true, buffer = bufnr })
end

return M
