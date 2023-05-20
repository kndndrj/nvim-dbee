local utils = require("dbee.utils")
local Conn = require("dbee.handler.conn")
local Lookup = require("dbee.handler.lookup")

---@alias handler_config { expand_help: boolean, default_page_size: integer }

-- Handler is an aggregator of connections
---@class Handler
---@field private ui Ui ui for results
---@field private lookup Lookup lookup for loaders and connections
---@field private default_loader_id string
---@field private opts handler_config
local Handler = {}

---@param ui Ui ui for displaying results
---@param default_loader Loader
---@param other_loaders? Loader[]
---@param opts? handler_config
---@return Handler
function Handler:new(ui, default_loader, other_loaders, opts)
  if not ui then
    error("no results Ui passed to Handler")
  end
  if not default_loader then
    error("no default Loader passed to Handler")
  end

  -- class object
  local o = {
    ui = ui,
    lookup = Lookup:new(),
    default_loader_id = default_loader:name(),
    opts = opts or {},
  }
  setmetatable(o, self)
  self.__index = self

  -- initialize the default loader and others
  o:loader_add(default_loader)

  other_loaders = other_loaders or {}
  for _, loader in ipairs(other_loaders) do
    pcall(o.loader_add, o, loader)
  end

  return o
end

-- add new source and load connections from it
---@param loader Loader
function Handler:loader_add(loader)
  local id = loader:name()
  -- add it
  self.lookup:add_loader(loader)
  -- and load it's connections
  self:loader_reload(id)
end

---@param id loader_id
function Handler:loader_reload(id)
  local loader = self.lookup:get_loader(id)
  if not loader then
    return
  end

  -- remove old connections
  local old_conns = self.lookup:get_connections(id)
  for _, conn in ipairs(old_conns) do
    self.lookup:remove_connection(conn:details().id)
  end

  -- add new connections
  for _, spec in ipairs(loader:load()) do
    -- create a new connection
    spec.page_size = spec.page_size or self.opts.default_page_size
    ---@type Conn
    local conn, ok
    ok, conn = pcall(Conn.new, Conn, self.ui, spec, {
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
---@param loader_id? loader_id id of the loader to save connection to
---@return conn_id # id of the added connection
function Handler:add_connection(params, loader_id)
  loader_id = loader_id or self.default_loader_id
  -- create a new connection
  params.page_size = params.page_size or self.opts.default_page_size
  ---@type Conn
  local conn, ok
  ok, conn = pcall(Conn.new, Conn, self.ui, params, {
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
  self.lookup:add_connection(conn, loader_id)

  -- save it to loader if it exists
  local loader = self.lookup:get_loader(loader_id)
  if loader and type(loader.save) == "function" then
    loader:save({ conn:original_details() }, "add")
  end

  return conn:details().id
end

-- removes/unregisters connection
-- also deletes it from the loader if it exists
---@param id conn_id connection id
function Handler:remove_connection(id)
  local conn = self.lookup:get_connection(id)
  if not conn then
    return
  end

  local original_details = conn:original_details()

  -- delete it from the loader
  local loader = self.lookup:get_loaders(id)[1]
  if loader and type(loader.save) == "function" then
    loader:save({ original_details }, "delete")
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
  return self.lookup:get_active_connection()
end

---@return Layout[]
function Handler:layout()
  ---@type Layout[]
  local layout = {}

  local all_loaders = self.lookup:get_loaders()

  for _, loader in ipairs(all_loaders) do
    local loader_id = loader:name()

    local children = {}

    -- loader can save edits
    if type(loader.save) == "function" or loader_id == self.default_loader_id then
      table.insert(children, {
        id = "__loader_add_connection__" .. loader_id,
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
              pcall(self.add_connection, self, spec --[[@as connection_details]], loader_id)
              cb()
            end,
          })
        end,
      })
    end
    -- loader has an editable source
    if type(loader.source) == "function" then
      table.insert(children, {
        id = "__loader_edit_connections__" .. loader_id,
        name = "edit source",
        type = "edit",
        action_1 = function(cb)
          utils.prompt.edit(loader:source(), {
            title = "Add Connection",
            callback = function()
              self:loader_reload(loader_id)
              cb()
            end,
          })
        end,
      })
    end

    for _, conn in ipairs(self.lookup:get_connections(loader_id)) do
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
              pcall(self.add_connection, self, spec --[[@as connection_details]], loader_id)
              cb()
            end,
          })
        end,
        -- remove connection (also trigger the loader function)
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
        id = "__loader__" .. loader_id,
        name = loader_id,
        do_expand = true,
        type = "loader",
        children = children,
      })
    end
  end

  return layout
end

return Handler
