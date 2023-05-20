---@alias result_config { mappings: table<string, mapping>, page_size: integer }

-- Result is a wrapper around the go code
-- it is the central part of the plugin and manages connections.
-- almost all functions take the connection id as their argument.
---@class Result
---@field private ui Ui
---@field private handler Handler
---@field private mappings table<string, mapping>
---@field private size integer number of rows per page
local Result = {}

---@param ui Ui
---@param handler Handler
---@param opts? result_config
---@return Result
function Result:new(ui, handler, opts)
  opts = opts or {}

  if not handler then
    error("no Handler passed to Result")
  end
  if not ui then
    error("no Ui passed to Result")
  end

  local page_size = opts.page_size or 100

  -- class object
  local o = {
    ui = ui,
    handler = handler,
    size = page_size,
    mappings = opts.mappings or {},
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

---@return integer # size of one page
function Result:page_size()
  return self.size
end

---@return table<string, fun()>
function Result:actions()
  return {
    page_next = function()
      self.handler:current_connection():page_next()
    end,
    page_prev = function()
      self.handler:current_connection():page_prev()
    end,
  }
end

---@private
function Result:map_keys(bufnr)
  local map_options = { noremap = true, nowait = true, buffer = bufnr }

  local actions = self:actions()

  for act, map in pairs(self.mappings) do
    local action = actions[act]
    if action and type(action) == "function" then
      vim.keymap.set(map.mode, map.key, action, map_options)
    end
  end
end

function Result:open()
  local _, bufnr = self.ui:open()

  -- set keymaps
  self:map_keys(bufnr)
end

function Result:close()
  self.ui:close()
end

return Result
