---@alias schemas { string: string[] }

---@class Connection
---@field meta { [string]: any } Table that holds metadata
---@field type string
---@field private ui UI
---@field private id string id to call the go side of the client
---@field private page_index integer current page
local Connection = {}

---@param opts? { name: string, type: string, url: string, ui: UI }
function Connection:new(opts)
  opts = opts or {}

  if not opts.url then
    print("url needs to be set!")
    return
  end

  if not opts.type then
    print("no type")
    return
  end

  if opts.ui == nil then
    print("no UI provided to connection")
    return
  end

  -- register the client on go side (ude id to match)
  local id = opts.type .. tostring(os.clock())
  vim.fn.Dbee_register_client(id, opts.url, opts.type)

  local o = {
    meta = {
      name = opts.name or "[empty name]",
      type = opts.type,
    },
    ui = opts.ui,
    type = opts.type,
    id = id,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param query string
function Connection:execute(query)
  -- call Go function here
  vim.fn.Dbee_execute(self.id, query)

  -- open the first page
  self.page_index = 0
  local bufnr = self.ui:open()
  vim.fn.Dbee_display(self.id, tostring(self.page_index), tostring(bufnr))
end

function Connection:page_next()
  -- open ui
  local bufnr = self.ui:open()

  -- go func returns selected page
  self.page_index = vim.fn.Dbee_display(self.id, tostring(self.page_index + 1), tostring(bufnr))
end

function Connection:page_prev()
  -- open ui
  local bufnr = self.ui:open()

  self.page_index = vim.fn.Dbee_display(self.id, tostring(self.page_index - 1), tostring(bufnr))
end

---@param id string history id
function Connection:display_history(id)
  -- call Go function here
  vim.fn.Dbee_history(self.id, id)

  -- open the first page
  self.page_index = 0
  local bufnr = self.ui:open()
  vim.fn.Dbee_display(self.id, tostring(self.page_index), tostring(bufnr))
end

function Connection:history()
  local h = vim.fn.Dbee_list_history(self.id)
  if not h or h == vim.NIL then
    return {}
  end
  return h
end

---@return schemas
function Connection:schemas()
  return vim.fn.Dbee_get_schema(self.id)
end

---@param format "csv"|"json" how to format the result
---@param file string file to write to
function Connection:save(format, file)
  -- TODO
  -- open ui
  local bufnr = self.ui:open()
  vim.fn.Dbee_write(self.id, tostring(bufnr))
end

return Connection
