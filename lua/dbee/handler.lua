---@alias connection_details { name: string, type: string, url: string, id: integer }
---@alias layout { name: string, schema: string, database: string, type: "record"|"table"|"history"|"scratch", children: layout[] }

-- Handler is a wrapper around the go code
-- it is the central part of the plugin and manages connections.
-- almost all functions take the connection id as their argument.
---@class Handler
---@field private connections { integer: connection_details } id - connection mapping
---@field private active_connection integer last called connection
---@field private last_id integer last id number
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

  local connections = {}
  local last_id = 0
  for id, con in ipairs(cons) do
    if not con.url then
      error("url needs to be set!")
    end
    if not con.type then
      error("no type")
    end

    con.name = con.name or "[empty name]"
    con.id = id

    -- register in go
    vim.fn.Dbee_register_connection(tostring(id), con.url, con.type)

    connections[id] = con
    last_id = id
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
    last_id = last_id,
    active_connection = 1,
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
    print("url needs to be set!")
    return
  end
  if not connection.type then
    print("no type")
    return
  end

  local name = connection.name or "[empty name]"

  for _, con in pairs(self.connections) do
    if con.name == name then
      return
    end
  end

  self.last_id = self.last_id + 1
  connection.id = self.last_id

  -- register in go
  vim.fn.Dbee_register_connection(tostring(self.last_id), connection.url, connection.type)

  self.connections[self.last_id] = connection
end

---@param id integer connection id
function Handler:set_active(id)
  if not id or self.connections[id] == nil then
    print("no id specified!")
    return
  end
  self.active_connection = id
end

---@return connection_details[] list of connections
function Handler:list_connections()
  local cons = {}
  for _, con in pairs(self.connections) do
    table.insert(cons, con)
  end
  return cons
end

---@return connection_details
---@param id? integer connection id
function Handler:connection_details(id)
  id = id or self.active_connection
  return self.connections[id]
end

---@param query string query to execute
---@param id? integer connection id
function Handler:execute(query, id)
  id = id or self.active_connection

  self:open_hook()
  self.page_index = 0
  vim.fn.Dbee_execute(tostring(id), query)
end

---@private
-- called when anything needs to be displayed on screen
function Handler:open_hook()
  self:open()
  vim.api.nvim_buf_set_option(self.bufnr, "modifiable", true)
  vim.api.nvim_buf_set_lines(self.bufnr, 0, -1, true, { "Loading..." })
  vim.api.nvim_buf_set_option(self.bufnr, "modifiable", false)
end

---@param id? integer connection id
function Handler:page_next(id)
  id = id or self.active_connection

  self:open_hook()
  self.page_index = vim.fn.Dbee_page(tostring(id), tostring(self.page_index + 1))
end

---@param id? integer connection id
function Handler:page_prev(id)
  id = id or self.active_connection

  self:open_hook()
  self.page_index = vim.fn.Dbee_page(tostring(id), tostring(self.page_index - 1))
end

---@param history_id string history id
---@param id? integer connection id
function Handler:history(history_id, id)
  id = id or self.active_connection

  self:open_hook()
  self.page_index = 0
  vim.fn.Dbee_history(tostring(id), history_id)
end

---@param id? integer connection id
function Handler:list_history(id)
  id = id or self.active_connection

  local h = vim.fn.Dbee_list_history(tostring(id))
  if not h or h == vim.NIL then
    return {}
  end
  return h
end

---@param id? integer connection id
---@return layout[]
function Handler:schema(id)
  id = id or self.active_connection
  return vim.fn.json_decode(vim.fn.Dbee_schema(tostring(id)))
end

-- get layout for the connection (combines history and schema)
---@param id? integer connection id
---@return layout[]
function Handler:layout(id)
  id = id or self.active_connection

  local structure = vim.fn.json_decode(vim.fn.Dbee_schema(tostring(id)))

  ---@type layout[]
  local history_children = {}
  for _, h in ipairs(self:list_history(id)) do
    ---@type layout
    local sch = {
      name = h,
      type = "history",
    }
    table.insert(history_children, sch)
  end

  return {
    { name = "structure", type = "record", children = structure },
    { name = "history",   type = "record", children = history_children },
  }
end

---@param format "csv"|"json" how to format the result
---@param file string file to write to
---@param id? integer connection id
function Handler:save(format, file, id)
  id = id or self.active_connection
  -- TODO
  vim.fn.Dbee_save(tostring(id))
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
  vim.fn.Dbee_set_results_buf(tostring(bufnr))
end

-- fill the Ui interface - close results
function Handler:close()
  vim.fn.Dbee_results("close")
end

return Handler
