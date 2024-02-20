local utils = require("dbee.utils")
local progress = require("dbee.ui.result.progress")
local common = require("dbee.ui.common")

-- ResultTile represents the part of ui with displayed results
---@class ResultUI
---@field private handler Handler
---@field private winid? integer
---@field private bufnr integer
---@field private current_call? CallDetails
---@field private page_size integer
---@field private mappings key_mapping[]
---@field private page_index integer index of the current page
---@field private page_ammount integer number of pages in the current result set
---@field private stop_progress fun() function that stops progress display
---@field private progress_opts progress_config
---@field private switch_handle fun(bufnr: integer)
local ResultTile = {}

---@param handler Handler
---@param quit_handle? fun()
---@param switch_handle? fun(bufnr: integer)
---@param opts? result_config
---@return ResultUI
function ResultTile:new(handler, quit_handle, switch_handle, opts)
  opts = opts or {}
  quit_handle = quit_handle or function() end

  if not handler then
    error("no Handler passed to ResultTile")
  end

  -- class object
  local o = {
    handler = handler,
    page_size = opts.page_size or 100,
    page_index = 0,
    page_ammount = 0,
    mappings = opts.mappings or {},
    stop_progress = function() end,
    progress_opts = opts.progress or {},
    switch_handle = switch_handle or function() end,
  }
  setmetatable(o, self)
  self.__index = self

  -- create a buffer for drawer and configure it
  o.bufnr = common.create_blank_buffer("dbee-result", {
    buflisted = false,
    bufhidden = "delete",
    buftype = "nofile",
    swapfile = false,
    modifiable = false,
  })
  common.configure_buffer_mappings(o.bufnr, o:get_actions(), opts.mappings)
  common.configure_buffer_quit_handle(o.bufnr, quit_handle)

  handler:register_event_listener("call_state_changed", function(data)
    o:on_call_state_changed(data)
  end)

  return o
end

-- event listener for new calls
---@private
---@param data { call: CallDetails }
function ResultTile:on_call_state_changed(data)
  local call = data.call

  -- we only care about the current call
  if not self.current_call or call.id ~= self.current_call.id then
    return
  end

  -- update the current call with up to date details
  self.current_call = call

  -- perform action based on the state
  if call.state == "executing" then
    self.stop_progress()
    self:display_progress()
  elseif call.state == "retrieving" then
    self.stop_progress()
    self:page_current()
  elseif call.state == "executing_failed" or call.state == "retrieving_failed" or call.state == "canceled" then
    self.stop_progress()
    self:display_status()
  else
    self.stop_progress()
  end
end

---@private
function ResultTile:apply_highlight(winid)
  -- switch to provided window, apply hightlight and jump back
  local current_win = vim.api.nvim_get_current_win()
  vim.api.nvim_set_current_win(winid)
  -- match table separators and leading row numbers
  vim.cmd([[match NonText /^\s*\d\+\|─\|│\|┼/]])
  vim.api.nvim_set_current_win(current_win)
end

---@private
function ResultTile:display_progress()
  self.stop_progress = progress.display(self.bufnr, self.progress_opts)

  vim.api.nvim_set_current_win(self.winid)
end

---@private
function ResultTile:display_status()
  if not self.current_call then
    error("no call set to result")
  end

  local state = self.current_call.state

  local msg = ""
  if state == "executing_failed" then
    msg = "Call execution failed"
  elseif state == "retrieving_failed" then
    msg = "Failed retrieving results"
  elseif state == "canceled" then
    msg = "Call canceled"
  end

  local seconds = self.current_call.time_taken_us / 1000000

  local lines = {
    string.format("%s after %.3f seconds", msg, seconds),
  }

  if self.current_call.error and self.current_call.error ~= "" then
    table.insert(lines, "Reason:")
    table.insert(lines, "    " .. string.gsub(self.current_call.error, "\n", " "))
  end

  vim.api.nvim_buf_set_option(self.bufnr, "modifiable", true)
  vim.api.nvim_buf_set_lines(self.bufnr, 0, -1, false, lines)

  vim.api.nvim_buf_set_option(self.bufnr, "modifiable", false)

  -- set winbar
  vim.api.nvim_win_set_option(self.winid, "winbar", "Results")

  -- reset modified flag
  vim.api.nvim_buf_set_option(self.bufnr, "modified", false)

  vim.api.nvim_set_current_win(self.winid)
end

--- Displays a page of the current result in the results buffer
---@private
---@param page integer zero based page index
---@return integer # current page
function ResultTile:display_result(page)
  if not self.current_call then
    error("no call set to result")
  end
  -- calculate the ranges
  if page < 0 then
    page = 0
  end
  if page > self.page_ammount then
    page = self.page_ammount
  end
  local from = self.page_size * page
  local to = self.page_size * (page + 1)

  -- call go function
  local length = self.handler:call_display_result(self.current_call.id, self.bufnr, from, to)

  -- adjust page ammount
  self.page_ammount = math.floor(length / self.page_size)
  if length % self.page_size == 0 and self.page_ammount ~= 0 then
    self.page_ammount = self.page_ammount - 1
  end

  -- convert from microseconds to seconds
  local seconds = self.current_call.time_taken_us / 1000000

  -- set winbar status
  if self.winid and vim.api.nvim_win_is_valid(self.winid) then
    vim.api.nvim_win_set_option(
      self.winid,
      "winbar",
      string.format("%d/%d%%=Took %.3fs", page + 1, self.page_ammount + 1, seconds)
    )
  end

  -- reset modified flag and set focus
  vim.api.nvim_buf_set_option(self.bufnr, "modified", false)
  vim.api.nvim_set_current_win(self.winid)

  return page
end

---@private
---@return table<string, fun()>
function ResultTile:get_actions()
  return {
    page_next = function()
      self:page_next()
    end,
    page_prev = function()
      self:page_prev()
    end,
    page_last = function()
      self:page_last()
    end,
    page_first = function()
      self:page_first()
    end,

    -- yank functions
    yank_current_json = function()
      self:store_current_wrapper("json", vim.v.register)
    end,
    yank_selection_json = function()
      self:store_selection_wrapper("json", vim.v.register)
    end,
    yank_all_json = function()
      self:store_all_wrapper("json", vim.v.register)
    end,
    yank_current_csv = function()
      self:store_current_wrapper("csv", vim.v.register)
    end,
    yank_selection_csv = function()
      self:store_selection_wrapper("csv", vim.v.register)
    end,
    yank_all_csv = function()
      self:store_all_wrapper("csv", vim.v.register)
    end,

    cancel_call = function()
      if self.current_call then
        self.handler:call_cancel(self.current_call.id)
      end
    end,
  }
end

-- sets call's result to Result's buffer
---@param call CallDetails
function ResultTile:set_call(call)
  self.page_index = 0
  self.page_ammount = 0
  self.current_call = call

  self.stop_progress()
end

-- Gets the currently displayed call.
---@return CallDetails?
function ResultTile:get_call()
  return self.current_call
end

function ResultTile:page_current()
  self.page_index = self:display_result(self.page_index)
end

function ResultTile:page_next()
  self.page_index = self:display_result(self.page_index + 1)
end

function ResultTile:page_prev()
  self.page_index = self:display_result(self.page_index - 1)
end


function ResultTile:page_last()
  self.page_index = self:display_result(self.page_ammount)
end

function ResultTile:page_first()
  self.page_index = self:display_result(0)
end

-- wrapper for storing the current row
---@private
---@param format string
---@param register string
function ResultTile:store_current_wrapper(format, register)
  if not self.current_call then
    error("no call set to result")
  end
  local index = self:current_row_index()

  -- indexes in table start with 1, but in go they start with 0,
  -- to correct this, we subtract 1 from sindex and eindex.
  -- Since range select [:] in go is exclusive for the upper bound, we additionally add 1 to eindex
  index = index - 1
  if index <= 0 then
    index = 0
  end

  self.handler:call_store_result(
    self.current_call.id,
    format,
    "yank",
    { from = index, to = index + 1, extra_arg = register }
  )
end

-- wrapper for storing the current visualy selected rows
---@private
---@param format string
---@param register string
function ResultTile:store_selection_wrapper(format, register)
  if not self.current_call then
    error("no call set to result")
  end
  local sindex, eindex = self:current_row_range()

  -- see above comment
  sindex = sindex - 1
  if sindex <= 0 then
    sindex = 0
  end

  self.handler:call_store_result(
    self.current_call.id,
    format,
    "yank",
    { from = sindex, to = eindex, extra_arg = register }
  )
end

-- wrapper for storing all rows
---@private
---@param format string
---@param register string
function ResultTile:store_all_wrapper(format, register)
  if not self.current_call then
    error("no call set to result")
  end
  self.handler:call_store_result(self.current_call.id, format, "yank", { extra_arg = register })
end

---@private
---@return number # index of the current row
function ResultTile:current_row_index()
  -- get position of the current line identifier
  local row = vim.fn.search([[^\s*[0-9]\+]], "bnc", 1)
  if row == 0 then
    error("couldn't retrieve current row number: row = 0")
  end

  -- get the line and extract the line number
  local line = vim.api.nvim_buf_get_lines(self.bufnr, row - 1, row, true)[1] or ""

  local index = line:match("%d+")
  if not index then
    error("couldn't retrieve current row number")
  end
  return index
end

---@private
---@return number # number of the first row
---@return number # number of the last row
function ResultTile:current_row_range()
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    error("result cannot operate without a valid window")
  end
  -- get current selection
  local srow, _, erow, _ = utils.visual_selection()

  srow = srow + 1
  erow = erow + 1

  -- save cursor position
  local cursor_position = vim.fn.getcurpos(self.winid)

  -- reposition the cursor
  vim.fn.cursor(srow, 1)
  -- get position of the start line identifier
  local row = vim.fn.search([[^\s*[0-9]\+]], "bnc", 1)
  if row == 0 then
    error("couldn't retrieve start row number: row = 0")
  end

  -- get the selected line and extract the line number
  local line = vim.api.nvim_buf_get_lines(self.bufnr, row - 1, row, true)[1] or ""

  local index_start = line:match("%d+")
  if not index_start then
    error("couldn't retrieve start row number")
  end

  -- reposition the cursor
  vim.fn.cursor(erow, 1)
  -- get position of the end line identifier
  row = vim.fn.search([[^\s*[0-9]\+]], "bnc", 1)
  if row == 0 then
    error("couldn't retrieve end row number: row = 0")
  end
  -- get the selected line and extract the line number
  line = vim.api.nvim_buf_get_lines(self.bufnr, row - 1, row, true)[1] or ""

  local index_end = tonumber(line:match("%d+"))
  if not index_end then
    error("couldn't retrieve end row number")
  end

  -- restore cursor position
  vim.fn.setpos(".", cursor_position)

  return index_start, index_end
end

---@param winid integer
function ResultTile:show(winid)
  self.winid = winid

  -- configure window options
  common.configure_window_options(self.winid, {
    wrap = false,
    winfixheight = true,
    winfixwidth = true,
    number = false,
    relativenumber = false,
    spell = false,
  })

  -- configure window highlights
  self:apply_highlight(self.winid)

  vim.api.nvim_win_set_buf(self.winid, self.bufnr)

  common.configure_buffer_options(self.bufnr, {
    buflisted = false,
    bufhidden = "delete",
    buftype = "nofile",
    swapfile = false,
    modifiable = false,
  })
  common.configure_buffer_mappings(self.bufnr, self:get_actions(), self.mappings)

  -- configure window immutablity
  common.configure_window_immutable_buffer(self.winid, self.bufnr, self.switch_handle)

  -- display the current result
  local ok = pcall(self.page_current, self)
  if not ok then
    vim.api.nvim_win_set_option(self.winid, "winbar", "Results")
  end
end

return ResultTile
