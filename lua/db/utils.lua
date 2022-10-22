local M = {}

---@param array string[]
function M.alphanumsort(array)
  local function padnum(d)
    return ("%03d%s"):format(#d, d)
  end

  table.sort(array, function(a, b)
    return tostring(a):gsub("%d+", padnum) < tostring(b):gsub("%d+", padnum)
  end)
  return array
end

---@param obj table targeted table
---@param fields { exact?: string[], prefixes?: string[] } exact field names or prefixes
function M.is_in_table(obj, fields)
  for _, f in pairs(fields) do
    if obj[f] == nil then
      return false
    end
  end
  return true
end

---@param obj table
---@param selector string|integer
function M.longest(obj, selector)
  local len = 0
  for _, item in pairs(obj) do
    local i = item[selector] or ""
    local item_len = string.len(i)
    if item_len > len then
      len = item_len
    end
  end
  return len
end

---@param level "info"|"warn"|"error"
---@param message string
---@param subtitle? string
function M.log(level, message, subtitle)
  -- log level
  local l = vim.log.levels.OFF
  if level == "info" then
    l = vim.log.levels.INFO
  elseif level == "warn" then
    l = vim.log.levels.WARN
  elseif level == "error" then
    l = vim.log.levels.ERROR
  end

  -- subtitle
  if subtitle then
    subtitle = "[" .. subtitle .. "]:"
  else
    subtitle = ""
  end
  vim.notify(subtitle .. " " .. message, l, { title = "nvim-db" })
end

return M
