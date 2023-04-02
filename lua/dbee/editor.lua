local utils = require("dbee.utils")

local SCRATCHES_DIR = vim.fn.stdpath("cache") .. "/dbee/scratches"

---@class Editor
---@field private handler Handler
---@field private ui_opts { win_cmd: string, winid: integer}
---@field private scratches string[] list of scratch files
---@field current_scratch integer id of the current scratch
local Editor = {}

---@param opts? { handler: Handler, win_cmd: string }
---@return Editor|nil
function Editor:new(opts)
  opts = opts or {}

  if opts.handler == nil then
    print("no Handler provided to editor")
    return
  end

  -- check for any existing scratches
  vim.fn.mkdir(SCRATCHES_DIR, "p")
  local scratches = {}
  for _, file in pairs(vim.split(vim.fn.glob(SCRATCHES_DIR .. "/*"), "\n")) do
    if file ~= "" then
      table.insert(scratches, file)
    end
  end
  if #scratches == 0 then
    table.insert(scratches, SCRATCHES_DIR .. "/scratch." .. tostring(os.clock()) .. ".sql")
  end

  -- class object
  local o = {
    handler = opts.handler,
    ui_opts = {
      win_cmd = opts.win_cmd or "vsplit",
      winid = nil,
    },
    scratches = scratches,
    current_scratch = 1,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

function Editor:execute_selection()
  local srow, scol, erow, ecol = utils.visual_selection()

  local selection = vim.api.nvim_buf_get_text(0, srow, scol, erow, ecol, {})
  local query = table.concat(selection, "\n")

  self.handler:execute(query)
end

function Editor:new_scratch()
  table.insert(self.scratches, SCRATCHES_DIR .. "/scratch." .. tostring(os.clock()) .. ".sql")
  self.current_scratch = #self.scratches
end

---@return string[]
function Editor:list_scratches()
  return self.scratches
end

---@param id integer scratch id
function Editor:set_active_scratch(id)
  if type(id) == "number" and id > 0 then
    self.current_scratch = id
  end
end

---TODO
---@private
function Editor:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }
end

---@param winid? integer if provided, use it instead of creating new window
function Editor:open(winid)
  -- if window doesn't exist, create it
  winid = winid or self.ui_opts.winid
  if not winid or not vim.api.nvim_win_is_valid(winid) then
    vim.cmd(self.ui_opts.win_cmd)
    winid = vim.api.nvim_get_current_win()
  end

  -- open the file
  vim.api.nvim_set_current_win(winid)

  local s = self.scratches[self.current_scratch]
  local bufnr
  -- if file doesn't exist, open new buffer and update list on save
  if vim.fn.filereadable(s) ~= 1 then
    bufnr = vim.api.nvim_create_buf(true, false)
    vim.api.nvim_win_set_buf(winid, bufnr)
    -- automatically fill the name of the file when saving for the first time
    vim.keymap.set("c", "w", "w " .. s, { noremap = true, nowait = true, buffer = bufnr })
    vim.api.nvim_create_autocmd("BufWritePost", {
      once = true,
      callback = function()
        -- remove mapping and update filename on write
        vim.keymap.del("c", "w", { buffer = bufnr })
        self.scratches[self.current_scratch] = vim.api.nvim_buf_get_name(bufnr)
      end,
    })
  else
    -- just open the file
    vim.cmd("e " .. s)
    bufnr = vim.api.nvim_get_current_buf()
  end

  -- set keymaps
  self:map_keys(bufnr)

  self.ui_opts.winid = winid

  -- set options
  local buf_opts = {
    buflisted = false,
    swapfile = false,
  }
  for opt, val in pairs(buf_opts) do
    vim.api.nvim_buf_set_option(bufnr, opt, val)
  end
end

function Editor:close()
  vim.api.nvim_win_close(self.ui_opts.winid, false)
end

return Editor
