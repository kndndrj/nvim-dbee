local helpers = require("dbee.helpers")

---@alias conn_id string
---@alias connection_details { name: string, type: string, url: string, id: conn_id }
--
---@class _LayoutGo
---@field name string display name
---@field type ""|"table"|"history" type of layout -> this infers action
---@field schema? string parent schema
---@field database? string parent database
---@field children? Layout[] child layout nodes

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
local Handler = {}

---@param opts? { connections: connection_details[], win_cmd: string|fun():integer }
---@return Handler
function Handler:new(opts)
  opts = opts or {}

  local cons = opts.connections or {}

  local active = "ž" -- this MUST get overwritten
  local connections = {}
  for _, con in ipairs(cons) do
    if not con.url then
      error("url needs to be set!")
    end
    if not con.type then
      error("no type")
    end

    con.name = con.name or "[empty name]"
    local id = con.name .. con.type

    con.id = id
    if id < active then
      active = id
    end

    -- register in go
    vim.fn.Dbee_register_connection(id, con.url, con.type)

    connections[id] = con
  end

  local win_cmd
  if type(opts.win_cmd) == "string" then
    win_cmd = function()
      vim.cmd(opts.win_cmd)
      return vim.api.nvim_get_current_win()
    end
  elseif type(opts.win_cmd) == "function" then
    win_cmd = opts.win_cmd
  else
    win_cmd = function()
      vim.cmd("bo 15split")
      return vim.api.nvim_get_current_win()
    end
  end

  -- class object
  local o = {
    connections = connections,
    active_connection = active,
    page_index = 0,
    win_cmd = win_cmd,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param connection connection_details
function Handler:add_connection(connection)
  if not connection.url then
    error("url needs to be set!")
    return
  end
  if not connection.type then
    error("no type")
    return
  end

  connection.name = connection.name or "[empty name]"

  local id = connection.name .. connection.type

  for _, con in pairs(self.connections) do
    if con.id == id then
      return
    end
  end

  -- register in go
  vim.fn.Dbee_register_connection(id, connection.url, connection.type)

  self.connections[id] = connection
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
---@param id? conn_id connection id
---@return Layout[]
function Handler:layout(id)
  id = id or self.active_connection

  ---@param layout_go _LayoutGo[] layout from go
  ---@return Layout[] layout with actions
  local function to_layout(layout_go)
    if not layout_go or layout_go == vim.NIL then
      return {}
    end

    local _new_layouts = {}
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

      local _ly = {
        name = lgo.name,
        schema = lgo.schema,
        database = lgo.database,
        type = lgo.type,
        action_1 = action_1,
        action_2 = action_2,
        action_3 = nil,
        children = to_layout(lgo.children),
      }

      table.insert(_new_layouts, _ly)
    end

    return _new_layouts
  end

  return to_layout(vim.fn.json_decode(vim.fn.Dbee_layout(id)))
end

---@param format "csv"|"json" how to format the result
---@param file string file to write to
---@param id? conn_id connection id
function Handler:save(format, file, id)
  id = id or self.active_connection
  -- TODO
  vim.fn.Dbee_save(id, format, file)
end

-- fill the Ui interface - open results
---@param winid? integer
function Handler:open(winid)
  winid = winid or self.winid
  if not winid or not vim.api.nvim_win_is_valid(winid) then
    winid = self.win_cmd()
  end

  -- if buffer doesn't exist, create it
  local bufnr = self.bufnr
  if not bufnr or not vim.api.nvim_buf_is_valid(bufnr) then
    bufnr = vim.api.nvim_create_buf(false, true)
  end
  vim.api.nvim_win_set_buf(winid, bufnr)
  vim.api.nvim_set_current_win(winid)
  vim.api.nvim_buf_set_name(bufnr, "dbee-results-" .. tostring(os.clock()))

  local win_opts = {
    wrap = false,
    winfixheight = true,
    winfixwidth = true,
    number = false,
  }
  for opt, val in pairs(win_opts) do
    vim.api.nvim_win_set_option(winid, opt, val)
  end

  self.winid = winid
  self.bufnr = bufnr

  -- register in go
  vim.fn.Dbee_set_results_buf(bufnr)
end

-- fill the Ui interface - close results
function Handler:close()
  vim.fn.Dbee_results("close")
end

return Handler
