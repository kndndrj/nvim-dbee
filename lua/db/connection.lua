---@alias schemas { string: string[] }

---@class Connection
---@field meta { [string]: any } Table that holds metadata
---@field type string
---@field history { query: string, file: string }[]
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
    history = {},
    ui = opts.ui,
    type = opts.type,
    id = id,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param query string
---@param format? "preview"|"csv" format of the output (default: preview)
---@param callback? fun() optional callback to execute when results are ready
function Connection:execute(query, format, callback)
  if not format then
    format = "preview"
  end

  local history_index = #self.history + 1

  local history_file = "/tmp/nvim-db-tmp" .. tostring(os.clock())
  self.history[history_index] = { query = query, file = history_file }

  -- call Go function here
  vim.fn.Dbee_execute(self.id, query)

  -- open the first page
  self.page_index = 0
  local bufnr = self.ui:open()
  vim.fn.Dbee_display(self.id, tostring(self.page_index), tostring(bufnr))

  -- TODO trigger this after the result is ready
  if type(callback) == "function" then
    callback()
  end
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

function Connection:hist(id)
  -- call Go function here
  vim.fn.Dbee_history(self.id, id)

  -- open the first page
  self.page_index = 0
  local bufnr = self.ui:open()
  vim.fn.Dbee_display(self.id, tostring(self.page_index), tostring(bufnr))
end

function Connection:list_hist()
  return vim.fn.Dbee_list_history(self.id)
end

---@return schemas
function Connection:schemas()
  return vim.fn.Dbee_get_schema(self.id)
end

---@param index integer history index
function Connection:display_history(index)
  local file = self.history[index].file

  self.ui:open()
  local winid = self.ui.winid
  vim.api.nvim_set_current_win(winid)
  vim.api.nvim_command("e " .. file)
end

return Connection
