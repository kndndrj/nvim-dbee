if vim.g.loaded_dbee == 1 then
  return
end
vim.g.loaded_dbee = 1

local COMMAND_NAME = "Dbee"

local commands = {
  open = require("dbee").open,
  close = require("dbee").close,
  toggle = require("dbee").toggle,
  execute = function(args)
    require("dbee").execute(table.concat(args, " "))
  end,
  store = function(args)
    -- args are "format", "output" and "extra_arg"
    if #args < 3 then
      error("not enough arguments, got " .. #args .. " want 3")
    end

    require("dbee").store(args[1], args[2], { extra_arg = args[3] })
  end,
}

---@param args string args in form of Dbee arg1 arg2 ...
---@return string[]
local function split_args(args)
  local stripped = args:gsub(COMMAND_NAME, "")

  local ret = {}
  for word in string.gmatch(stripped, "([^ |\t]+)") do
    table.insert(ret, word)
  end

  return ret
end

---@param input integer[]
---@return string[]
local function tostringlist(input)
  local ret = {}
  for _, elem in ipairs(input) do
    table.insert(ret, tostring(elem))
  end
  return ret
end

-- Create user command for dbee
vim.api.nvim_create_user_command(COMMAND_NAME, function(opts)
  local args = split_args(opts.args)
  if #args < 1 then
    -- default is toggle
    require("dbee").toggle()
    return
  end

  local cmd = args[1]
  table.remove(args, 1)

  local fn = commands[cmd]
  if fn then
    fn(args)
    return
  end

  error("unsupported subcommand: " .. (cmd or ""))
end, {
  nargs = "*",
  complete = function(_, cmdline, _)
    local line = split_args(cmdline)
    if #line < 1 then
      return vim.tbl_keys(commands)
    end

    if line[1] ~= "store" then
      return {}
    end

    local nargs = #line
    if nargs == 1 then
      -- format
      return { "csv", "json", "table" }
    elseif nargs == 2 then
      -- output
      return { "file", "yank", "buffer" }
    elseif nargs == 3 then
      -- extra_arg
      if line[3] == "buffer" then
        return tostringlist(vim.api.nvim_list_bufs())
      end

      return
    end

    return vim.tbl_keys(commands)
  end,
})
