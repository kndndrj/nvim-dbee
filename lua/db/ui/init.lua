local Drawer = require("db.ui.drawer")
local Editor = require("db.ui.editor")
local Result = require("db.ui.result")

local UI = {}

function UI:new()
  ---@type Result
  local result = Result:new()
  local editor = Editor:new()

  local drawer = Drawer:new {
    connections = require("db").connections,
    on_result = function(lines, type)
      vim.pretty_print(lines)
      result:set(lines, type)
      result:show()
    end,
  }

  local o = {
    drawer = drawer,
    editor = editor,
    result = result,
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

function UI:open()
  vim.cmd("to 40vsplit")
  self.drawer:render(0)
  -- self.result:show()
end

function UI:close()
  self.drawer:hide()
end

return UI
