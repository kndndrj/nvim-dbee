local config = {}

---@mod dbee.ref.config Dbee Configuration

-- Configuration object.
---@class Config
---@field default_connection? string
---@field sources? Source[] list of connection sources
---@field extra_helpers? table<string, table<string, string>>
---@field float_options? table<string, any>
---@field drawer? drawer_config
---@field editor? editor_config
---@field result? result_config
---@field call_log? call_log_config
---@field window_layout? Layout

---@class Candy
---@field icon string
---@field icon_highlight string
---@field text_highlight string

---Keymap options.
---@alias key_mapping { key: string, mode: string, opts: table, action: string|fun() }

---@divider -

---Configuration for result UI tile.
---@alias result_config { focus_result: boolean, mappings: key_mapping[], page_size: integer, progress: progress_config, window_options: table<string, any>, buffer_options: table<string, any> }

---Configuration for editor UI tile.
---@alias editor_config { directory: string, mappings: key_mapping[], window_options: table<string, any>, buffer_options: table<string, any> }

---Configuration for call log UI tile.
---@alias call_log_config { mappings: key_mapping[], disable_candies: boolean, candies: table<string, Candy>, window_options: table<string, any>, buffer_options: table<string, any> }

---Configuration for drawer UI tile.
---@alias drawer_config { disable_candies: boolean, candies: table<string, Candy>, mappings: key_mapping[], disable_help: boolean, window_options: table<string, any>, buffer_options: table<string, any> }

---@divider -

-- DOCGEN_START
---Default configuration.
---To see defaults, run :lua= require"dbee.config".default
---@type Config config
config.default = {
  -- you can specify an optional default connection id and it will be the active one
  -- when dbee starts
  default_connection = nil,
  -- loads connections from files and environment variables
  sources = {
    require("dbee.sources").EnvSource:new("DBEE_CONNECTIONS"),
    require("dbee.sources").FileSource:new(vim.fn.stdpath("state") .. "/dbee/persistence.json"),
  },
  -- extra table helpers per connection type
  -- every helper value is a go-template with values set for
  -- "Table", "Schema" and "Materialization"
  extra_helpers = {
    -- example:
    -- ["postgres"] = {
    --   ["List All"] = "select * from {{ .Table }}",
    -- },
  },
  -- options passed to floating windows - :h nvim_open_win()
  float_options = {},

  -- drawer window config
  drawer = {
    -- these two option settings can be added to all UI elements and
    -- allow for passing specific window/buffer options.
    -- Note that you probably shouldn't be passing buffer options, since
    -- the functionality of the plugin might rely on them.
    -- TL;DR: only use this if you know what you are doing!
    window_options = {},
    buffer_options = {},

    -- show help or not
    disable_help = false,
    -- mappings for the buffer
    mappings = {
      -- manually refresh drawer
      { key = "r", mode = "n", action = "refresh" },
      -- actions perform different stuff depending on the node:
      -- action_1 opens a note or executes a helper
      { key = "<CR>", mode = "n", action = "action_1" },
      -- action_2 renames a note or sets the connection as active manually
      { key = "cw", mode = "n", action = "action_2" },
      -- action_3 deletes a note or connection (removes connection from the file if you configured it like so)
      { key = "dd", mode = "n", action = "action_3" },
      -- these are self-explanatory:
      -- { key = "c", mode = "n", action = "collapse" },
      -- { key = "e", mode = "n", action = "expand" },
      { key = "o", mode = "n", action = "toggle" },
      -- mappings for menu popups:
      { key = "<CR>", mode = "n", action = "menu_confirm" },
      { key = "y", mode = "n", action = "menu_yank" },
      { key = "<Esc>", mode = "n", action = "menu_close" },
      { key = "q", mode = "n", action = "menu_close" },
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
        text_highlight = "",
      },
      schema = {
        icon = "",
        icon_highlight = "Removed",
        text_highlight = "",
      },
      table = {
        icon = "",
        icon_highlight = "Conditional",
        text_highlight = "",
      },
      streaming_table = {
        icon = "",
        icon_highlight = "Conditional",
        text_highlight = "",
      },
      managed = {
        icon = "",
        icon_highlight = "Conditional",
        text_highlight = "",
      },
      view = {
        icon = "",
        icon_highlight = "Debug",
        text_highlight = "",
      },
      materialized_view = {
        icon = "",
        icon_highlight = "Type",
        text_highlight = "",
      },
      sink = {
        icon = "",
        icon_highlight = "String",
        text_highlight = "",
      },
      column = {
        icon = "󰠵",
        icon_highlight = "WarningMsg",
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
        icon_highlight = "",
        text_highlight = "",
      },
      -- ...and use this for nodes with children
      none_dir = {
        icon = "",
        icon_highlight = "NonText",
        text_highlight = "",
      },

      -- chevron icons for expanded/closed nodes
      node_expanded = {
        icon = "",
        icon_highlight = "NonText",
        text_highlight = "",
      },
      node_closed = {
        icon = "",
        icon_highlight = "NonText",
        text_highlight = "",
      },
    },
  },

  -- results window config
  result = {
    -- see drawer comment.
    window_options = {},
    buffer_options = {},

    -- number of rows in the results set to display per page
    page_size = 100,

    -- whether to focus the result window after a query
    focus_result = true,

    -- progress (loading) screen options
    progress = {
      -- spinner to use in progress display
      spinner = { "⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏" },
      -- prefix to display before the timer
      text_prefix = "Executing...",
    },

    -- mappings for the buffer
    mappings = {
      -- next/previous page
      { key = "L", mode = "", action = "page_next" },
      { key = "H", mode = "", action = "page_prev" },
      { key = "E", mode = "", action = "page_last" },
      { key = "F", mode = "", action = "page_first" },
      -- yank rows as csv/json
      { key = "yaj", mode = "n", action = "yank_current_json" },
      { key = "yaj", mode = "v", action = "yank_selection_json" },
      { key = "yaJ", mode = "", action = "yank_all_json" },
      { key = "yac", mode = "n", action = "yank_current_csv" },
      { key = "yac", mode = "v", action = "yank_selection_csv" },
      { key = "yaC", mode = "", action = "yank_all_csv" },

      -- cancel current call execution
      { key = "<C-c>", mode = "", action = "cancel_call" },
    },
  },

  -- editor window config
  editor = {
    -- see drawer comment.
    window_options = {},
    buffer_options = {},

    -- directory where to store the scratchpads.
    --directory = "path/to/scratchpad/dir",

    -- mappings for the buffer
    mappings = {
      -- run what's currently selected on the active connection
      { key = "BB", mode = "v", action = "run_selection" },
      -- run the whole file on the active connection
      { key = "BB", mode = "n", action = "run_file" },
      -- run what's under the cursor to the next newline
      { key = "<CR>", mode = "n", action = "run_under_cursor" },
    },
  },

  -- call log window config
  call_log = {
    -- see drawer comment.
    window_options = {},
    buffer_options = {},

    -- mappings for the buffer
    mappings = {
      -- show the result of the currently selected call record
      { key = "<CR>", mode = "", action = "show_result" },
      -- cancel the currently selected call (if its still executing)
      { key = "<C-c>", mode = "", action = "cancel_call" },
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
---@package
---@param cfg Config
function config.validate(cfg)
  vim.validate {
    sources = { cfg.sources, "table" },
    extra_helpers = { cfg.extra_helpers, "table" },
    float_options = { cfg.float_options, "table" },

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
    window_layout_is_open = { cfg.window_layout.is_open, "function" },
    window_layout_close = { cfg.window_layout.close, "function" },
  }
end

-- Merges changes from config on top of default config.
-- Keys settings provided via arguments take precedence.
---@package
---@param changes? Config
---@return Config
function config.merge_with_default(changes)
  changes = changes or {}

  -- merge basic settings
  ---@type Config
  local merged = vim.tbl_deep_extend("force", config.default, changes)

  -- classes need some extra love (frustrating, I know)
  if changes.sources and #changes.sources > 0 then
    merged.sources = changes.sources
  end
  if changes.window_layout then
    merged.window_layout = changes.window_layout
  end

  return merged
end

return config
