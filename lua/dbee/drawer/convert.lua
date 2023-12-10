local floats = require("dbee.floats")
local utils = require("dbee.utils")
local NuiTree = require("nui.tree")

local M = {}

---@param handler Handler
---@param conn connection_details
---@param result Result
---@return DrawerNode[]
local function connection_nodes(handler, conn, result)
  ---@param structs DBStructure[]
  ---@param parent_id string
  ---@return DrawerNode[]
  local function to_tree_nodes(structs, parent_id)
    if not structs or structs == vim.NIL then
      return {}
    end

    table.sort(structs, function(k1, k2)
      return k1.type .. k1.name < k2.type .. k2.name
    end)

    ---@type DrawerNode[]
    local nodes = {}

    for _, struct in ipairs(structs) do
      local node_id = (parent_id or "") .. "__connection_" .. struct.name .. struct.schema .. struct.type .. "__"
      local node = NuiTree.Node({
        id = node_id,
        name = struct.name,
        schema = struct.schema,
        type = struct.type,
      }, to_tree_nodes(struct.children, node_id)) --[[@as DrawerNode]]

      -- table helpers
      if struct.type == "table" or struct.type == "view" then
        local helper_opts = { table = struct.name, schema = struct.schema, materialization = struct.type }
        node.action_1 = function(cb, pick)
          local items = vim.tbl_keys(handler:helpers_get(conn.type, helper_opts))
          table.sort(items)

          pick {
            title = "Select a Query",
            items = items,
            on_select = function(selection)
              local helpers = handler:helpers_get(conn.type, helper_opts)
              local call = handler:connection_execute(conn.id, helpers[selection])
              result:set_call(call)
              cb()
            end,
          }
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
      action_1 = function(cb, pick)
        pick {
          title = "Select a Database",
          items = available_dbs,
          on_select = function(selection)
            handler:connection_select_database(conn.id, selection)
            cb()
          end,
        }
      end,
    } --[[@as DrawerNode]]
    table.insert(nodes, 1, ly)
  end

  return nodes
end

---@param handler Handler
---@param result Result
---@return DrawerNode[]
local function handler_real_nodes(handler, result)
  ---@type DrawerNode[]
  local nodes = {}

  for _, source in ipairs(handler:get_sources()) do
    local source_id = source:name()

    ---@type DrawerNode[]
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
            floats.prompt(prompt, {
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
        } --[[@as DrawerNode]]
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
            floats.editor(source:file(), {
              title = "Add Connection",
              callback = function()
                handler:source_reload(source_id)
                cb()
              end,
            })
          end,
        } --[[@as DrawerNode]]
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
          floats.prompt(prompt, {
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
        action_3 = function(cb, pick)
          pick {
            title = "Confirm Deletion",
            items = { "Yes", "No" },
            on_select = function(selection)
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
      } --[[@as DrawerNode]]

      table.insert(children, node)
    end

    if #children > 0 then
      local node = NuiTree.Node({
        id = "__source__" .. source_id,
        name = source_id,
        type = "source",
      }, children) --[[@as DrawerNode]]

      if utils.once("handler_expand_once_id" .. source_id) then
        node:expand()
      end

      table.insert(nodes, node)
    end
  end

  return nodes
end

---@return DrawerNode[]
local function handler_help_nodes()
  local node = NuiTree.Node({
    {
      id = "__handler_help_id__",
      name = "No sources :(",
      type = "",
    },
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

  return node
end

---@param handler Handler
---@param result Result
---@return DrawerNode[]
function M.handler_nodes(handler, result)
  -- in case there are no sources defined, return helper nodes
  if #handler:get_sources() < 1 then
    return handler_help_nodes()
  end
  return handler_real_nodes(handler, result)
end

-- whitespace between nodes
---@return DrawerNode
function M.separator_node()
  return NuiTree.Node {
    id = "__separator_node__" .. tostring(math.random()),
    name = "",
    type = "",
  } --[[@as DrawerNode]]
end

---@param mappings table<string, mapping>
---@return DrawerNode
function M.help_node(mappings)
  -- help node
  ---@type DrawerNode[]
  local children = {}
  for act, map in pairs(mappings) do
    table.insert(
      children,
      NuiTree.Node {
        id = "__help_action_" .. act,
        name = act .. " = " .. map.key .. " (" .. map.mode .. ")",
        type = "",
      }
    )
  end

  table.sort(children, function(k1, k2)
    return k1.id < k2.id
  end)

  local node = NuiTree.Node({
    id = "__help_node__",
    name = "help",
    type = "help",
  }, children) --[[@as DrawerNode]]

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
    suffix = " - o"
  end

  utils.create_singleton_autocmd({ "BufModifiedSet" }, {
    buffer = bufnr,
    callback = refresh,
  })

  return suffix
end

---@param editor Editor
---@param namespace namespace_id
---@param refresh fun() function that refreshes the tree
---@return DrawerNode[]
local function editor_namespace_nodes(editor, namespace, refresh)
  ---@type DrawerNode[]
  local nodes = {}

  table.insert(
    nodes,
    NuiTree.Node {
      id = "__new_" .. namespace .. "_note__",
      name = "new",
      type = "add",
      action_1 = function(cb)
        -- TODO: name
        local id = editor:namespace_create_note(namespace, "note_" .. tostring(os.clock()))
        editor:set_current_note(id)
        cb()
      end,
    } --[[@as DrawerNode]]
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
      action_2 = function(cb)
        vim.ui.input({ prompt = "new name: ", default = note.name }, function(input)
          if not input or input == "" then
            return
          end
          editor:note_rename(note.id, input)
          cb()
        end)
      end,
      action_3 = function(cb, pick)
        pick {
          title = "Confirm Deletion",
          items = { "Yes", "No" },
          on_select = function(selection)
            if selection == "Yes" then
              editor:namespace_remove_note(namespace, note.id)
            end
            cb()
          end,
        }
      end,
    } --[[@as DrawerNode]]

    table.insert(nodes, node)
  end

  return nodes
end

---@param editor Editor
---@param current_connection_id conn_id
---@param refresh fun() function that refreshes the tree
---@return DrawerNode[]
function M.editor_nodes(editor, current_connection_id, refresh)
  local nodes = {
    NuiTree.Node({
      id = "__master_note_global__",
      name = "global notes",
      type = "note",
    }, editor_namespace_nodes(editor, "global", refresh)),
    NuiTree.Node({
      id = "__master_note_local__",
      name = "local notes",
      type = "note",
    }, editor_namespace_nodes(editor, current_connection_id, refresh)),
  }

  if utils.once("editor_global_expand") then
    nodes[1]:expand()
  end
  if utils.once("editor_local_expand") then
    nodes[2]:expand()
  end

  return nodes
end

return M
