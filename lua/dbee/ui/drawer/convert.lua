local utils = require("dbee.utils")
local common = require("dbee.ui.common")
local NuiTree = require("nui.tree")

local M = {}

---@param parent_id string
---@param columns Column[]
---@return DrawerUINode[]
local function column_nodes(parent_id, columns)
  ---@type DrawerUINode[]
  local nodes = {}

  for _, column in ipairs(columns) do
    table.insert(
      nodes,
      NuiTree.Node {
        id = parent_id .. column.type .. column.name,
        name = column.name .. "   [" .. column.type .. "]",
        type = "column",
      }
    )
  end

  return nodes
end

---@param handler Handler
---@param conn ConnectionParams
---@param result ResultUI
---@return DrawerUINode[]
local function connection_nodes(handler, conn, result)
  ---@param structs DBStructure[]
  ---@param parent_id string
  ---@return DrawerUINode[]
  local function to_tree_nodes(structs, parent_id)
    if not structs or structs == vim.NIL then
      return {}
    end

    table.sort(structs, function(k1, k2)
      return k1.type .. k1.name < k2.type .. k2.name
    end)

    ---@type DrawerUINode[]
    local nodes = {}

    for _, struct in ipairs(structs) do
      local node_id = (parent_id or "") .. "__connection_" .. struct.name .. struct.schema .. struct.type .. "__"
      local node = NuiTree.Node({
        id = node_id,
        name = struct.name,
        schema = struct.schema,
        type = struct.type,
      }, to_tree_nodes(struct.children, node_id)) --[[@as DrawerUINode]]

      if struct.type == "table" or struct.type == "view" then
        local table_opts = { table = struct.name, schema = struct.schema, materialization = struct.type }

        -- table helpers
        node.action_1 = function(cb, select)
          local helpers = handler:connection_get_helpers(conn.id, table_opts)
          local items = vim.tbl_keys(helpers)
          table.sort(items)

          select {
            title = "Select a Query",
            items = items,
            on_confirm = function(selection)
              local call = handler:connection_execute(conn.id, helpers[selection])
              result:set_call(call)
              cb()
            end,
            on_yank = function(selection)
              vim.fn.setreg(vim.v.register, helpers[selection])
            end,
          }
        end

        node.lazy_children = function()
          return column_nodes(node_id, handler:connection_get_columns(conn.id, table_opts))
        end
      end

      table.insert(nodes, node)
    end

    return nodes
  end

  -- recursively parse structure to drawer nodes
  local nodes = to_tree_nodes(handler:connection_get_structure(conn.id), conn.id)

  -- database switching
  local current_db, available_dbs = handler:connection_list_databases(conn.id)
  if current_db ~= "" and #available_dbs > 0 then
    local ly = NuiTree.Node {
      id = conn.id .. "_database_switch__",
      name = current_db,
      type = "database_switch",
      action_1 = function(cb, select)
        select {
          title = "Select a Database",
          items = available_dbs,
          on_confirm = function(selection)
            handler:connection_select_database(conn.id, selection)
            cb()
          end,
        }
      end,
    } --[[@as DrawerUINode]]
    table.insert(nodes, 1, ly)
  end

  return nodes
end

---@param handler Handler
---@param result ResultUI
---@return DrawerUINode[]
local function handler_real_nodes(handler, result)
  ---@type DrawerUINode[]
  local nodes = {}

  for _, source in ipairs(handler:get_sources()) do
    local source_id = source:name()

    ---@type DrawerUINode[]
    local children = {}

    -- source can save edits
    if type(source.save) == "function" then
      table.insert(
        children,
        NuiTree.Node {
          id = "__source_add_connection__" .. source_id,
          name = "add",
          type = "add",
          action_1 = function(cb)
            local prompt = {
              { name = "name" },
              { name = "type" },
              { name = "url" },
            }
            common.float_prompt(prompt, {
              title = "Add Connection",
              callback = function(res)
                local spec = {
                  id = res.id,
                  name = res.name,
                  url = res.url,
                  type = res.type,
                }
                pcall(handler.source_add_connections, handler, source_id, { spec })
                cb()
              end,
            })
          end,
        } --[[@as DrawerUINode]]
      )
    end

    -- source has an editable source file
    if type(source.file) == "function" then
      table.insert(
        children,
        NuiTree.Node {
          id = "__source_edit_connections__" .. source_id,
          name = "edit source",
          type = "edit",
          action_1 = function(cb)
            common.float_editor(source:file(), {
              title = "Add Connection",
              callback = function()
                handler:source_reload(source_id)
                cb()
              end,
            })
          end,
        } --[[@as DrawerUINode]]
      )
    end

    -- get connections of that source
    for _, conn in ipairs(handler:source_get_connections(source_id)) do
      local node = NuiTree.Node {
        id = conn.id,
        name = conn.name,
        type = "connection",
        -- set connection as active manually
        action_1 = function(cb)
          handler:set_current_connection(conn.id)
          cb()
        end,
        -- edit connection
        action_2 = function(cb)
          local original_details = handler:connection_get_params(conn.id)
          if not original_details then
            return
          end
          local prompt = {
            { name = "name", default = original_details.name },
            { name = "type", default = original_details.type },
            { name = "url", default = original_details.url },
          }
          common.float_prompt(prompt, {
            title = "Edit Connection",
            callback = function(res)
              local spec = {
                -- keep the old id
                id = original_details.id,
                name = res.name,
                url = res.url,
                type = res.type,
                page_size = tonumber(res["page size"]),
              }
              pcall(handler.source_add_connections, handler, source_id, { spec })
              cb()
            end,
          })
        end,
        -- remove connection
        action_3 = function(cb, select)
          select {
            title = "Confirm Deletion",
            items = { "Yes", "No" },
            on_confirm = function(selection)
              if selection == "Yes" then
                handler:source_remove_connections(source_id, conn)
              end
              cb()
            end,
          }
        end,
        lazy_children = function()
          return connection_nodes(handler, conn, result)
        end,
      } --[[@as DrawerUINode]]

      table.insert(children, node)
    end

    if #children > 0 then
      local node = NuiTree.Node({
        id = "__source__" .. source_id,
        name = source_id,
        type = "source",
      }, children) --[[@as DrawerUINode]]

      if utils.once("handler_expand_once_id" .. source_id) then
        node:expand()
      end

      table.insert(nodes, node)
    end
  end

  return nodes
end

---@return DrawerUINode[]
local function handler_help_nodes()
  local node = NuiTree.Node({
    id = "__handler_help_id__",
    name = "No sources :(",
    type = "",
  }, {
    NuiTree.Node {
      id = "__handler_help_id_child_1__",
      name = 'Type ":h dbee.txt"',
      type = "",
    },
    NuiTree.Node {
      id = "__handler_help_id_child_2__",
      name = "to define your first source!",
      type = "",
    },
  })

  if utils.once("handler_expand_once_helper_id") then
    node:expand()
  end

  return { node }
end

---@param handler Handler
---@param result ResultUI
---@return DrawerUINode[]
function M.handler_nodes(handler, result)
  -- in case there are no sources defined, return helper nodes
  if #handler:get_sources() < 1 then
    return handler_help_nodes()
  end
  return handler_real_nodes(handler, result)
end

-- whitespace between nodes
---@return DrawerUINode
function M.separator_node()
  return NuiTree.Node {
    id = "__separator_node__" .. tostring(math.random()),
    name = "",
    type = "",
  } --[[@as DrawerUINode]]
end

---@param mappings key_mapping[]
---@return DrawerUINode
function M.help_node(mappings)
  -- help node
  ---@type DrawerUINode[]
  local children = {}
  for _, km in ipairs(mappings) do
    if type(km.action) == "string" then
      table.insert(
        children,
        NuiTree.Node {
          id = "__help_action_" .. utils.generate_guid(7),
          name = km.action .. " = " .. km.key .. " (" .. km.mode .. ")",
          type = "",
        }
      )
    end
  end

  table.sort(children, function(k1, k2)
    return k1.id < k2.id
  end)

  local node = NuiTree.Node({
    id = "__help_node__",
    name = "help",
    type = "help",
  }, children) --[[@as DrawerUINode]]

  if utils.once("help_expand_once_id") then
    node:expand()
  end

  return node
end

---@param bufnr integer
---@param refresh fun() function that refreshes the tree
---@return string suffix
local function modified_suffix(bufnr, refresh)
  if not bufnr or not vim.api.nvim_buf_is_valid(bufnr) then
    return ""
  end

  local suffix = ""
  if vim.api.nvim_buf_get_option(bufnr, "modified") then
    suffix = " ●"
  end

  utils.create_singleton_autocmd({ "BufModifiedSet" }, {
    buffer = bufnr,
    callback = refresh,
  })

  return suffix
end

---@param editor EditorUI
---@param namespace namespace_id
---@param refresh fun() function that refreshes the tree
---@return DrawerUINode[]
local function editor_namespace_nodes(editor, namespace, refresh)
  ---@type DrawerUINode[]
  local nodes = {}

  table.insert(
    nodes,
    NuiTree.Node {
      id = "__new_" .. namespace .. "_note__",
      name = "new",
      type = "add",
      action_1 = function(cb, _, input)
        input {
          title = "Enter Note Name",
          default = "note_" .. tostring(os.clock()) .. ".sql",
          on_confirm = function(value)
            if not value or value == "" then
              return
            end
            local id = editor:namespace_create_note(namespace, value)
            editor:set_current_note(id)
            cb()
          end,
        }
      end,
    } --[[@as DrawerUINode]]
  )

  -- global notes
  for _, note in ipairs(editor:namespace_get_notes(namespace)) do
    local node = NuiTree.Node {
      id = note.id,
      name = note.name .. modified_suffix(note.bufnr, refresh),
      type = "note",
      action_1 = function(cb)
        editor:set_current_note(note.id)
        cb()
      end,
      action_2 = function(cb, _, input)
        input {
          title = "New Name",
          default = note.name,
          on_confirm = function(value)
            if not value or value == "" then
              return
            end
            editor:note_rename(note.id, value)
            cb()
          end,
        }
      end,
      action_3 = function(cb, select)
        select {
          title = "Confirm Deletion",
          items = { "Yes", "No" },
          on_confirm = function(selection)
            if selection == "Yes" then
              editor:namespace_remove_note(namespace, note.id)
            end
            cb()
          end,
        }
      end,
    } --[[@as DrawerUINode]]

    table.insert(nodes, node)
  end

  return nodes
end

---@param editor EditorUI
---@param current_connection_id connection_id
---@param refresh fun() function that refreshes the tree
---@return DrawerUINode[]
function M.editor_nodes(editor, current_connection_id, refresh)
  local nodes = {
    NuiTree.Node({
      id = "__master_note_global__",
      name = "global notes",
      type = "note",
    }, editor_namespace_nodes(editor, "global", refresh)),
  }

  if utils.once("editor_global_expand") then
    nodes[1]:expand()
  end

  if current_connection_id then
    table.insert(
      nodes,
      NuiTree.Node({
        id = "__master_note_local__",
        name = "local notes",
        type = "note",
      }, editor_namespace_nodes(editor, current_connection_id, refresh))
    )
    if utils.once("editor_local_expand") then
      nodes[2]:expand()
    end
  end

  return nodes
end

return M
