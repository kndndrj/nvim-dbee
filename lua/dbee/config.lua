local layout = require("dbee.utils").layout

local M = {}
local m = {}

---@alias mapping {key: string, mode: string}

---@class UiConfig
---@field window_open_order table example: { "result", "editor", "drawer" } - in which order are the windows open
---@field pre_open_hook fun() execute this before opening ui
---@field post_open_hook fun() execute this after opening ui
---@field pre_close_hook fun() execute this before closing ui
---@field post_close_hook fun() execute this after closing ui

-- configuration object
---@class Config
---@field connections connection_details[] list of configured database connections
---@field extra_helpers table<string, table_helpers> extra table helpers to provide besides built-ins. example: { postgres = { List = "select..." }
---@field lazy boolean lazy load the plugin or not?
---@field drawer drawer_config
---@field editor editor_config
---@field result handler_config
---@field ui UiConfig

-- default configuration
---@type Config
-- DOCGEN_START
M.default = {
  -- lazy load the plugin or not?
  lazy = false,

  -- list of connections
  -- don't commit that, use something like nvim-projector for project specific config.
  connections = {
    -- example:
    -- {
    --   name = "example-pg",
    --   type = "postgres",
    --   url = "postgres://user:password@localhost:5432/db?sslmode=disable",
    -- },
  },
  -- extra table helpers per connection type
  extra_helpers = {
    -- example:
    -- ["postgres"] = {
    --   ["List All"] = "select * from {table}",
    -- },
  },

  -- drawer window config
  drawer = {
    -- command that opens the window if the window is closed
    -- string or function
    window_command = "to 40vsplit",
    -- mappings for the buffer
    mappings = {
      -- manually refresh drawer
      refresh = { key = "r", mode = "n" },
      -- actions perform different stuff depending on the node:
      -- action_1 opens a scratchpad or executes a helper
      action_1 = { key = "<CR>", mode = "n" },
      -- action_2 renames a scratchpad or sets the connection as active manually
      action_2 = { key = "da", mode = "n" },
      -- action_3 deletes a scratchpad
      action_3 = { key = "dd", mode = "n" },
      -- these are self-explanatory:
      collapse = { key = "c", mode = "n" },
      expand = { key = "e", mode = "n" },
      toggle = { key = "o", mode = "n" },
    },
    -- icon settings:
    disable_icons = false,
    icons = {
      -- these are what's available for now:
      history = {
        icon = "",
        highlight = "Constant",
      },
      scratch = {
        icon = "",
        highlight = "Character",
      },
      database = {
        icon = "",
        highlight = "SpecialChar",
      },
      table = {
        icon = "",
        highlight = "Conditional",
      },
      add = {
        icon = "",
        highlight = "String",
      },
      remove = {
        icon = "󰆴",
        highlight = "SpellBad",
      },
      help = {
        icon = "󰋖",
        highlight = "NormalFloat",
      },

      -- if there is no type
      -- use this for normal nodes...
      none = {
        icon = " ",
      },
      -- ...and use this for nodes with children
      none_dir = {
        icon = "",
        highlight = "NonText",
      },

      -- chevron icons for expanded/closed nodes
      node_expanded = {
        icon = "",
        highlight = "NonText",
      },
      node_closed = {
        icon = "",
        highlight = "NonText",
      },
    },
  },

  -- results window config
  result = {
    -- command that opens the window if the window is closed
    -- string or function
    window_command = "bo 15split",
    -- number of rows per page
    page_size = 100,
    -- mappings for the buffer
    mappings = {
      -- next/previous page
      page_next = { key = "L", mode = "" },
      page_prev = { key = "H", mode = "" },
    },
  },

  -- editor window config
  editor = {
    -- command that opens the window if the window is closed
    -- string or function
    window_command = function()
      vim.cmd("new")
      vim.cmd("only")
      m.tmp_buf = vim.api.nvim_get_current_buf()
      return vim.api.nvim_get_current_win()
    end,
    -- mappings for the buffer
    mappings = {
      -- run what's currently selected on the active connection
      run_selection = { key = "BB", mode = "v" },
      -- run the whole file on the active connection
      run_file = { key = "BB", mode = "n" },
    },
  },

  -- general UI config
  -- Default configuration uses a "layout" helper to save the existing ui before opening any windows,
  -- then makes a new empty window for the editor and then opens result and drawer.
  -- When later calling dbee.close(), the previously saved layout is restored.
  -- NOTE: "m" is just a global object - nothing special about it - you might as well just use global vars.
  --
  -- You can probably do anything you imagine with this - for example all floating windows, tiled/floating mix etc.
  ui = {
    -- how to open windows in order (with specified "window_command"s -- see above)
    window_open_order = { "editor", "result", "drawer" },

    -- hooks before/after dbee.open()/.close()
    pre_open_hook = function()
      -- save layout before opening ui
      m.egg = layout.save()
    end,
    post_open_hook = function()
      -- delete temporary editor buffer
      vim.cmd("bd " .. m.tmp_buf)
    end,
    pre_close_hook = function() end,
    post_close_hook = function()
      layout.restore(m.egg)
      m.egg = nil
    end,
  },
}
-- DOCGEN_END

return M
