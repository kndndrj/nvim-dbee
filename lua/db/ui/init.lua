local Drawer = require("db.ui.drawer")
local Editor = require("db.ui.editor")
local Result = require("db.ui.result")

local UI = {}

function UI:new()

  ---@type Result
  local result = Result:new()
  local editor = Editor:new()

  -- class object
  local o = {
    drawer = nil,
    editor = editor,
    result = result,
  }

  o.drawer = Drawer:new {
    connections = require("db").connections,
    on_result = function(lines, type)
      result:set(lines, type)
      o:open_result()
    end,
  }

  setmetatable(o, self)
  self.__index = self
  return o
end

function UI:open_drawer()
  vim.cmd("to 40vsplit")
  self.drawer:render(0)
end

function UI:open_result()
  vim.cmd("bo 15split")
  self.result:render(0)
end

function UI:close()
  self.drawer:hide()
end

return UI
