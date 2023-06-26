local utils = require("dbee.utils")

---@alias conn_id string
---@alias connection_details { name: string, type: string, url: string, id: conn_id, page_size: integer }
--
---@class _LayoutGo
---@field name string display name
---@field type ""|"table"|"history" type of layout -> this infers action
---@field schema? string parent schema
---@field database? string parent database
---@field children? Layout[] child layout nodes

-- Conn is a 1:1 mapping to go's connections
---@class Conn
---@field private ui Ui
---@field private helpers Helpers
---@field private __original connection_details original unmodified fields passed on initialization (params)
---@field private id conn_id
---@field private name string
---@field private type string --TODO enum?
---@field private page_size integer
---@field private page_index integer index of the current page
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
function Conn:execute(query)
  self.on_exec()

  self:__wrap_open(function(_)
    self.page_index = 0
    vim.fn.Dbee_execute(self.id, query)
    return self.page_index, nil
  end)
end

---@param history_id string history id
function Conn:history(history_id)
  self.on_exec()

  self:__wrap_open(function(_)
    self.page_index = 0
    vim.fn.Dbee_history(self.id, history_id)
    return self.page_index, nil
  end)
end

---@return integer # total number of pages
function Conn:page_next()
  self.on_exec()

  local count
  self:__wrap_open(function(_)
    self.page_index, count = unpack(vim.fn.Dbee_page(self.id, tostring(self.page_index + 1)))
    return self.page_index, count
  end)
  return count
end

---@return integer # total number of pages
function Conn:page_prev()
  self.on_exec()

  local count
  self:__wrap_open(function(_)
    self.page_index, count = unpack(vim.fn.Dbee_page(self.id, tostring(self.page_index - 1)))
    return self.page_index, count
  end)
  return count
end

---@param format "csv"|"json"|"table" format of the output
---@param output "file"|"yank"|"buffer" where to pipe the results
---@param arg any argument for specific format/output combination - example file path or buffer number
function Conn:store(format, output, arg)
  format = tostring(format) or ""
  output = tostring(output) or ""
  arg = tostring(arg) or ""

  vim.fn.Dbee_store(self.id, format, output, arg)
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
      return k1.name < k2.name
    end)

    local new_layouts = {}
    for _, lgo in ipairs(layout_go) do
      -- action 1 executes query or history
      local action_1
      if lgo.type == "table" then
        action_1 = function(cb)
          local helpers = self.helpers:get(self.type, { table = lgo.name, schema = lgo.schema, dbname = lgo.database })
          vim.ui.select(utils.sorted_keys(helpers), {
            prompt = "select a helper to execute:",
          }, function(selection)
            if selection then
              self:execute(helpers[selection])
            end
            cb()
          end)
        end
      elseif lgo.type == "history" then
        action_1 = function(cb)
          self:history(lgo.name)
          cb()
        end
      end
      -- action 2 activates the connection manually TODO
      local action_2 = function(cb)
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

  return to_layout(vim.fn.json_decode(vim.fn.Dbee_layout(self.id)), self.id)
end

---@private
-- wraps a function to add buffer decorations
-- fn can optionally return page/total
---@param fn fun(bufnr: integer):integer?,integer?
function Conn:__wrap_open(fn)
  -- open ui window
  local winid, bufnr = self.ui:open()

  -- register buffer in go
  vim.fn.Dbee_set_results_buf(bufnr)

  vim.api.nvim_buf_set_option(bufnr, "modifiable", true)
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, true, { "Loading..." })
  vim.api.nvim_buf_set_option(bufnr, "modifiable", false)

  local page, total = fn(bufnr)

  local tot = "?"
  if total then
    tot = tostring(total + 1)
  end
  local pg = "0"
  if page then
    pg = tostring(page + 1)
  end

  -- set winbar
  vim.api.nvim_win_set_option(winid, "winbar", "%=" .. pg .. "/" .. tot)
end

return Conn
