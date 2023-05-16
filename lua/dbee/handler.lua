local helpers = require("dbee.helpers")
local utils = require("dbee.utils")

---@alias conn_id string
---@alias connection_details { name: string, type: string, url: string, id: conn_id }
--
---@class _LayoutGo
---@field name string display name
---@field type ""|"table"|"history" type of layout -> this infers action
---@field schema? string parent schema
---@field database? string parent database
---@field children? Layout[] child layout nodes
--
---@alias result_config { mappings: table<string, mapping>, page_size: integer, window_command: string|fun():integer }
---@alias loader_config { save: fun(conns: connection_details[]), load: fun():connection_details[] }

-- Handler is a wrapper around the go code
-- it is the central part of the plugin and manages connections.
-- almost all functions take the connection id as their argument.
---@class Handler
---@field private connections table<conn_id, connection_details> id - connection mapping
---@field private active_connection conn_id last called connection
---@field private page_index integer current page
---@field private winid integer
---@field private bufnr integer
---@field private win_cmd fun():integer function which opens a new window and returns a window id
---@field private page_size integer number of rows per page
---@field private mappings table<string, mapping>
---@field private loader_save fun(conns: connection_details[]) function to save connections
---@field private loader_load fun():connection_details[] function to load connections
local Handler = {}

---@param opts? { result: result_config, loader: loader_config }
---@return Handler
function Handler:new(opts)
  opts = opts or {}
  opts.result = opts.result or {}
  opts.loader = opts.loader or {}

  local page_size = opts.result.page_size or 100

  local win_cmd
  if type(opts.result.window_command) == "string" then
    win_cmd = function()
      vim.cmd(opts.result.window_command)
      return vim.api.nvim_get_current_win()
    end
  elseif type(opts.result.window_command) == "function" then
    win_cmd = opts.result.window_command
  else
    win_cmd = function()
      vim.cmd("bo 15split")
      return vim.api.nvim_get_current_win()
    end
  end

  -- class object
  local o = {
    connections = {},
    active_connection = "",
    page_index = 0,
    win_cmd = win_cmd,
    page_size = page_size,
    mappings = opts.result.mappings or {},
    loader_save = opts.loader.save or function() end,
    loader_load = opts.loader.load or function() end,
  }
  setmetatable(o, self)
  self.__index = self

  -- initialize connections from loader
  local conns = o.loader_load()
  for _, conn in ipairs(conns) do
    pcall(o.add_connection, o, conn)
  end

  return o
end

---@param connection connection_details
function Handler:add_connection(connection)
  if not connection.url then
    error("url needs to be set!")
  end
  if not connection.type or connection.type == "" then
    error("no type")
  end

  connection.name = connection.name or "[empty name]"
  connection.type = utils.type_alias(connection.type)
  connection.id = "__master_connection_id_" .. connection.name .. connection.type .. "__"

  -- register in go
  local ok = vim.fn.Dbee_register_connection(connection.id, connection.url, connection.type, tostring(self.page_size))
  if not ok then
    error("problem adding connection")
  end

  self.connections[connection.id] = connection
  self.active_connection = connection.id
end

---@param id conn_id connection id
function Handler:set_active(id)
  if not id or self.connections[id] == nil then
    error("no id specified!")
  end
  self.active_connection = id
end

---@return connection_details[] list of connections
function Handler:list_connections()
  local cons = {}
  for _, con in pairs(self.connections) do
    table.insert(cons, con)
  end

  -- sort keys
  table.sort(cons, function(k1, k2)
    return k1.name < k2.name
  end)
  return cons
end

---@return connection_details
---@param id? conn_id connection id
function Handler:connection_details(id)
  id = id or self.active_connection
  return self.connections[id]
end

---@param query string query to execute
---@param id? conn_id connection id
function Handler:execute(query, id)
  id = id or self.active_connection

  self:open_pre_hook()
  self.page_index = 0
  vim.fn.Dbee_execute(id, query)
  self:open_post_hook()
end

---@private
-- called before anything needs to be displayed on screen
function Handler:open_pre_hook()
  self:open()
  vim.api.nvim_buf_set_option(self.bufnr, "modifiable", true)
  vim.api.nvim_buf_set_lines(self.bufnr, 0, -1, true, { "Loading..." })
  vim.api.nvim_buf_set_option(self.bufnr, "modifiable", false)
end

---@private
-- called after anything needs to be displayed on screen
---@param count? integer total number of pages
function Handler:open_post_hook(count)
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    return
  end

  local total = "?"
  if count then
    total = tostring(count + 1)
  end
  local index = "0"
  if self.page_index then
    index = tostring(self.page_index + 1)
  end

  -- set winbar
  vim.api.nvim_win_set_option(self.winid, "winbar", "%=" .. index .. "/" .. total)
end

---@param id? conn_id connection id
function Handler:page_next(id)
  id = id or self.active_connection

  self:open_pre_hook()
  local count
  self.page_index, count = unpack(vim.fn.Dbee_page(id, tostring(self.page_index + 1)))
  self:open_post_hook(count)
end

---@param id? conn_id connection id
function Handler:page_prev(id)
  id = id or self.active_connection

  self:open_pre_hook()
  local count
  self.page_index, count = unpack(vim.fn.Dbee_page(id, tostring(self.page_index - 1)))
  self:open_post_hook(count)
end

---@param history_id string history id
---@param id? conn_id connection id
function Handler:history(history_id, id)
  id = id or self.active_connection

  self:open_pre_hook()
  self.page_index = 0
  vim.fn.Dbee_history(id, history_id)
  self:open_post_hook()
end

-- get layout for the connection
---@private
---@param id? conn_id connection id
---@return Layout[]
function Handler:get_connection_layout(id)
  id = id or self.active_connection

  ---@param layout_go _LayoutGo[] layout from go
  ---@return Layout[] layout with actions
  local function to_layout(layout_go, parent_id)
    if not layout_go or layout_go == vim.NIL then
      return {}
    end

    -- sort keys
    table.sort(layout_go, function(k1, k2)
      return k1.name < k2.name
    end)

    local new_layouts = {}
    for _, lgo in ipairs(layout_go) do
      -- action 1 executes query or history
      local action_1
      if lgo.type == "table" then
        action_1 = function(cb)
          local details = self:connection_details(id)
          local table_helpers = helpers.get(details.type)
          local helper_keys = {}
          for k, _ in pairs(table_helpers) do
            table.insert(helper_keys, k)
          end
          table.sort(helper_keys)
          -- select a helper to execute
          vim.ui.select(helper_keys, {
            prompt = "select a helper to execute:",
          }, function(selection)
            if selection then
              self:execute(
                helpers.expand_query(
                  table_helpers[selection],
                  { table = lgo.name, schema = lgo.schema, dbname = lgo.database }
                ),
                id
              )
            end
            cb()
          end)
          self:set_active(id)
        end
      elseif lgo.type == "history" then
        action_1 = function(cb)
          self:history(lgo.name, id)
          self:set_active(id)
          cb()
        end
      end
      -- action 2 activates the connection manually
      local action_2 = function(cb)
        self:set_active(id)
        cb()
      end
      -- action_3 is empty

      local l_id = (parent_id or "") .. "__connection_" .. lgo.name .. lgo.schema .. lgo.type .. "__"
      local ly = {
        id = l_id,
        name = lgo.name,
        schema = lgo.schema,
        database = lgo.database,
        type = lgo.type,
        action_1 = action_1,
        action_2 = action_2,
        action_3 = nil,
        children = to_layout(lgo.children, l_id),
      }

      table.insert(new_layouts, ly)
    end

    return new_layouts
  end

  return to_layout(vim.fn.json_decode(vim.fn.Dbee_layout(id)), id)
end

---@return Layout[]
function Handler:layout()
  ---@type Layout[]
  local layout = {}

  for _, conn in ipairs(self:list_connections()) do
    table.insert(layout, {
      id = conn.id,
      name = conn.name,
      type = "database",
      -- set connection as active manually
      action_2 = function(cb)
        self:set_active(conn.id)
        cb()
      end,
      children = function()
        return self:get_connection_layout(conn.id)
      end,
    })
  end

  -- add connection dialog
  table.insert(layout, {
    id = "__add_connection__",
    name = "add connection",
    type = "add",
    action_1 = function(cb)
      local prompt = {
        "name",
        "type",
        "url",
      }
      utils.prompt.open(prompt, {
        title = "Add Connection",
        callback = function(result)
          local ok = pcall(self.add_connection, self, utils.expand_environmet(result) --[[@as connection_details]])
          if ok then
            self.loader_save { result }
          end
          cb()
        end,
      })
    end,
  })

  return layout
end

---@param format "csv"|"json" how to format the result
---@param file string file to write to
---@param id? conn_id connection id
function Handler:save(format, file, id)
  id = id or self.active_connection
  if not format or not file then
    error("save method requires format and file to be set")
  end
  vim.fn.Dbee_save(id, format, file)
end

---@return table<string, fun()>
function Handler:actions()
  return {
    page_next = function()
      self:page_next()
    end,
    page_prev = function()
      self:page_prev()
    end,
  }
end

---@private
function Handler:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }

  local actions = self:actions()

  for act, map in pairs(self.mappings) do
    local action = actions[act]
    if action and type(action) == "function" then
      vim.keymap.set(map.mode, map.key, action, map_options)
    end
  end
end

-- fill the Ui interface - open results
function Handler:open()
  if not self.winid or not vim.api.nvim_win_is_valid(self.winid) then
    self.winid = self.win_cmd()
  end

  -- if buffer doesn't exist, create it
  local bufnr = self.bufnr
  if not bufnr or not vim.api.nvim_buf_is_valid(bufnr) then
    bufnr = vim.api.nvim_create_buf(false, true)
  end
  vim.api.nvim_win_set_buf(self.winid, bufnr)
  vim.api.nvim_set_current_win(self.winid)
  vim.api.nvim_buf_set_name(bufnr, "dbee-results-" .. tostring(os.clock()))

  -- set keymaps
  self:map_keys(bufnr)

  local win_opts = {
    wrap = false,
    winfixheight = true,
    winfixwidth = true,
    number = false,
  }
  for opt, val in pairs(win_opts) do
    vim.api.nvim_win_set_option(self.winid, opt, val)
  end

  self.bufnr = bufnr

  -- register in go
  vim.fn.Dbee_set_results_buf(bufnr)
end

function Handler:close()
  pcall(vim.api.nvim_win_close, self.winid, false)
end

return Handler
