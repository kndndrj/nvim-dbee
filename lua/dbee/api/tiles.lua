---@mod dbee.api.tiles Dbee Tiles API
---@brief [[
---Tiles API module for nvim dbee.
---
---This module contains functions to operate with tiles (UI).
---Functions are prefixed with a tile name:
---- editor
---- result
---- drawer
---- call_log
---
--- Access the module like this:
--->
---require("dbee").api.tiles.func()
---<
---@brief ]]

local entry = require("dbee.entry")

local tiles = {}

---@divider -
---@tag dbee.api.tiles.editor
---@brief [[
---Editor API
---@brief ]]

---Registers an event handler for editor events.
---@param event editor_event_name
---@param listener event_listener
function tiles.editor_register_event_listener(event, listener)
  entry.get_tiles().editor:register_event_listener(event, listener)
end

--- Search for a note across namespaces.
---@param id note_id
---@return note_details|nil
---@return namespace_id _ namespace of the note
function tiles.editor_search_note(id)
  return entry.get_tiles().editor:search_note(id)
end

--- Creates a new note in namespace.
--- Errors if id or name is nil or there is a note with the same
--- name in namespace already.
---@param id namespace_id
---@param name string
---@return note_id
function tiles.editor_namespace_create_note(id, name)
  return entry.get_tiles().editor:namespace_create_note(id, name)
end

--- Get notes of a specified namespace.
---@param id namespace_id
---@return note_details[]
function tiles.editor_namespace_get_notes(id)
  return entry.get_tiles().editor:namespace_get_notes(id)
end

--- Removes an existing note.
--- Errors if there is no note with provided id in namespace.
---@param id namespace_id
---@param note_id note_id
function tiles.editor_namespace_remove_note(id, note_id)
  entry.get_tiles().editor:namespace_remove_note(id, note_id)
end

--- Renames an existing note.
--- Errors if no name or id provided, there is no note with provided id or
--- there is already an existing note with the same name in the same namespace.
---@param id note_id
---@param name string new name
function tiles.editor_note_rename(id, name)
  entry.get_tiles().editor:note_rename(id, name)
end

--- Get details of a current note
---@return note_details|nil
function tiles.editor_get_current_note()
  return entry.get_tiles().editor:get_current_note()
end

--- Sets note with id as the current note
--- and opens it in the window.
---@param id note_id
function tiles.editor_set_current_note(id)
  entry.get_tiles().editor:set_current_note(id)
end

--- Open the editor UI.
---@param winid integer
function tiles.editor_show(winid)
  entry.get_tiles().editor:show(winid)
end

---@divider -
---@tag dbee.api.tiles.call_log
---@brief [[
---Call Log API
---@brief ]]

--- Refresh the call log.
function tiles.call_log_refresh()
  entry.get_tiles().call_log:refresh()
end

--- Open the call log UI.
---@param winid integer
function tiles.call_log_show(winid)
  entry.get_tiles().call_log:show(winid)
end

---@divider -
---@tag dbee.api.tiles.drawer
---@brief [[
---Drawer API
---@brief ]]

--- Refresh the drawer.
function tiles.drawer_refresh()
  entry.get_tiles().drawer:refresh()
end

--- Open the drawer UI.
---@param winid integer
function tiles.drawer_show(winid)
  entry.get_tiles().drawer:show(winid)
end

---@divider -
---@tag dbee.api.tiles.result
---@brief [[
---Result API
---@brief ]]

--- Sets call's result to Result's buffer.
---@param call CallDetails
function tiles.result_set_call(call)
  entry.get_tiles().result:set_call(call)
end

--- Gets the currently displayed call.
---@return CallDetails|nil
function tiles.result_get_call()
  return entry.get_tiles().result:get_call()
end

--- Display the currently selected page in results UI.
function tiles.result_page_current()
  entry.get_tiles().result:page_current()
end

--- Go to next page in results UI and display it.
function tiles.result_page_next()
  entry.get_tiles().result:page_next()
end

--- Go to previous page in results UI and display it.
function tiles.result_page_prev()
  entry.get_tiles().result:page_prev()
end

--- Open the result UI.
---@param winid integer
function tiles.result_show(winid)
  entry.get_tiles().result:show(winid)
end

return tiles
