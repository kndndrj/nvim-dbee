local utils = require("dbee.utils")

---@alias scratch_id string
---@alias scratch_details { file: string, bufnr: integer, type: "file"|"buffer", id: scratch_id }

local SCRATCHES_DIR = vim.fn.stdpath("cache") .. "/dbee/scratches"

---@class Editor
---@field private handler Handler
---@field private scratches table<scratch_id, scratch_details> id - scratch mapping
---@field private active_scratch scratch_id id of the current scratch
---@field private winid integer
---@field private win_cmd fun():integer function which opens a new window and returns a window id
local Editor = {}

---@param opts? { handler: Handler, win_cmd: string | fun():integer }
---@return Editor
function Editor:new(opts)
  opts = opts or {}

  if not opts.handler then
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
  local active = ""
  for _, file in pairs(vim.split(vim.fn.glob(SCRATCHES_DIR .. "/*"), "\n")) do
    if file ~= "" then
      local id = file .. tostring(os.clock())
      scratches[id] = { id = id, file = file, type = "file", bufnr = nil }
      active = id
    end
  end
  if vim.tbl_isempty(scratches) then
    local file = SCRATCHES_DIR .. "/scratch." .. tostring(os.clock()) .. ".sql"
    local id = file .. tostring(os.clock())
    scratches[id] = { id = id, file = file, type = "file", bufnr = nil }
    active = id
  end

  -- class object
  local o = {
    handler = opts.handler,
    winid = nil,
    scratches = scratches,
    active_scratch = active,
    win_cmd = win_cmd,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

function Editor:new_scratch()
  local file = SCRATCHES_DIR .. "/scratch." .. tostring(os.clock()) .. ".sql"
  local id = file .. tostring(os.clock())
  ---@type scratch_details
  local s = {
    file = file,
    id = id,
    type = "buffer",
  }
  self.scratches[id] = s
  self.active_scratch = id
end

---@param id scratch_id scratch id
---@param name string new name
function Editor:rename_scratch(id, name)
  if not id or not self.scratches[id] then
    error("invalid id to rename")
  end
  if not name or name == "" then
    error("invalid name")
  end

  local s = self.scratches[id]
  local bufnr = s.bufnr
  local file = s.file

  -- rename file
  if vim.fn.filereadable(file) then
    vim.fn.rename(file, name)
  end
  self.scratches[id].file = name

  -- rename buffer
  if bufnr then
    if vim.api.nvim_buf_get_name(bufnr) == file then
      vim.api.nvim_buf_set_name(bufnr, name)
    elseif vim.api.nvim_buf_get_name(bufnr) == vim.fs.basename(file) then
      vim.api.nvim_buf_set_name(bufnr, vim.fs.basename(name))
    end
  end
end

---@param id scratch_id scratch id
function Editor:delete_scratch(id)
  if not id or not self.scratches[id] then
    error("invalid id to delete")
  end

  local file = self.scratches[id].file

  -- delete file
  if vim.fn.filereadable(file) then
    vim.fn.delete(file)
  end
  -- delete record
  self.scratches[id] = nil

  -- open a different scratchpad
  local id_other = utils.random_key(self.scratches)
  if not id_other then
    self:new_scratch()
    self:open()
    return
  end
  self:set_active_scratch(id_other)
  self:open()
end

-- get layout of scratchpads
---@return Layout[]
function Editor:layout()
  ---@type Layout[]
  local scratches = {
    {
      name = "- new -",
      action_1 = function(cb)
        self:new_scratch()
        self:open()
        cb()
      end,
    },
  }

  for _, s in pairs(self.scratches) do
    ---@type Layout
    local sch = {
      name = vim.fs.basename(s.file),
      type = "scratch",
      action_1 = function(cb)
        self:set_active_scratch(s.id)
        self:open()
        cb()
      end,
      action_2 = function(cb)
        local file = self.scratches[s.id].file
        vim.ui.input({ prompt = "new name: ", default = file }, function(input)
          if not input or input == "" then
            return
          end
          self:rename_scratch(s.id, input)
          cb()
        end)
      end,
      action_3 = function(cb)
        local file = self.scratches[s.id].file
        vim.ui.input({ prompt = 'confirm deletion of "' .. file .. '"', default = "Y" }, function(input)
          if not input or string.lower(input) ~= "y" then
            return
          end
          self:delete_scratch(s.id)
          cb()
        end)
      end,
    }
    table.insert(scratches, sch)
  end

  return scratches
end

---@param id scratch_id scratch id - name
function Editor:set_active_scratch(id)
  if not id or not self.scratches[id] then
    error("no id specified!")
  end
  self.active_scratch = id
end

---TODO
---@private
function Editor:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }

  -- run the whole file
  vim.keymap.set("n", "BB", function()
    local bnr = self.scratches[self.active_scratch].bufnr
    local lines = vim.api.nvim_buf_get_lines(bnr, 0, -1, false)
    local query = table.concat(lines, "\n")

    self.handler:execute(query)
  end, map_options)

  -- run selection
  vim.keymap.set("v", "BB", function()
    local srow, scol, erow, ecol = utils.visual_selection()

    local selection = vim.api.nvim_buf_get_text(0, srow, scol, erow, ecol, {})
    local query = table.concat(selection, "\n")

    self.handler:execute(query)
  end, map_options)
end

---@param winid? integer
function Editor:open(winid)
  winid = winid or self.winid
  if not winid or not vim.api.nvim_win_is_valid(winid) then
    winid = self.win_cmd()
  end

  vim.api.nvim_set_current_win(winid)

  -- get current scratch details
  local id = self.active_scratch
  local s = self.scratches[id]
  if not s then
    error("no scratchpad selected")
  end

  local bufnr
  -- if file doesn't exist, open new buffer and update list on save
  if vim.fn.filereadable(s.file) ~= 1 then
    bufnr = s.bufnr or vim.api.nvim_create_buf(true, false)
    vim.api.nvim_win_set_buf(winid, bufnr)

    -- automatically fill the name of the file when saving for the first time
    vim.keymap.set("c", "w", function()
      return "w " .. self.scratches[id].file
    end, { noremap = true, nowait = true, buffer = bufnr, expr = true })
    vim.api.nvim_create_autocmd("BufWritePost", {
      once = true,
      callback = function()
        -- remove mapping and update filename on write
        pcall(vim.keymap.del, "c", "w", { buffer = bufnr })

        -- it's possible that multiple autocmds get registered
        if not self.scratches[id] then
          return
        end
        local n = vim.api.nvim_buf_get_name(bufnr)
        if n and n ~= "" then
          self.scratches[id].file = vim.api.nvim_buf_get_name(bufnr)
          self.scratches[id].type = "file"
        end
      end,
    })
  else
    -- just open the file
    bufnr = s.bufnr or vim.api.nvim_create_buf(true, false)
    vim.api.nvim_win_set_buf(winid, bufnr)
    vim.cmd("e " .. s.file)
  end

  -- set keymaps
  self:map_keys(bufnr)

  self.winid = winid
  self.scratches[self.active_scratch].bufnr = bufnr

  -- set options
  local buf_opts = {
    buflisted = false,
    swapfile = false,
    filetype = "sql",
  }
  for opt, val in pairs(buf_opts) do
    vim.api.nvim_buf_set_option(bufnr, opt, val)
  end
end

function Editor:close()
  vim.api.nvim_win_close(self.winid, false)
end

return Editor
