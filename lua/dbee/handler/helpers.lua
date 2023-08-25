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
    List = 'SELECT * FROM "{table}" LIMIT 500',
    Columns = "SELECT * FROM information_schema.columns WHERE table_name='{table}' AND table_schema='{schema}'",
    Indexes = "SELECT * FROM pg_indexes WHERE tablename='{table}' AND schemaname='{schema}'",
    ["Foreign Keys"] = basic_constraint_query
      .. "WHERE constraint_type = 'FOREIGN KEY' AND tc.table_name = '{table}' AND tc.table_schema = '{schema}'",
    References = basic_constraint_query
      .. "WHERE constraint_type = 'FOREIGN KEY' AND ccu.table_name = '{table}' AND tc.table_schema = '{schema}'",
    ["Primary Keys"] = basic_constraint_query
      .. "WHERE constraint_type = 'PRIMARY KEY' AND tc.table_name = '{table}' AND tc.table_schema = '{schema}'",
  }
end

---@private
---@return table_helpers helpers list of table helpers
function Helpers:__mysql()
  return {
    List = "SELECT * FROM `{table}` LIMIT 500",
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
    List = 'SELECT * FROM "{table}" LIMIT 500',
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
      SELECT c.column_name + ' (' +
          ISNULL(( SELECT 'PK, ' FROM information_schema.table_constraints AS k JOIN information_schema.key_column_usage AS kcu ON k.constraint_name = kcu.constraint_name WHERE constraint_type='PRIMARY KEY' AND k.table_name = c.table_name AND kcu.column_name = c.column_name), '') +
          ISNULL(( SELECT 'FK, ' FROM information_schema.table_constraints AS k JOIN information_schema.key_column_usage AS kcu ON k.constraint_name = kcu.constraint_name WHERE constraint_type='FOREIGN KEY' AND k.table_name = c.table_name AND kcu.column_name = c.column_name), '') +
          data_type + COALESCE('(' + RTRIM(CAST(character_maximum_length AS VARCHAR)) + ')','(' + RTRIM(CAST(numeric_precision AS VARCHAR)) + ',' + RTRIM(CAST(numeric_scale AS VARCHAR)) + ')','(' + RTRIM(CAST(datetime_precision AS VARCHAR)) + ')','') + ', ' +
          CASE WHEN is_nullable = 'YES' THEN 'null' ELSE 'not null' END + ')' AS Columns
      FROM information_schema.columns c WHERE c.table_name='{table}' AND c.TABLE_SCHEMA = '{schema}' ]]

  local foreign_keys_query = [[
      SELECT c.constraint_name
         , kcu.column_name AS column_name
         , c2.table_name AS foreign_table_name
         , kcu2.column_name AS foreign_column_name
      FROM information_schema.table_constraints c
            INNER JOIN information_schema.key_column_usage kcu
              ON c.constraint_schema = kcu.constraint_schema
                AND c.constraint_name = kcu.constraint_name
            INNER JOIN information_schema.referential_constraints rc
              ON c.constraint_schema = rc.constraint_schema
                AND c.constraint_name = rc.constraint_name
            INNER JOIN information_schema.table_constraints c2
              ON rc.unique_constraint_schema = c2.constraint_schema
                AND rc.unique_constraint_name = c2.constraint_name
            INNER JOIN information_schema.key_column_usage kcu2
              ON c2.constraint_schema = kcu2.constraint_schema
                AND c2.constraint_name = kcu2.constraint_name
                AND kcu.ordinal_position = kcu2.ordinal_position
      WHERE c.constraint_type = 'FOREIGN KEY'
      AND c.TABLE_NAME = '{table}' AND c.TABLE_SCHEMA = '{schema}' ]]

  local references_query = [[
      SELECT kcu1.constraint_name AS constraint_name
          , kcu1.table_name AS foreign_table_name
          , kcu1.column_name AS foreign_column_name
          , kcu2.column_name AS column_name
      FROM information_schema.referential_constraints AS rc
      INNER JOIN information_schema.key_column_usage AS kcu1
          ON kcu1.constraint_catalog = rc.constraint_catalog
          AND kcu1.constraint_schema = rc.constraint_schema
          AND kcu1.constraint_name = rc.constraint_name
      INNER JOIN information_schema.key_column_usage AS kcu2
          ON kcu2.constraint_catalog = rc.unique_constraint_catalog
          AND kcu2.constraint_schema = rc.unique_constraint_schema
          AND kcu2.constraint_name = rc.unique_constraint_name
          AND kcu2.ordinal_position = kcu1.ordinal_position
      WHERE kcu2.table_name='{table}' AND kcu2.table_schema = '{schema}' ]]

  local primary_keys_query = [[
       SELECT tc.constraint_name, kcu.column_name
       FROM
           information_schema.table_constraints AS tc
           JOIN information_schema.key_column_usage AS kcu
             ON tc.constraint_name = kcu.constraint_name
           JOIN information_schema.constraint_column_usage AS ccu
             ON ccu.constraint_name = tc.constraint_name
      WHERE constraint_type = 'PRIMARY KEY'
      AND tc.table_name = '{table}' AND tc.table_schema = '{schema}' ]]

  local constraints_query = [[
      SELECT u.CONSTRAINT_NAME, c.CHECK_CLAUSE FROM INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE u
          INNER JOIN INFORMATION_SCHEMA.CHECK_CONSTRAINTS c ON u.CONSTRAINT_NAME = c.CONSTRAINT_NAME
      WHERE TABLE_NAME = '{table}' AND u.TABLE_SCHEMA = '{schema}' ]]

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
    Columns = [[ SELECT col.column_id,
          col.owner AS schema_name,
          col.table_name,
          col.column_name,
          col.data_type,
          col.data_length,
          col.data_precision,
          col.data_scale,
          col.nullable
        FROM sys.all_tab_columns col
        INNER JOIN sys.all_tables t
          ON col.owner = t.owner
          AND col.table_name = t.table_name
        WHERE col.owner = '{schema}'
        AND col.table_name = '{table}'
        ORDER BY col.owner, col.table_name, col.column_id ]],
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
---
---@private
---@param materialization string
---@return table_helpers helpers list of table helpers
function Helpers:__redshift(materialization)
  local default_list_query = 'SELECT * FROM "{schema}"."{table}" LIMIT 500;'
  if materialization == "table" then
    return {
      List = default_list_query,
      Columns = "SELECT * FROM information_schema.columns WHERE table_name='{table}' AND table_schema='{schema}';",
      Indexes = "SELECT * FROM pg_indexes WHERE tablename='{table}' AND schemaname='{schema}';",
      ["Foreign Keys"] = [[
      SELECT tc.constraint_name, tc.table_name, kcu.column_name, ccu.table_name AS foreign_table_name, ccu.column_name AS foreign_column_name, rc.update_rule, rc.delete_rule
      FROM
           information_schema.table_constraints AS tc
           JOIN information_schema.key_column_usage AS kcu
             ON tc.constraint_name = kcu.constraint_name
           JOIN information_schema.referential_constraints as rc
             ON tc.constraint_name = rc.constraint_name
           JOIN information_schema.constraint_column_usage AS ccu
             ON ccu.constraint_name = tc.constraint_name
      WHERE constraint_type = 'FOREIGN KEY' AND tc.table_name = '{table}' AND tc.table_schema = '{schema}';
      ]],
      ["Table Definition"] = [[
    SELECT
        *
    FROM svv_table_info
    WHERE "schema" = '{schema}'
      AND "table" = '{table}';
    ]],
    }
  elseif materialization == "view" then
    return {
      List = default_list_query,
      ["View Definition"] = [[
    SELECT
        *
    FROM pg_views
    WHERE schemaname = '{schema}'
      AND viewname = '{table}';
    ]],
    }
  end
  return {}
end

---@param type string
---@param vars { table: string, schema: string, dbname: string , materialization: string}
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
  elseif type == "redshift" then
    helpers = self:__redshift(vars.materialization)
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

---@param type string
---@param vars { table: string, schema: string, dbname: string , materialization: string}
---@return string[] # list of available table helpers for given type
function Helpers:list(type, vars)
  local helpers = self:get(type, vars)

  return utils.sorted_keys(helpers)
end

return Helpers
