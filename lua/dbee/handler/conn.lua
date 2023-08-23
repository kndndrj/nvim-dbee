local utils = require("dbee.utils")
local callbacker = require("dbee.handler.__callbacks")

---@alias conn_id string
---@alias connection_details { name: string, type: string, url: string, id: conn_id, page_size: integer }
--
---@class _LayoutGo
---@field name string display name
---@field type ""|"table"|"history"|"database_switch" type of layout -> this infers action
---@field type ""|"table"|"history"|"view" type of layout -> this infers action
---@field schema? string parent schema
---@field database? string parent database
---@field children? _LayoutGo[] child layout nodes
---@field pick_items?  string[] pick items

-- Conn is a 1:1 mapping to go's connections
---@class Conn
---@field private ui Ui
---@field private helpers Helpers
---@field private __original connection_details original unmodified fields passed on initialization (params)
---@field private id conn_id
---@field private name string
---@field private type string type of connection, e.g. "postgres", "mysql", "sqlite" TODO: enum?
---@field private page_size integer
---@field private page_index integer index of the current page
---@field private page_ammount integer number of pages in the current result set
---@field private on_exec fun() callback which gets triggered on any action
local Conn = {}

---@param ui Ui
---@param helpers Helpers
---@param params connection_details
---@param opts? { fallback_page_size: integer, on_exec: fun() }
---@return Conn
function Conn:new(ui, helpers, params, opts)
  params = params or {}
  opts = opts or {}

  local expanded = utils.expand_environment(params)

  -- validation
  if not ui then
    error("no Ui provided to Conn!")
  end
  if not helpers then
    error("no Helpers provided to Conn!")
  end
  if not expanded.url then
    error("url needs to be set!")
  end
  if not expanded.type or expanded.type == "" then
    error("no type")
  end

  -- get needed fields
  local name = expanded.name
  if not name or name == "" then
    name = "[no name]"
  end
  local type = utils.type_alias(expanded.type)
  local id = expanded.id or ("__master_connection_id_" .. expanded.name .. expanded.type .. "__")
  local page_size = params.page_size or opts.fallback_page_size or 100

  -- register in go
  local ok = vim.fn.Dbee_register_connection(id, expanded.url, type, tostring(page_size))
  if not ok then
    error("problem adding connection")
  end

  params.id = id

  -- class object
  local o = {
    ui = ui,
    helpers = helpers,
    __original = params,
    id = id,
    name = name,
    type = type,
    page_size = page_size,
    page_index = 0,
    page_ammount = 0,
    on_exec = opts.on_exec or function() end,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

function Conn:close()
  -- TODO
end

---@return connection_details
function Conn:details()
  return {
    id = self.id,
    name = self.name,
    -- url shouldn't be seen as expanded - it has secrets
    url = self.__original.url,
    type = self.type,
    page_size = self.page_size,
  }
end

---@return connection_details
function Conn:original_details()
  return self.__original
end

---@param query string query to execute
---@param cb? fun() callback to execute when finished
function Conn:execute(query, cb)
  cb = cb or function() end
  self.on_exec()

  local cb_id = tostring(math.random(10000))
  callbacker.register(cb_id, function()
    self:show_page(0)
    cb()
  end)

  self.page_index = 0
  self.page_ammount = 0

  vim.fn.Dbee_execute(self.id, query, cb_id)
end

---@param history_id string history id
---@param cb? fun() callback to execute when finished
function Conn:history(history_id, cb)
  cb = cb or function() end
  self.on_exec()

  local cb_id = tostring(math.random(10000))
  callbacker.register(cb_id, function()
    self:show_page(0)
    cb()
  end)

  self.page_index = 0
  self.page_ammount = 0

  vim.fn.Dbee_history(self.id, history_id, cb_id)
end

---@param name string name of the database
function Conn:switch_database(name)
  vim.fn.Dbee_switch_database(self.id, name)
end

function Conn:page_next()
  self.on_exec()

  self.page_index = self:show_page(self.page_index + 1)
end

function Conn:page_prev()
  self.on_exec()

  self.page_index = self:show_page(self.page_index - 1)
end

--- Displays a page of the current result in the results buffer
---@private
---@param page integer zero based page index
---@return integer # current page
function Conn:show_page(page)
  -- calculate the ranges
  if page < 0 then
    page = 0
  end
  if page > self.page_ammount then
    page = self.page_ammount
  end
  local from = self.page_size * page
  local to = self.page_size * (page + 1)

  -- open ui window and register it's buffer in go
  local winid, bufnr = self.ui:open()
  vim.fn.Dbee_set_results_buf(bufnr)

  -- call go function
  local length = vim.fn.Dbee_get_current_result(self.id, tostring(from), tostring(to))

  -- adjust page ammount
  self.page_ammount = math.floor(length / self.page_size)
  if length % self.page_size == 0 and self.page_ammount ~= 0 then
    self.page_ammount = self.page_ammount - 1
  end

  -- set winbar status
  vim.api.nvim_win_set_option(winid, "winbar", "%=" .. tostring(page + 1) .. "/" .. tostring(self.page_ammount + 1))

  return page
end

---@param format "csv"|"json"|"table" format of the output
---@param output "file"|"yank"|"buffer" where to pipe the results
---@param opts { from: number, to: number, extra_arg: any }
function Conn:store(format, output, opts)
  opts = opts or {}

  -- options:
  local from = opts.from or 0
  local to = opts.to or -1
  local arg = opts.extra_arg or ""

  vim.fn.Dbee_store(self.id, format, output, tostring(from), tostring(to), tostring(arg))
end

-- get layout for the connection
---@return Layout[]
function Conn:layout()
  ---@param layout_go _LayoutGo[] layout from go
  ---@return Layout[] layout with actions
  local function to_layout(layout_go, parent_id)
    if not layout_go or layout_go == vim.NIL then
      return {}
    end

    -- sort keys
    table.sort(layout_go, function(k1, k2)
      return k1.type .. k1.name < k2.type .. k2.name
    end)

    local new_layouts = {}
    for _, lgo in ipairs(layout_go) do
      local l_id = (parent_id or "") .. "__connection_" .. lgo.name .. lgo.schema .. lgo.type .. "__"
      ---@type Layout
      local ly = {
        id = l_id,
        name = lgo.name,
        schema = lgo.schema,
        database = lgo.database,
        type = lgo.type,
        pick_items = lgo.pick_items,
        action_2 = function(cb)
          cb()
        end,
        children = to_layout(lgo.children, l_id),
      }

      if lgo.type == "table" then
        ly.action_1 = function(cb, selection)
          local helpers = self.helpers:get(self.type, { table = lgo.name, schema = lgo.schema, dbname = lgo.database })
          self:execute(helpers[selection], cb)
        end
        ly.pick_items = function()
          return self.helpers:list(self.type)
        end
        ly.pick_title = "Select a Query"
      elseif lgo.type == "history" then
        ly.action_1 = function(cb)
          self:history(lgo.name, cb)
        end
      elseif lgo.type == "database_switch" then
        ly.action_1 = function(cb, selection)
          self:switch_database(selection)
          cb()
        end
        ly.pick_title = "Select a Database"
      end

      table.insert(new_layouts, ly)
    end

    return new_layouts
  end

  return to_layout(vim.fn.json_decode(vim.fn.Dbee_layout(self.id)), self.id)
end

return Conn
