local utils = require("dbee.utils")

---@alias table_helpers table<string, string>

---@class Helpers
---@field private extras table<string, table_helpers> extra table helpers per type
local Helpers = {}

---@param opts? { extras: table<string, table_helpers> }
---@return Helpers
function Helpers:new(opts)
  opts = opts or {}

  local o = {
    extras = opts.extras or {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__postgres()
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

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__mysql()
  return {
    List = "SELECT * from `{table}` LIMIT 500",
    Columns = "DESCRIBE `{table}`",
    Indexes = "SHOW INDEXES FROM `{table}`",
    ["Foreign Keys"] = "SELECT * FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA = '{schema}' AND TABLE_NAME = '{table}' AND CONSTRAINT_TYPE = 'FOREIGN KEY'",
    ["Primary Keys"] = "SELECT * FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA = '{schema}' AND TABLE_NAME = '{table}' AND CONSTRAINT_TYPE = 'PRIMARY KEY'",
  }
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__sqlite()
  return {
    List = 'select * from "{table}" LIMIT 500',
    Indexes = "SELECT * FROM pragma_index_list('{table}')",
    ["Foreign Keys"] = "SELECT * FROM pragma_foreign_key_list('{table}')",
    ["Primary Keys"] = "SELECT * FROM pragma_index_list('{table}') WHERE origin = 'pk'",
  }
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__redis()
  return {
    List = "KEYS *",
  }
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__mongo()
  return {
    List = '{"find": "{table}"}',
  }
end

---@param type string
---@param vars { table: string, schema: string, dbname: string }
---@return table_helpers helpers list of table helpers
function Helpers:get(type, vars)
  local helpers
  if type == "postgres" then
    helpers = self:__postgres()
  elseif type == "mysql" then
    helpers = self:__mysql()
  elseif type == "sqlite" then
    helpers = self:__sqlite()
  elseif type == "redis" then
    helpers = self:__redis()
  elseif type == "mongo" then
    helpers = self:__mongo()
  end

  if not helpers then
    error("unsupported table type for helpers: " .. type)
  end

  -- apply extras
  local ex = self.extras[type] or {}
  helpers = vim.tbl_deep_extend("force", helpers, ex)

  return self:expand(helpers, vars or {}) --[[@as table_helpers]]
end

---@private
---@param obj string|table_helpers
---@param vars { table: string, schema: string, dbname: string }
---@return string|table_helpers # returns depending on what's passed in
function Helpers:expand(obj, vars)
  local function exp(o)
    if type(o) ~= "string" then
      return o
    end
    local ret = o
    for key, val in pairs(vars) do
      ret = ret:gsub("{" .. key .. "}", val)
    end
    return ret
  end

  if type(obj) == "table" then
    return vim.tbl_map(exp, obj)
  end

  return exp(obj)
end

---@param helpers table<string, table_helpers> extra helpers to add (per type)
function Helpers:add(helpers)
  local ext = {}
  for t, hs in pairs(helpers) do
    ext[utils.type_alias(t)] = hs
  end
  self.extras = vim.tbl_deep_extend("force", self.extras, ext)
end

return Helpers
