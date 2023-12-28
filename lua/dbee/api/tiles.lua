local entry = require("dbee.entry")

local M = {}

--
-- Editor
--

---@param event editor_event_name
---@param listener editor_event_listener
function M.editor_register_event_listener(event, listener)
  entry.get_tiles().editor:register_event_listener(event, listener)
end

-- Search for a note across namespaces.
---@param id note_id
---@return note_details?
---@return namespace_id namespace
function M.editor_search_note(id)
  return entry.get_tiles().editor:search_note(id)
end

-- Creates a new note in namespace.
-- Errors if id or name is nil or there is a note with the same
-- name in namespace already.
---@param id namespace_id
---@param name string
---@return note_id
function M.editor_namespace_create_note(id, name)
  return entry.get_tiles().editor:namespace_create_note(id, name)
end

---@param id namespace_id
---@return note_details[]
function M.editor_namespace_get_notes(id)
  return entry.get_tiles().editor:namespace_get_notes(id)
end

-- Removes an existing note.
-- Errors if there is no note with provided id in namespace.
---@param id namespace_id
---@param note_id note_id
function M.editor_namespace_remove_note(id, note_id)
  entry.get_tiles().editor:namespace_remove_note(id, note_id)
end

-- Renames an existing note.
-- Errors if no name or id provided, there is no note with provided id or
-- there is already an existing note with the same name in the same namespace.
---@param id note_id
---@param name string new name
function M.editor_note_rename(id, name)
  entry.get_tiles().editor:note_rename(id, name)
end

---@return note_details?
function M.editor_get_current_note()
  return entry.get_tiles().editor:get_current_note()
end

-- Sets note with id as the current note
-- and opens it in the window
---@param id note_id
function M.editor_set_current_note(id)
  entry.get_tiles().editor:set_current_note(id)
end

---@param winid integer
function M.editor_show(winid)
  entry.get_tiles().editor:show(winid)
end

--
-- Call Log
--

-- Refresh the call log.
function M.call_log_refresh()
  entry.get_tiles().call_log:refresh()
end

---@param winid integer
function M.call_log_show(winid)
  entry.get_tiles().call_log:show(winid)
end

--
-- Drawer
--

-- Refresh the drawer.
function M.drawer_refresh()
  entry.get_tiles().drawer:refresh()
end

---@param winid integer
function M.drawer_show(winid)
  entry.get_tiles().drawer:show(winid)
end

--
-- Result
--

-- Sets call's result to Result's buffer.
---@param call call_details
function M.result_set_call(call)
  entry.get_tiles().result:set_call(call)
end

function M.result_page_current()
  entry.get_tiles().result:page_current()
end

function M.result_page_next()
  entry.get_tiles().result:page_next()
end

function M.result_page_prev()
  entry.get_tiles().result:page_prev()
end

---@param winid integer
function M.result_show(winid)
  entry.get_tiles().result:show(winid)
end

return M
