---@alias schemas { string: string[] }

---@class Connection
---@field meta { [string]: any } Table that holds metadata
---@field client Client
---@field history { query: string, file: string }[]
---@field ui UI
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

  ---@type boolean, Client
  local ok, Client = pcall(require, "db.clients." .. opts.type)
  if not ok then
    print("client not found for type " .. opts.type)
    return
  end

  if opts.ui == nil then
    print("no UI provided to connection")
    return
  end

  local client = Client:new { url = opts.url }


  local o = {
    meta = {
      name = opts.name or "[empty name]",
      type = opts.type,
    },
    client = client,
    history = {},
    ui = opts.ui,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param query string
---@param format? "preview"|"csv" format of the output (default: preview)
---@param callback? fun() optional callback to execute when results are ready
function Connection:execute_to_result(query, format, callback)
  if not format then
    format = "preview"
  end

  local history_index = #self.history + 1

  self.history[history_index] = { query = query }

  local uv = vim.loop
  local ctx = uv.new_work(
    -- this executes with it's own lua interpreter (new thread)
    ---@param package_path string
    ---@param package_cpath string
    ---@param client_type string
    ---@param url string
    ---@param sql string
    ---@param fmt "preview"|"csv"
    function(package_path, package_cpath, client_type, url, sql, fmt)
      ---@diagnostic disable-next-line redefined local
      local uv = vim.loop

      -- check for all arguments
      if not package_path or not package_cpath or not sql or not url then
        return
      end

      -- needed for external imports package.path = package_path package.cpath = package_cpath
      package.path = package_path
      package.cpath = package_cpath

      ---@type boolean, Client
      local ok, Client = pcall(require, "db.clients." .. client_type)
      if not ok then
        print("client not found for type " .. client_type)
        return
      end

      ---@type Client
      ---@diagnostic disable-next-line
      local client = Client:new { url = url }
      if not client then
        return
      end

      ---@type grid
      local result = client:execute(sql)

      -- format
      if fmt == "preview" then
        local f = require("db.format")
        result = f.display(result)
      elseif fmt == "csv" then
      end

      -- encode for passing between threads
      local result_encoded = vim.json.encode(result)
      local history_file = "/tmp/nvim-db-tmp" .. tostring(os.clock())

      -- write to history file async
      uv.new_thread(function(file_path, encoded_res)
        local r = vim.json.decode(encoded_res)
        local file = assert(io.open(file_path, "w"))
        -- io.output(file)
        for _, l in ipairs(r) do
          file:write(l .. "\n")
        end
        file:close()
      end, history_file, result_encoded)

      -- return encoded table
      return result_encoded, history_file
    end,
    -- this executes inside main thread
    vim.schedule_wrap(function(encoded, history_file_path)
      -- get encoded table from the thread
      if not encoded then
        return
      end
      ---@type string[]
      local result = vim.json.decode(encoded)

      self.history[history_index].file = history_file_path

      local bufnr = self.ui:open()
      vim.api.nvim_buf_set_option(bufnr, "modifiable", true)
      vim.api.nvim_buf_set_lines(bufnr, 0, 0, true, result)
      vim.api.nvim_buf_set_option(bufnr, "modifiable", false)

      if type(callback) == "function" then
        callback()
      end
    end)
  )
  vim.pretty_print(query)
  vim.pretty_print(format)
  vim.pretty_print(self.client.url)
  vim.pretty_print(self.client.type)

  uv.queue_work(ctx, package.path, package.cpath, self.client.type, self.client.url, query, format)
end

---@return schemas
function Connection:schemas()
  return self.client:schemas()
end

---@return string
function Connection:type()
  return self.client.type
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
