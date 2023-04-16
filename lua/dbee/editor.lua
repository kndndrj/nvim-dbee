local utils = require("dbee.utils")

local SCRATCHES_DIR = vim.fn.stdpath("cache") .. "/dbee/scratches"

---@class Editor
---@field private handler Handler
---@field private scratches string[] list of scratch files
---@field private current_scratch integer id of the current scratch
---@field private winid integer
---@field private win_cmd fun():integer function which opens a new window and returns a window id
local Editor = {}

---@param opts? { handler: Handler, win_cmd: string | fun():integer }
---@return Editor
function Editor:new(opts)
  opts = opts or {}

  if opts.handler == nil then
    error("no Handler provided to editor")
  end

  local win_cmd
  if type(opts.win_cmd) == "string" then
    win_cmd = function()
      vim.cmd(opts.win_cmd)
      return vim.api.nvim_get_current_win()
    end
  elseif type(opts.win_cmd) == "function" then
    win_cmd = opts.win_cmd
  else
    win_cmd = function()
      vim.cmd("split")
      return vim.api.nvim_get_current_win()
    end
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
    winid = nil,
    scratches = scratches,
    current_scratch = 1,
    win_cmd = win_cmd,
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

-- get layout of scratchpads
---@return layout[]
function Editor:layout()
  ---@type layout[]
  local scratches = {}

  for _, s in ipairs(self.scratches) do
    ---@type layout
    local sch = {
      name = s,
      type = "scratch",
    }
    table.insert(scratches, sch)
  end

  return scratches
end

---@param id string scratch id - name
function Editor:set_active_scratch(id)
  local rev_lookup = {}
  for i, s in ipairs(self.scratches) do
    rev_lookup[s] = i
  end
  self.current_scratch = rev_lookup[id]
end

---TODO
---@private
function Editor:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }
end

---@param winid? integer
function Editor:open(winid)
  winid = winid or self.winid
  if not winid or not vim.api.nvim_win_is_valid(winid) then
    winid = self.win_cmd()
  end

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

  self.winid = winid

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
  vim.api.nvim_win_close(self.winid, false)
end

return Editor
