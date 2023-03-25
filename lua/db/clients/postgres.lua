-- Everything in this file is executed inside a separate lua process.
-- package paths are already handled, just make sure not to run any commands uv.new_work can't handle

---@class Client
---@field url string
---@field type string
local PostgresClient = {}

---@param opts? { url: string }
function PostgresClient:new(opts)
  opts = opts or {}

  if not opts.url then
    print("url needs to be set!")
    return
  end

  local o = {
    url = opts.url,
    type = "postgres",
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@param query string query to execute
---@return grid
function PostgresClient:execute(query)
  local driver = require("luasql.postgres")
  local env = assert(driver.postgres())
  local con = assert(env:connect(self.url))

  -- retrieve a cursor
  local cur = assert(con:execute(query))

  -- get headers
  ---@type string[]
  local header = cur:getcolnames()

  -- get all rows
  ---@type string[][]
  local results = {}
  while true do
    ---@type string[]
    local row = cur:fetch({}, "n")
    if not row then
      break
    end
    table.insert(results, row)
  end

  -- close everything
  cur:close()
  con:close()
  env:close()

  return { header = header, rows = results }
end

---@return schemas
function PostgresClient:schemas()
  local tables_and_views_query = [[
    SELECT table_schema, table_name FROM information_schema.tables UNION ALL
    SELECT schemaname, matviewname FROM pg_matviews;
  ]]
  local schemas_and_tables = self:execute(tables_and_views_query)

  local header_map = {}
  for i, h in ipairs(schemas_and_tables.header) do
    header_map[h] = i
  end

  local ret = {}
  for _, t in ipairs(schemas_and_tables.rows) do
    if not ret[t[header_map["table_schema"]]] then
      ret[t[header_map["table_schema"]]] = {}
    end
    if t[header_map["table_schema"]] and t[header_map["table_name"]] then
      table.insert(ret[t[header_map["table_schema"]]], t[header_map["table_name"]])
    end
  end
  return ret
end

function PostgresClient:get_default_schema()
  return "public"
end

return PostgresClient
