local utils = require("dbee.utils")
local ui_helper = require("dbee.ui_helper")

-- events:
---@alias editor_event_name "note_state_changed"|"note_removed"|"note_created"|"current_note_changed"
---@alias editor_event_listener fun(data: any)

---@alias namespace_id "global"|string

---@alias note_id string
---@alias note_details { id: note_id, name: string, file: string, bufnr: integer? }

---@alias editor_config { directory: string, mappings: table<string, mapping> }

---@class Editor
---@field private handler Handler
---@field private result Result
---@field private quit_handle fun()
---@field private winid? integer
---@field private mappings table<string, mapping>
---@field private notes table<namespace_id, table<note_id, note_details>> namespace: { id: note_details } mapping
---@field private current_note_id? note_id
---@field private directory string directory where notes are stored
---@field private event_callbacks table<editor_event_name, editor_event_listener[]> callbacks for events
---@field private configured_autocmd_windows table<integer, boolean> windows that had autocommands configured.
local Editor = {}

---@param handler Handler
---@param result Result
---@param quit_handle? fun()
---@param opts? editor_config
---@return Editor
function Editor:new(handler, result, quit_handle, opts)
  opts = opts or {}

  if not handler then
    error("no Handler provided to Editor")
  end
  if not result then
    error("no Result provided to Editor")
  end

  -- class object
  ---@type Editor
  local o = {
    handler = handler,
    result = result,
    quit_handle = quit_handle or function() end,
    notes = {},
    event_callbacks = {},
    directory = opts.directory or vim.fn.stdpath("cache") .. "/dbee/notes",
    mappings = opts.mappings,
    configured_autocmd_windows = {},
  }
  setmetatable(o, self)
  self.__index = self

  -- search for existing notes
  o:search_existing_namespaces()

  return o
end

-- Look for existing namespaces and their notes on disk.
---@private
function Editor:search_existing_namespaces()
  -- search all directories (namespaces) and their notes
  for _, dir in pairs(vim.split(vim.fn.glob(self.directory .. "/*"), "\n")) do
    if vim.fn.isdirectory(dir) == 1 then
      for _, file in pairs(vim.split(vim.fn.glob(dir .. "/*"), "\n")) do
        if vim.fn.filereadable(file) == 1 then
          local namespace = vim.fs.basename(dir)
          local id = file .. tostring(os.clock())

          self.notes[namespace] = self.notes[namespace] or {}
          self.notes[namespace][id] = {
            id = id,
            name = vim.fs.basename(file),
            file = file,
          }
        end
      end
    end
  end
end

---@private
---@param mappings table<string, mapping>
---@return keymap[]
function Editor:generate_keymap(mappings)
  mappings = mappings or {}
  return {
    {
      action = function()
        if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
          return
        end
        local bufnr = vim.api.nvim_win_get_buf(self.winid)
        local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
        local query = table.concat(lines, "\n")

        local conn = self.handler:get_current_connection()
        if not conn then
          return
        end
        local call = self.handler:connection_execute(conn.id, query)
        self.result:set_call(call)
      end,
      mapping = mappings["run_file"],
    },
    {
      action = function()
        local srow, scol, erow, ecol = utils.visual_selection()

        local selection = vim.api.nvim_buf_get_text(0, srow, scol, erow, ecol, {})
        local query = table.concat(selection, "\n")

        local conn = self.handler:get_current_connection()
        if not conn then
          return
        end
        local call = self.handler:connection_execute(conn.id, query)
        self.result:set_call(call)
      end,
      mapping = mappings["run_selection"],
    },
  }
end

---@private
---@param event editor_event_name
---@param data any
function Editor:trigger_event(event, data)
  local cbs = self.event_callbacks[event] or {}
  for _, cb in ipairs(cbs) do
    cb(data)
  end
end

---@param event editor_event_name
---@param listener editor_event_listener
function Editor:register_event_listener(event, listener)
  self.event_callbacks[event] = self.event_callbacks[event] or {}
  table.insert(self.event_callbacks[event], listener)
end

---@private
---@param namespace string
---@return string
function Editor:dir(namespace)
  return self.directory .. "/" .. namespace
end

---@private
---@param id namespace_id
---@param name string name to check
---@return boolean # true - conflict, false - no conflict
function Editor:namespace_check_conflict(id, name)
  local notes = self.notes[id] or {}
  for _, note in pairs(notes) do
    if note.name == name then
      return true
    end
  end
  return false
end

---@private
---@param id note_id
---@return note_details?
---@return namespace_id namespace
function Editor:get_note(id)
  for namespace, per_namespace in pairs(self.notes) do
    for _, note in pairs(per_namespace) do
      if note.id == id then
        return note, namespace
      end
    end
  end
  return nil, ""
end

---@private
---@param bufnr integer
---@return note_details?
---@return namespace_id namespace
function Editor:get_note_by_buf(bufnr)
  for namespace, per_namespace in pairs(self.notes) do
    for _, note in pairs(per_namespace) do
      if note.bufnr and note.bufnr == bufnr then
        return note, namespace
      end
    end
  end
  return nil, ""
end

-- Creates a new note in namespace.
-- Errors if id or name is nil or there is a note with the same
-- name in namespace already.
---@param id namespace_id
---@param name string
---@return note_id
function Editor:namespace_create_note(id, name)
  local namespace = id
  if not namespace or namespace == "" then
    error("invalid namespace id")
  end
  if not name or name == "" then
    error("no name for global note")
  end

  if not vim.endswith(name, ".sql") then
    name = name .. ".sql"
  end

  -- create namespace directory
  vim.fn.mkdir(self:dir(namespace), "p")

  if self:namespace_check_conflict(namespace, name) then
    error('note with this name already exists in "' .. namespace .. '" namespace')
  end

  local file = self:dir(namespace) .. "/" .. name
  local note_id = file .. tostring(os.clock())
  ---@type note_details
  local s = {
    id = note_id,
    name = name,
    file = file,
  }

  self.notes[namespace] = self.notes[namespace] or {}
  self.notes[namespace][note_id] = s

  self:trigger_event("note_created", { note = s })

  return note_id
end

---@param id namespace_id
---@return note_details[]
function Editor:namespace_get_notes(id)
  local namespace = id
  if not namespace or namespace == "" then
    error("invalid namespace id")
  end

  local notes = vim.tbl_values(self.notes[namespace] or {})

  table.sort(notes, function(k1, k2)
    return k1.name < k2.name
  end)
  return notes
end

-- Removes an existing note.
-- Errors if there is no note with provided id in namespace.
---@param id namespace_id
---@param note_id note_id
function Editor:namespace_remove_note(id, note_id)
  local namespace = id
  if not self.notes[namespace] then
    error("invalid namespace id to remove the note from")
  end

  local note = self.notes[namespace][note_id]
  if not note then
    error("invalid note id to remove")
  end

  -- delete file
  vim.fn.delete(note.file)

  -- delete record
  self.notes[namespace][note_id] = nil

  self:trigger_event("note_removed", { note_id = note_id })
end

-- Renames an existing note.
-- Errors if no name or id provided, there is no note with provided id or
-- there is already an existing note with the same name in the same namespace.
---@param id note_id
---@param name string new name
function Editor:note_rename(id, name)
  local note, namespace = self:get_note(id)
  if not note then
    error("invalid note id to rename")
  end
  if not name or name == "" then
    error("invalid name")
  end

  if not vim.endswith(name, ".sql") then
    name = name .. ".sql"
  end

  if self:namespace_check_conflict(namespace, name) then
    error('note with this name already exists in "' .. namespace .. '" namespace')
  end

  local new_file = self:dir(namespace) .. "/" .. name

  -- rename file
  if vim.fn.filereadable(note.file) == 1 then
    vim.fn.rename(note.file, new_file)
  end

  -- rename buffer
  if note.bufnr and vim.api.nvim_buf_get_name(note.bufnr) == note.file then
    vim.api.nvim_buf_set_name(note.bufnr, new_file)
  end

  -- save changes
  self.notes[namespace][id].file = new_file
  self.notes[namespace][id].name = name

  self:trigger_event("note_state_changed", { note = self.notes[namespace][id] })
end

---@return note_details?
function Editor:get_current_note()
  local note, _ = self:get_note(self.current_note_id)
  return note
end

-- Sets note with id as the current note
-- and opens it in the window
---@param id note_id
function Editor:set_current_note(id)
  if id and self.current_note_id == id then
    self:display_note(id)
    return
  end

  local note, _ = self:get_note(id)
  if not note then
    error("invalid note set as current")
  end

  self.current_note_id = id

  self:display_note(id)

  self:trigger_event("current_note_changed", { note_id = id })
end

---@private
---@param id note_id
function Editor:display_note(id)
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    return
  end

  local note, namespace = self:get_note(id)
  if not note then
    return
  end

  -- if buffer is configured, just open it
  if note.bufnr and vim.api.nvim_buf_is_valid(note.bufnr) then
    vim.api.nvim_win_set_buf(self.winid, note.bufnr)
    vim.api.nvim_set_current_win(self.winid)
    return
  end

  -- otherwise open a file and update note's buffer
  vim.api.nvim_set_current_win(self.winid)
  vim.cmd("e " .. note.file)

  local bufnr = vim.api.nvim_get_current_buf()
  self.notes[namespace][id].bufnr = bufnr

  -- configure options on new buffer
  ui_helper.configure_buffer_options(bufnr, {
    buflisted = false,
    swapfile = false,
    filetype = "sql",
  })
end

---@private
---@param winid integer
function Editor:configure_autocommands(winid)
  if self.configured_autocmd_windows[winid] then
    return
  end

  -- remove current note if another buffer is opened in the window
  -- and set current note if any known note is opened in the window.
  utils.create_window_autocmd({ "BufWinEnter" }, winid, function(event)
    if not self.current_note_id then
      local note, _ = self:get_note_by_buf(event.buf)
      if note then
        self.current_note_id = note.id
        self:trigger_event("current_note_changed", { note_id = note.id })
      end
      return
    end

    local note, _ = self:get_note(self.current_note_id)
    if not note then
      self.current_note_id = nil
      self:trigger_event("current_note_changed", { note_id = nil })
      return
    end

    if not note.bufnr or note.bufnr ~= event.buf then
      self.current_note_id = nil
      self:trigger_event("current_note_changed", { note_id = nil })
      return
    end
  end)

  self.configured_autocmd_windows[winid] = true
end

---@private
---@return note_id
function Editor:create_welcome_note()
  local note_id = self:namespace_create_note("global", "welcome")
  local note = self:get_note(note_id)
  if not note then
    error("failed creating welcome note")
  end

  -- TODO: nicer text
  local text = {
    "",
    "",
    "",
    "Welcome to DBee!",
  }

  -- create note buffer with contents
  local bufnr = vim.api.nvim_create_buf(false, false)
  vim.api.nvim_buf_set_name(bufnr, note.file)
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, true, text)
  vim.api.nvim_buf_set_option(bufnr, "modified", false)

  self.notes["global"][note_id].bufnr = bufnr

  -- remove all text when first change happens to text
  vim.api.nvim_create_autocmd({ "InsertEnter" }, {
    once = true,
    buffer = bufnr,
    callback = function()
      vim.api.nvim_buf_set_lines(bufnr, 0, -1, true, {})
      vim.api.nvim_buf_set_option(bufnr, "modified", false)
    end,
  })

  return note_id
end

---@param winid integer
function Editor:show(winid)
  self.winid = winid
  self:configure_autocommands(winid)

  ui_helper.configure_window_mappings(winid, self:generate_keymap(self.mappings))
  ui_helper.configure_window_quit_handle(winid, self.quit_handle)

  -- open current note if configured
  if self.current_note_id then
    self:display_note(self.current_note_id)
    return
  end

  -- else show the first global note
  local notes = self:namespace_get_notes("global")
  if not vim.tbl_isempty(notes) then
    self:set_current_note(notes[1].id)
    return
  end

  -- otherwise create a welcome note in global namespace
  local note_id = self:create_welcome_note()
  self:set_current_note(note_id)
end

return Editor
