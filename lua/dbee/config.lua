local spinners = require("dbee.progress").spinners

local M = {}

---@alias mapping { key: string, mode: string, opts: table }|{ key: string, mode: string, opts: table }[]
---@alias keymap { action: fun(), mapping: mapping }

-- configuration object
---@class Config
---@field sources Source[] list of connection sources
---@field extra_helpers table<string, table_helpers> extra table helpers to provide besides built-ins. example: { postgres = { List = "select..." }
---@field page_size integer
---@field progress_bar progress_config
---@field drawer drawer_config
---@field editor editor_config
---@field result result_config
---@field call_log call_log_config
---@field window_layout WindowLayout

-- default configuration
---@type Config
-- DOCGEN_START
M.default = {
  -- loads connections from files and environment variables
  sources = {
    require("dbee.sources").EnvSource:new("DBEE_CONNECTIONS"),
    require("dbee.sources").FileSource:new(vim.fn.stdpath("cache") .. "/dbee/persistence.json"),
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
    -- show help or not
    disable_help = false,
    -- mappings for the buffer
    mappings = {
      -- quit the dbee interface
      quit = { key = "q", mode = "n" },
      -- manually refresh drawer
      refresh = { key = "r", mode = "n" },
      -- actions perform different stuff depending on the node:
      -- action_1 opens a note or executes a helper
      action_1 = { key = "<CR>", mode = "n" },
      -- action_2 renames a note or sets the connection as active manually
      action_2 = { key = "cw", mode = "n" },
      -- action_3 deletes a note or connection (removes connection from the file if you configured it like so)
      action_3 = { key = "dd", mode = "n" },
      -- these are self-explanatory:
      -- collapse = { key = "c", mode = "n" },
      -- expand = { key = "e", mode = "n" },
      toggle = { key = "o", mode = "n" },
    },
    -- icon settings:
    disable_candies = false,
    candies = {
      -- these are what's available for now:
      history = {
        icon = "",
        icon_highlight = "Constant",
        text_highlight = "",
      },
      note = {
        icon = "",
        icon_highlight = "Character",
        text_highlight = "",
      },
      connection = {
        icon = "󱘖",
        icon_highlight = "SpecialChar",
        text_highlight = "",
      },
      database_switch = {
        icon = "",
        icon_highlight = "Character",
      },
      table = {
        icon = "",
        icon_highlight = "Conditional",
        text_highlight = "",
      },
      view = {
        icon = "",
        icon_highlight = "Debug",
        text_highlight = "",
      },
      add = {
        icon = "",
        icon_highlight = "String",
        text_highlight = "String",
      },
      edit = {
        icon = "󰏫",
        icon_highlight = "Directory",
        text_highlight = "Directory",
      },
      remove = {
        icon = "󰆴",
        icon_highlight = "SpellBad",
        text_highlight = "SpellBad",
      },
      help = {
        icon = "󰋖",
        icon_highlight = "Title",
        text_highlight = "Title",
      },
      source = {
        icon = "󰃖",
        icon_highlight = "MoreMsg",
        text_highlight = "MoreMsg",
      },

      -- if there is no type
      -- use this for normal nodes...
      none = {
        icon = " ",
      },
      -- ...and use this for nodes with children
      none_dir = {
        icon = "",
        icon_highlight = "NonText",
      },

      -- chevron icons for expanded/closed nodes
      node_expanded = {
        icon = "",
        icon_highlight = "NonText",
      },
      node_closed = {
        icon = "",
        icon_highlight = "NonText",
      },
    },
  },

  -- results window config
  result = {
    -- number of rows in the results set to display per page
    page_size = 100,

    -- progress (loading) screen options
    progress = {
      -- spinner to use, see lua/dbee/spinners.lua
      spinner = spinners.dots,
      -- prefix to display before the timer
      text_prefix = "Executing...",
    },

    -- mappings for the buffer
    mappings = {
      -- next/previous page
      page_next = { key = "L", mode = "" },
      page_prev = { key = "H", mode = "" },
      -- yank rows as csv/json
      yank_current_json = { key = "yaj", mode = "n" },
      yank_selection_json = { key = "yaj", mode = "v" },
      yank_all_json = { key = "yaJ", mode = "" },
      yank_current_csv = { key = "yac", mode = "n" },
      yank_selection_csv = { key = "yac", mode = "v" },
      yank_all_csv = { key = "yaC", mode = "" },
    },
  },

  -- editor window config
  editor = {
    -- mappings for the buffer
    mappings = {
      -- run what's currently selected on the active connection
      run_selection = { key = "BB", mode = "v" },
      -- run the whole file on the active connection
      run_file = { key = "BB", mode = "n" },
    },
  },

  -- call log window config
  call_log = {
    -- mappings for the buffer
    mappings = {
      -- show the result of the currently selected call record
      show_result = { key = "<CR>", mode = "" },
      -- cancel the currently selected call (if its still executing)
      cancel = { key = "d", mode = "" },
    },

    -- candies (icons and highlights)
    disable_candies = false,
    candies = {
      -- all of these represent call states
      unknown = {
        icon = "", -- this or first letters of state
        icon_highlight = "NonText", -- highlight of the state
        text_highlight = "", -- highlight of the rest of the line
      },
      executing = {
        icon = "󰑐",
        icon_highlight = "Constant",
        text_highlight = "Constant",
      },
      executing_failed = {
        icon = "󰑐",
        icon_highlight = "Error",
        text_highlight = "",
      },
      retrieving = {
        icon = "",
        icon_highlight = "String",
        text_highlight = "String",
      },
      retrieving_failed = {
        icon = "",
        icon_highlight = "Error",
        text_highlight = "",
      },
      archived = {
        icon = "",
        icon_highlight = "Title",
        text_highlight = "",
      },
      archive_failed = {
        icon = "",
        icon_highlight = "Error",
        text_highlight = "",
      },
      canceled = {
        icon = "",
        icon_highlight = "Error",
        text_highlight = "",
      },
    },
  },

  -- window layout
  window_layout = require("dbee.layouts").Default:new(),
}
-- DOCGEN_END

-- Validates provided input config
---@param cfg Config
function M.validate(cfg)
  vim.validate {
    sources = { cfg.sources, "table" },
    extra_helpers = { cfg.extra_helpers, "table" },

    drawer_disable_candies = { cfg.drawer.disable_candies, "boolean" },
    drawer_disable_help = { cfg.drawer.disable_help, "boolean" },
    drawer_candies = { cfg.drawer.candies, "table" },
    drawer_mappings = { cfg.drawer.mappings, "table" },
    result_page_size = { cfg.result.page_size, "number" },
    result_progress = { cfg.result.progress, "table" },
    result_mappings = { cfg.result.mappings, "table" },
    editor_mappings = { cfg.editor.mappings, "table" },
    call_log_mappings = { cfg.call_log.mappings, "table" },

    window_layout = { cfg.window_layout, "table" },
    window_layout_open = { cfg.window_layout.open, "function" },
    window_layout_close = { cfg.window_layout.close, "function" },
  }
end

return M
