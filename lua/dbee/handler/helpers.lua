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

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__bigquery()
  return {
    List = "SELECT * FROM `{table}` LIMIT 500",
    Columns = "SELECT * FROM `{schema}.INFORMATION_SCHEMA.COLUMNS` WHERE TABLE_SCHEMA = '{schema}' AND TABLE_NAME = '{table}'",
  }
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__sqlserver()
  local column_summary_query = [[
      select c.column_name + ' (' +
          isnull(( select 'PK, ' from information_schema.table_constraints as k join information_schema.key_column_usage as kcu on k.constraint_name = kcu.constraint_name where constraint_type='PRIMARY KEY' and k.table_name = c.table_name and kcu.column_name = c.column_name), '') +
          isnull(( select 'FK, ' from information_schema.table_constraints as k join information_schema.key_column_usage as kcu on k.constraint_name = kcu.constraint_name where constraint_type='FOREIGN KEY' and k.table_name = c.table_name and kcu.column_name = c.column_name), '') +
          data_type + coalesce('(' + rtrim(cast(character_maximum_length as varchar)) + ')','(' + rtrim(cast(numeric_precision as varchar)) + ',' + rtrim(cast(numeric_scale as varchar)) + ')','(' + rtrim(cast(datetime_precision as varchar)) + ')','') + ', ' +
          case when is_nullable = 'YES' then 'null' else 'not null' end + ')' as Columns
      from information_schema.columns c where c.table_name='{table}' and c.TABLE_SCHEMA = '{schema}' ]]

  local foreign_keys_query = [[
      SELECT c.constraint_name
         ,kcu.column_name as column_name
         ,c2.table_name as foreign_table_name
         ,kcu2.column_name as foreign_column_name
      from   information_schema.table_constraints c
             inner join information_schema.key_column_usage kcu
               on c.constraint_schema = kcu.constraint_schema
                  and c.constraint_name = kcu.constraint_name
             inner join information_schema.referential_constraints rc
               on c.constraint_schema = rc.constraint_schema
                  and c.constraint_name = rc.constraint_name
             inner join information_schema.table_constraints c2
               on rc.unique_constraint_schema = c2.constraint_schema
                  and rc.unique_constraint_name = c2.constraint_name
             inner join information_schema.key_column_usage kcu2
               on c2.constraint_schema = kcu2.constraint_schema
                  and c2.constraint_name = kcu2.constraint_name
                  and kcu.ordinal_position = kcu2.ordinal_position
      where  c.constraint_type = 'FOREIGN KEY'
      and c.TABLE_NAME = '{table}' and c.TABLE_SCHEMA = '{schema}' ]]

  local references_query = [[
      select kcu1.constraint_name as constraint_name
          ,kcu1.table_name as foreign_table_name
          ,kcu1.column_name as foreign_column_name
          ,kcu2.column_name as column_name
      from information_schema.referential_constraints as rc
      inner join information_schema.key_column_usage as kcu1
          on kcu1.constraint_catalog = rc.constraint_catalog
          and kcu1.constraint_schema = rc.constraint_schema
          and kcu1.constraint_name = rc.constraint_name
      inner join information_schema.key_column_usage as kcu2
          on kcu2.constraint_catalog = rc.unique_constraint_catalog
          and kcu2.constraint_schema = rc.unique_constraint_schema
          and kcu2.constraint_name = rc.unique_constraint_name
          and kcu2.ordinal_position = kcu1.ordinal_position
      where kcu2.table_name='{table}' and kcu2.table_schema = '{schema}' ]]

  local primary_keys_query = [[
       select tc.constraint_name, kcu.column_name
       from
           information_schema.table_constraints AS tc
           JOIN information_schema.key_column_usage AS kcu
             ON tc.constraint_name = kcu.constraint_name
           JOIN information_schema.constraint_column_usage AS ccu
             ON ccu.constraint_name = tc.constraint_name
      where constraint_type = 'PRIMARY KEY'
      and tc.table_name = '{table}' and tc.table_schema = '{schema}' ]]

  local constraints_query = [[
      SELECT u.CONSTRAINT_NAME, c.CHECK_CLAUSE FROM INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE u
          inner join INFORMATION_SCHEMA.CHECK_CONSTRAINTS c on u.CONSTRAINT_NAME = c.CONSTRAINT_NAME
      where TABLE_NAME = '{table}' and u.TABLE_SCHEMA = '{schema}' ]]

  return {
    List = "select top 200 * from [{table}]",
    Columns = column_summary_query,
    Indexes = "exec sp_helpindex '{schema}.{table}'",
    ["Foreign Keys"] = foreign_keys_query,
    References = references_query,
    ["Primary Keys"] = primary_keys_query,
    Constraints = constraints_query,
    Describe = "exec sp_help ''{schema}.{table}''",
  }
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__oracle()
  local oracle_from = [[
      FROM all_constraints N
      JOIN all_cons_columns L
      ON N.constraint_name = L.constraint_name
      AND N.owner = L.owner ]]

  local oracle_qualify_and_order_by = [[
      L.table_name = '{table}'
      ORDER BY ]]

  local oracle_key_cmd = function(constraint)
    return [[
      SELECT
      L.table_name,
      L.column_name
      ]] .. oracle_from .. [[
      WHERE
      N.constraint_type = ']] .. constraint .. "'" .. "AND" .. oracle_qualify_and_order_by .. "L.column_name"
  end

  return {
    Columns = [[ select col.column_id,
          col.owner as schema_name,
          col.table_name,
          col.column_name,
          col.data_type,
          col.data_length,
          col.data_precision,
          col.data_scale,
                col.nullable
        from sys.all_tab_columns col
        inner join sys.all_tables t on col.owner = t.owner
                                      and col.table_name = t.table_name
        where col.owner = '{schema}'
        AND col.table_name = '{table}'
        order by col.owner, col.table_name, col.column_id ]],
    ["Foreign Keys"] = oracle_key_cmd("R"),
    Indexes = [[
          SELECT DISTINCT
          N.owner,
          N.index_name,
          N.constraint_type
          ]] .. oracle_from .. [[
          WHERE
          ]] .. oracle_qualify_and_order_by .. "N.index_name",
    List = 'SELECT * FROM "{schema}"."{table}"',
    ["Primary Keys"] = oracle_key_cmd("P"),
    References = [[
            SELECT
            RFRING.owner,
            RFRING.table_name,
            RFRING.column_name
            FROM all_cons_columns RFRING
            JOIN all_constraints N
            ON RFRING.constraint_name = N.constraint_name
            JOIN all_cons_columns RFRD
            ON N.r_constraint_name = RFRD.constraint_name
            JOIN all_users U
            ON N.owner = U.username
            WHERE
            N.constraint_type = 'R'
            AND
            U.common = 'NO'
            AND
            RFRD.owner = '{schema}'
            AND
            RFRD.table_name = '{table}'
            ORDER BY
            RFRING.owner,
            RFRING.table_name,
            RFRING.column_name ]],
  }
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__duck()
  return {
    List = "SELECT * FROM '{table}' LIMIT 500",
    Columns = 'DESCRIBE "{table}"',
    Indexes = "SELECT * FROM duckdb_indexes() WHERE table_name = '{table}'",
    Constraints = "SELECT * FROM duckdb_constraints() WHERE table_name = '{table}'",
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
  elseif type == "bigquery" then
    helpers = self:__bigquery()
  elseif type == "sqlserver" then
    helpers = self:__sqlserver()
  elseif type == "oracle" then
    helpers = self:__oracle()
  elseif type == "duck" then
    helpers = self:__duck()
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
