local Drawer = require("db.ui.drawer")
local Editor = require("db.ui.editor")
local Result = require("db.ui.result")

local UI = {}

function UI:new()
  local o = {
    drawer = Drawer:new { connections = require("db").connections },
    editor = Editor:new(),
    result = Result:new(),
  }
  setmetatable(o, self)
  self.__index = self
  return o
end

function UI:open()
  self.drawer:show()
end

function UI:close()
  self.drawer:hide()
end

return UI
