local utils = require("db.utils")
-- module to format results for different use cases
local M = {}

---@param input grid
---@param opts? { vertical_separator: string, horizontal_separator: string }
---@return string[]
function M.display(input, opts)
  local headers = input.header
  local results = input.rows

  opts = opts or {}
  local vsep = opts.vertical_separator or " │ "
  local hsep = opts.horizontal_separator or "─"

  -- get max character lengths
  local max_lenghts = {}
  for i, h in ipairs(headers) do
    local max = utils.longest(results, i)
    h = h or ""
    if max < string.len(h) then
      max_lenghts[h] = string.len(h)
    else
      max_lenghts[h] = max
    end
  end

  -- headers to string
  local head
  for _, h in ipairs(headers) do
    h = h or ""
    if head then
      head = head .. vsep .. h .. string.rep(" ", max_lenghts[h] - string.len(h))
    else
      head = h .. string.rep(" ", max_lenghts[h] - string.len(h))
    end
  end
  -- add header with border (horizontal line)
  ---@type string[]
  local ret = { head, string.rep(hsep, string.len(head)) }

  -- rows to strings
  for _, v in ipairs(results) do
    local value_row
    for i, h in ipairs(headers) do
      h = h or ""
      local val = v[i] or ""
      if value_row then
        value_row = value_row .. vsep .. val .. string.rep(" ", max_lenghts[h] - string.len(val))
      else
        value_row = val .. string.rep(" ", max_lenghts[h] - string.len(val))
      end
    end
    table.insert(ret, value_row)
  end

  return ret
end

return M
