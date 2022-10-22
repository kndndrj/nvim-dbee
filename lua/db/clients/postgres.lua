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

---@return { string: string } helpers list of table helpers
function PostgresClient:table_helpers()
  local basic_constraint_query = [[
    SELECT tc.constraint_name, tc.table_name, kcu.column_name, ccu.table_name AS foreign_table_name, ccu.column_name AS foreign_column_name, rc.update_rule, rc.delete_rule
    FROM 
         information_schema.table_constraints AS tc
         JOIN information_schema.key_column_usage AS kcu
           ON tc.constraint_name = kcu.constraint_name
         JOIN information_schema.referential_constraints as rc
           ON tc.constraint_name = rc.constraint_name
         JOIN information_schema.constraint_column_usage AS ccu
           ON ccu.constraint_name = tc.constraint_name ]]

  return {
    List = 'select * from "{table}" LIMIT 500',
    Columns = "select * from information_schema.columns where table_name='{table}' and table_schema='{schema}'",
    Indexes = "SELECT * FROM pg_indexes where tablename='{table}' and schemaname='{schema}'",
    ["Foreign Keys"] = basic_constraint_query
      .. "WHERE constraint_type = 'FOREIGN KEY' and tc.table_name = '{table}' and tc.table_schema = '{schema}'",
    References = basic_constraint_query
      .. "WHERE constraint_type = 'FOREIGN KEY' and ccu.table_name = '{table}' and tc.table_schema = '{schema}'",
    ["Primary Keys"] = basic_constraint_query
      .. "WHERE constraint_type = 'PRIMARY KEY' and tc.table_name = '{table}' and tc.table_schema = '{schema}'",
  }
end

return PostgresClient
