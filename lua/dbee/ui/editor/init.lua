local utils = require("dbee.utils")
local common = require("dbee.ui.common")
local welcome = require("dbee.ui.editor.welcome")

---@alias namespace_id "global"|string

---@alias note_id string
---@alias note_details { id: note_id, name: string, file: string, bufnr: integer? }

---@class EditorUI
---@field private handler Handler
---@field private result ResultUI
---@field private winid? integer
---@field private mappings key_mapping[]
---@field private notes table<namespace_id, table<note_id, note_details>> namespace: { id: note_details } mapping
---@field private current_note_id? note_id
---@field private directory string directory where notes are stored
---@field private event_callbacks table<editor_event_name, event_listener[]> callbacks for events
---@field private window_options table<string, any> a table of window options.
---@field private buffer_options table<string, any> a table of buffer options for all notes.
local EditorUI = {}

---@param handler Handler
---@param result ResultUI
---@param opts? editor_config
---@return EditorUI
function EditorUI:new(handler, result, opts)
  opts = opts or {}

  if not handler then
    error("no Handler provided to EditorTile")
  end
  if not result then
    error("no Result provided to EditorTile")
  end

  -- class object
  ---@type EditorUI
  local o = {
    handler = handler,
    result = result,
    notes = {},
    event_callbacks = {},
    directory = opts.directory or vim.fn.stdpath("state") .. "/dbee/notes",
    mappings = opts.mappings,
    window_options = vim.tbl_extend("force", {}, opts.window_options or {}),
    buffer_options = vim.tbl_extend("force", {
      buflisted = false,
      swapfile = false,
      filetype = "sql",
    }, opts.buffer_options or {}),
  }
  setmetatable(o, self)
  self.__index = self

  -- set the current note as first note from global namespace
  local global_notes = o:namespace_get_notes("global")
  if not vim.tbl_isempty(global_notes) then
    o.current_note_id = global_notes[1].id
  else
    -- otherwise create a welcome note in global namespace
    o.current_note_id = o:create_welcome_note()
  end

  return o
end

---@private
---@return note_id
function EditorUI:create_welcome_note()
  local note_id = self:namespace_create_note("global", "welcome")
  local note = self:search_note(note_id)
  if not note then
    error("failed creating welcome note")
  end

  -- create note buffer with contents
  local bufnr = vim.api.nvim_create_buf(false, false)
  vim.api.nvim_buf_set_name(bufnr, note.file)
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, true, welcome.banner())
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

  -- configure options and mappings on new buffer
  common.configure_buffer_options(bufnr, self.buffer_options)
  common.configure_buffer_mappings(bufnr, self:get_actions(), self.mappings)

  return note_id
end

---@private
---@return table<string, fun()>
function EditorUI:get_actions()
  return {
    run_file = function()
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
    run_selection = function()
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
    run_under_cursor = function()
      local bufnr = vim.api.nvim_get_current_buf()
      local query, srow, erow = utils.query_under_cursor(bufnr)

      if query ~= "" then
        -- highlight the statement that will be executed
        local ns_id = vim.api.nvim_create_namespace("dbee_query_highlight")
        vim.api.nvim_buf_clear_namespace(bufnr, ns_id, 0, -1)
        vim.api.nvim_buf_set_extmark(bufnr, ns_id, srow, 0, {
          end_row = erow + 1,
          end_col = 0,
          hl_group = "DiffText",
          priority = 100,
        })

        -- run the query
        local conn = self.handler:get_current_connection()
        if conn then
          self.result:set_call(self.handler:connection_execute(conn.id, query))
        end

        -- remove highlighting after delay
        vim.defer_fn(function()
          vim.api.nvim_buf_clear_namespace(bufnr, ns_id, 0, -1)
        end, 750)
      end
    end,
  }
end

---Triggers an in-built action.
---@param action string
function EditorUI:do_action(action)
  local act = self:get_actions()[action]
  if not act then
    error("unknown action: " .. action)
  end
  act()
end

---@private
---@param event editor_event_name
---@param data any
function EditorUI:trigger_event(event, data)
  local cbs = self.event_callbacks[event] or {}
  for _, cb in ipairs(cbs) do
    cb(data)
  end
end

---@param event editor_event_name
---@param listener event_listener
function EditorUI:register_event_listener(event, listener)
  self.event_callbacks[event] = self.event_callbacks[event] or {}
  table.insert(self.event_callbacks[event], listener)
end

---@private
---@param namespace string
---@return string
function EditorUI:dir(namespace)
  return self.directory .. "/" .. namespace
end

---@private
---@param id namespace_id
---@param name string name to check
---@return boolean # true - conflict, false - no conflict
function EditorUI:namespace_check_conflict(id, name)
  local notes = self.notes[id] or {}
  for _, note in pairs(notes) do
    if note.name == name then
      return true
    end
  end
  return false
end

---@param id note_id
---@return note_details?
---@return namespace_id namespace
function EditorUI:search_note(id)
  for namespace, per_namespace in pairs(self.notes) do
    for _, note in pairs(per_namespace) do
      if note.id == id then
        return note, namespace
      end
    end
  end
  return nil, ""
end

---@param bufnr integer
---@return note_details?
---@return namespace_id namespace
function EditorUI:search_note_with_buf(bufnr)
  for namespace, per_namespace in pairs(self.notes) do
    for _, note in pairs(per_namespace) do
      if note.bufnr and note.bufnr == bufnr then
        return note, namespace
      end
    end
  end
  return nil, ""
end

---@param file string
---@return note_details?
---@return namespace_id namespace
function EditorUI:search_note_with_file(file)
  for namespace, per_namespace in pairs(self.notes) do
    for _, note in pairs(per_namespace) do
      if note.file and note.file == file then
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
function EditorUI:namespace_create_note(id, name)
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
  local note_id = file .. utils.random_string()
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
function EditorUI:namespace_get_notes(id)
  local namespace = id
  if not namespace or namespace == "" then
    error("invalid namespace id")
  end

  if not self.notes[namespace] then
    self.notes[namespace] = self:load_notes_from_disk(namespace)
  end
  local notes_list = vim.tbl_values(self.notes[namespace])

  table.sort(notes_list, function(k1, k2)
    return k1.name < k2.name
  end)
  return notes_list
end

-- If no notes were found, return an empty table.
---@private
---@param namespace_id namespace_id
---@return table<note_id, note_details>
function EditorUI:load_notes_from_disk(namespace_id)
  local full_dir = self.directory .. "/" .. namespace_id
  local ret = {}
  for _, file in pairs(vim.split(vim.fn.glob(full_dir .. "/*"), "\n")) do
    if vim.fn.filereadable(file) == 1 then
      local id = file .. utils.random_string()
      ret[id] = {
        id = id,
        name = vim.fs.basename(file),
        file = file,
      }
    end
  end
  return ret
end

-- Removes an existing note.
-- Errors if there is no note with provided id in namespace.
---@param id namespace_id
---@param note_id note_id
function EditorUI:namespace_remove_note(id, note_id)
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
function EditorUI:note_rename(id, name)
  local note, namespace = self:search_note(id)
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
function EditorUI:get_current_note()
  local note, _ = self:search_note(self.current_note_id)
  return note
end

-- Sets note with id as the current note
-- and opens it in the window
---@param id note_id
function EditorUI:set_current_note(id)
  if id and self.current_note_id == id then
    self:display_note(id)
    return
  end

  local note, _ = self:search_note(id)
  if not note then
    error("invalid note set as current")
  end

  self.current_note_id = id

  self:display_note(id)

  self:trigger_event("current_note_changed", { note_id = id })
end

---@private
---@param id note_id
function EditorUI:display_note(id)
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    return
  end

  local note, namespace = self:search_note(id)
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

  -- configure options and mappings on new buffer
  common.configure_buffer_options(bufnr, self.buffer_options)
  common.configure_buffer_mappings(bufnr, self:get_actions(), self.mappings)
end

---@param winid integer
function EditorUI:show(winid)
  self.winid = winid

  -- open current note
  self:display_note(self.current_note_id)

  -- configure window options (needs to be set after setting the buffer to window)
  common.configure_window_options(winid, self.window_options)
end

return EditorUI
