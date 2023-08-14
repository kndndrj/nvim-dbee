local utils = require("dbee.utils")
local Conn = require("dbee.handler.conn")
local MockedConn = require("dbee.handler.conn_mock")
local Helpers = require("dbee.handler.helpers")
local Lookup = require("dbee.handler.lookup")

---@alias handler_config { fallback_page_size: integer }

-- Handler is an aggregator of connections
---@class Handler
---@field private ui Ui ui for results
---@field private lookup Lookup lookup for sources and connections
---@field private helpers Helpers query helpers
---@field private opts handler_config
local Handler = {}

---@param ui Ui ui for displaying results
---@param sources? Source[]
---@param opts? handler_config
---@return Handler
function Handler:new(ui, sources, opts)
  if not ui then
    error("no results Ui passed to Handler")
  end

  -- class object
  local o = {
    ui = ui,
    lookup = Lookup:new(),
    helpers = Helpers:new(),
    opts = opts or {},
  }
  setmetatable(o, self)
  self.__index = self

  -- initialize the sources
  sources = sources or {}
  for _, source in ipairs(sources) do
    pcall(o.source_add, o, source)
  end

  return o
end

-- add new source and load connections from it
---@param source Source
function Handler:source_add(source)
  local id = source:name()
  -- add it
  self.lookup:add_source(source)
  -- and load it's connections
  self:source_reload(id)
end

---@param id source_id
function Handler:source_reload(id)
  local source = self.lookup:get_source(id)
  if not source then
    return
  end

  -- remove old connections
  local old_conns = self.lookup:get_connections(id)
  for _, conn in ipairs(old_conns) do
    self.lookup:remove_connection(conn:details().id)
  end

  -- add new connections
  for _, spec in ipairs(source:load()) do
    -- create a new connection
    ---@type Conn
    local conn, ok
    ok, conn = pcall(Conn.new, Conn, self.ui, self.helpers, spec, {
      fallback_page_size = self.opts.fallback_page_size,
      on_exec = function()
        self:set_active(conn:details().id)
      end,
    })
    if ok then
      -- add it to lookup
      self.lookup:add_connection(conn, id)
    else
      utils.log("error", tostring(conn), "handler")
    end
  end
end

--- adds connection
---@param params connection_details
---@param source_id source_id id of the source to save connection to
---@return conn_id # id of the added connection
function Handler:add_connection(params, source_id)
  if not source_id then
    error("no source id provided")
  end
  -- create a new connection
  ---@type Conn
  local conn, ok
  ok, conn = pcall(Conn.new, Conn, self.ui, self.helpers, params, {
    fallback_page_size = self.opts.fallback_page_size,
    on_exec = function()
      self:set_active(conn:details().id)
    end,
  })
  if not ok then
    utils.log("error", tostring(conn), "handler")
    return ""
  end

  -- remove it if the same one exists
  self:remove_connection(conn:details().id)

  -- add it to lookup
  self.lookup:add_connection(conn, source_id)

  -- save it to source if it exists
  local source = self.lookup:get_source(source_id)
  if source and type(source.save) == "function" then
    source:save({ conn:original_details() }, "add")
  end

  return conn:details().id
end

-- removes/unregisters connection
-- also deletes it from the source if it exists
---@param id conn_id connection id
function Handler:remove_connection(id)
  local conn = self.lookup:get_connection(id)
  if not conn then
    return
  end

  local original_details = conn:original_details()

  -- delete it from the source
  local source = self.lookup:get_sources(id)[1]
  if source and type(source.save) == "function" then
    source:save({ original_details }, "delete")
  end

  -- delete it
  self.lookup:remove_connection(id)
end

---@param id conn_id connection id
function Handler:set_active(id)
  self.lookup:set_active_connection(id)
end

---@return Conn # currently active connection
function Handler:current_connection()
  return self.lookup:get_active_connection() or MockedConn:new()
end

---@param helpers table<string, table_helpers> extra helpers per type
function Handler:add_helpers(helpers)
  self.helpers:add(helpers)
end

---@param type string
---@param vars { table: string, schema: string, dbname: string }
---@return table_helpers helpers list of table helpers
function Handler:get_helpers(type, vars)
  return self.helpers:get(type, vars)
end

---@return Layout[]
function Handler:layout()
  -- in case there are no sources defined, return a helper layout
  if #self.lookup:get_sources() < 1 then
    print("here")
    vim.print(self:layout_help())
    return self:layout_help()
  end
  return self:layout_real()
end

---@private
---@return Layout[]
function Handler:layout_help()
  return {
    {
      id = "__handler_help_id__",
      name = "No sources :(",
      default_expand = utils.once:new("handler_expand_once_helper_id"),
      type = "",
      children = {
        {
          id = "__handler_help_id_child_1__",
          name = 'Type ":h dbee.txt"',
          type = "",
        },
        {
          id = "__handler_help_id_child_2__",
          name = "to define your first source!",
          type = "",
        },
      },
    },
  }
end

---@private
---@return Layout[]
function Handler:layout_real()
  ---@type Layout[]
  local layout = {}

  local all_sources = self.lookup:get_sources()

  for _, source in ipairs(all_sources) do
    local source_id = source:name()

    local children = {}

    -- source can save edits
    if type(source.save) == "function" then
      table.insert(children, {
        id = "__source_add_connection__" .. source_id,
        name = "add",
        type = "add",
        action_1 = function(cb)
          local prompt = {
            { name = "name" },
            { name = "type" },
            { name = "url" },
            { name = "page size" },
          }
          utils.prompt.open(prompt, {
            title = "Add Connection",
            callback = function(result)
              local spec = {
                id = result.id,
                name = result.name,
                url = result.url,
                type = result.type,
                page_size = tonumber(result["page size"]),
              }
              pcall(self.add_connection, self, spec --[[@as connection_details]], source_id)
              cb()
            end,
          })
        end,
      })
    end
    -- source has an editable source
    if type(source.file) == "function" then
      table.insert(children, {
        id = "__source_edit_connections__" .. source_id,
        name = "edit source",
        type = "edit",
        action_1 = function(cb)
          utils.prompt.edit(source:file(), {
            title = "Add Connection",
            callback = function()
              self:source_reload(source_id)
              cb()
            end,
          })
        end,
      })
    end

    for _, conn in ipairs(self.lookup:get_connections(source_id)) do
      local details = conn:details()
      table.insert(children, {
        id = details.id,
        name = details.name,
        type = "database",
        -- set connection as active manually
        action_1 = function(cb)
          self:set_active(details.id)
          cb()
        end,
        action_2 = function(cb)
          local original_details = conn:original_details()
          local prompt = {
            { name = "name", default = original_details.name },
            { name = "type", default = original_details.type },
            { name = "url", default = original_details.url },
            { name = "page size", default = tostring(original_details.page_size or "") },
          }
          utils.prompt.open(prompt, {
            title = "Edit Connection",
            callback = function(result)
              local spec = {
                -- keep the old id
                id = original_details.id,
                name = result.name,
                url = result.url,
                type = result.type,
                page_size = tonumber(result["page size"]),
              }
              -- parse page size to int
              pcall(self.add_connection, self, spec --[[@as connection_details]], source_id)
              cb()
            end,
          })
        end,
        -- remove connection (also trigger the source's function)
        action_3 = function(cb)
          vim.ui.input({ prompt = 'confirm deletion of "' .. details.name .. '"', default = "Y" }, function(input)
            if not input or string.lower(input) ~= "y" then
              return
            end
            self:remove_connection(details.id)
            cb()
          end)
        end,
        children = function()
          return conn:layout()
        end,
      })
    end

    if #children > 0 then
      table.insert(layout, {
        id = "__source__" .. source_id,
        name = source_id,
        default_expand = utils.once:new("handler_expand_once_id" .. source_id),
        type = "source",
        children = children,
      })
    end
  end

  return layout
end

return Handler
