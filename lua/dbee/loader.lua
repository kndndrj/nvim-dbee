local utils = require("dbee.utils")

local M = {}

-- Parses json file with connections
---@param path? string path to file
---@return connection_details[]
function M.from_file(path)
  path = path or vim.fn.getcwd() .. "/.vim/dbee.json"

  ---@type connection_details[]
  local conns = {}

  if not vim.loop.fs_stat(path) then
    return {}
  end

  local lines = {}
  for line in io.lines(path) do
    if not vim.startswith(vim.trim(line), "//") then
      table.insert(lines, line)
    end
  end

  local contents = table.concat(lines, "\n")
  local ok, data = pcall(vim.fn.json_decode, contents)
  if not ok then
    utils.log("error", 'Could not parse json file: "' .. path .. '".', "loader")
    return {}
  end

  for _, conn in pairs(data) do
    if type(conn) == "table" and conn.url and conn.type then
      table.insert(conns, conn)
    end
  end

  return conns
end

-- Parses env variable if it exists
---@param var? string env var to check - default: DBEE_CONNECTIONS
---@return connection_details[]
function M.from_env(var)
  var = var or "DBEE_CONNECTIONS"

  ---@type connection_details[]
  local conns = {}

  local raw = os.getenv(var)
  if not raw then
    return {}
  end

  local ok, data = pcall(vim.fn.json_decode, raw)
  if not ok then
    utils.log("error", 'Could not parse connections from env: "' .. var .. '".', "loader")
    return {}
  end

  for _, conn in pairs(data) do
    if type(conn) == "table" and conn.url and conn.type then
      table.insert(conns, conn)
    end
  end

  return conns
end

return M
