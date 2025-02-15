<!-- DOCGEN_IGNORE_START -->

<!-- This tag controlls what's ignored by the docgen workflow. -->

![Linting Status](https://img.shields.io/github/actions/workflow/status/kndndrj/nvim-dbee/lint.yml?label=linting&style=for-the-badge)
![Docgen Status](https://img.shields.io/github/actions/workflow/status/kndndrj/nvim-dbee/docgen.yml?label=docgen&logo=neovim&logoColor=white&style=for-the-badge)
![Backend](https://img.shields.io/badge/go-backend-lightblue?style=for-the-badge&logo=go&logoColor=white)
![Frontend](https://img.shields.io/badge/lua-frontend-blue?style=for-the-badge&logo=lua&logoColor=white)

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/yellow_img.png)](https://www.buymeacoffee.com/kndndrj)

<!-- DOCGEN_IGNORE_END -->

# Neovim DBee

**Database Client for NeoVim!**

**Execute Your Favourite Queries From the Comfort of Your Editor!**

**Backend in Go!**

**Frontend in Lua!**

**Doesn't rely on CLI tools!**

**Get Results FAST With Under-the-hood Iterator!**

**Bees Love It!**

***Alpha Software - Expect Breaking Changes!***

<!-- DOCGEN_IGNORE_START -->

![Screenshot](assets/screenshot.jpg)

<!-- DOCGEN_IGNORE_END -->

<!-- DOCGEN_IGNORE_START -->

### Video Introduction

If you prefer to watch a video than to browse through docs, I made a video, which you can watch
[here](https://youtu.be/MDlYsGbKJyQ)

<!-- DOCGEN_IGNORE_END -->

## Installation

**requires nvim>=0.10**

- packer.nvim:

  ```lua
  use {
    "kndndrj/nvim-dbee",
    requires = {
      "MunifTanjim/nui.nvim",
    },
    run = function()
      -- Install tries to automatically detect the install method.
      -- if it fails, try calling it with one of these parameters:
      --    "curl", "wget", "bitsadmin", "go"
      require("dbee").install()
    end,
    config = function()
      require("dbee").setup(--[[optional config]])
    end
  }
  ```

- lazy.nvim:

  ```lua
  {
    "kndndrj/nvim-dbee",
    dependencies = {
      "MunifTanjim/nui.nvim",
    },
    build = function()
      -- Install tries to automatically detect the install method.
      -- if it fails, try calling it with one of these parameters:
      --    "curl", "wget", "bitsadmin", "go"
      require("dbee").install()
    end,
    config = function()
      require("dbee").setup(--[[optional config]])
    end,
  },
  ```

### Platform Support

<!-- DOCGEN_IGNORE_START -->

<details>
  <summary>Click to expand</summary>
<!-- DOCGEN_IGNORE_END -->

This project aims to be as cross-platform as possible, but there are some limitations (for example
some of the go dependencies only work on certain platforms). To address this issue, the client
implementations are detached from the main logic and they register themselves to dbee backend on
plugin start. This allows the use of build constraints, which we use to exclued certain client
implementations on certain platforms.

The CI pipeline tries building the binary for GOARCH/GOOS combinations specified in
[targets.json](ci/targets.json) - if the builds succeed, they are stored in a
[remote bucket](https://github.com/kndndrj/nvim-dbee-bucket) on a separate branch per run.
Additionally, the [install manifest](lua/dbee/install/__manifest.lua) gets created.

To increase cgo cross-platform support, the pipeline uses zig as a C compiler.

To check if your platform is currently supported, check out the mentioned manifest and the targets
file.

<!-- DOCGEN_IGNORE_START -->

</details>
<!-- DOCGEN_IGNORE_END -->

### Manual Binary Installation

<!-- DOCGEN_IGNORE_START -->

<details>
  <summary>Click to expand</summary>
<!-- DOCGEN_IGNORE_END -->

The installation examples include the `build`/`run` functions, which get triggered once the plugin
updates. This should be sufficient for the majority of users. If that doesn't include you, then you
have a few options:

- just install with the `"go"` option (this performs `go build` under the hood):
  ```lua
  require("dbee").install("go")
  ```
- Download an already compiled binary from one of urls in the
  [install manifest](lua/dbee/install/__manifest.lua)
- `go install` (the install location will vary depending on your local go configuration):
  ```sh
  go install github.com/kndndrj/nvim-dbee/dbee@<version>
  ```
- Clone and build
  ```sh
  # Clone the repository and cd into the "go subfolder"
  git clone <this_repo>
  cd <this_repo>/dbee
  # Build the binary (optional output path)
  go build [-o ~/.local/share/nvim/dbee/bin/dbee]
  ```

<!-- DOCGEN_IGNORE_START -->

</details>
<!-- DOCGEN_IGNORE_END -->

## Configuration

You can pass an optional table parameter to `setup()` function.

Here are the defaults:

<!--DOCGEN_CONFIG_START-->

<!-- Contents from lua/dbee/config.lua are inserted between these tags for docgen. -->

[`config.lua`](lua/dbee/config.lua)

<!--DOCGEN_CONFIG_END-->

## Usage

Call the `setup()` function with an optional config parameter.

<!-- DOCGEN_IGNORE_START -->

<details>
  <summary>Brief reference (click to expand):</summary>
<!-- DOCGEN_IGNORE_END -->

```lua
-- Open/close/toggle the UI.
require("dbee").open()
require("dbee").close()
require("dbee").toggle()
-- Run a query on the currently active connection.
require("dbee").execute(query)
-- Store the current result to file/buffer/yank-register (see "Getting Started").
require("dbee").store(format, output, opts)
```

The same functions are also available through the `:Dbee` user command.

<!-- DOCGEN_IGNORE_START -->

</details>
<!-- DOCGEN_IGNORE_END -->

### Getting Started

Here are a few steps to quickly get started:

- call the `setup()` function in your `init.lua`

- Specify connections using one or more sources (reffer to [this section](#specifying-connections)).

- When you restart the editor, call `lua require("dbee").open()` to open the UI.

- Navigate to the drawer (tree) and use the following key-bindings to perform different actions
  depending on the context (the mappings can all be changed in the config):

  - All nodes:

    - Press `o` to toggle the tree node.
    - Press `r` to manually refresh the tree.

  - Connections:

    - Press `cw` to edit the connection
    - Press `dd` to delete it (if source supports saving, it's also removed from there - see more
      below.)
    - Press `<CR>` to perform an action - view history or look at helper queries. Pressing `<CR>`
      directly on the connection node will set it as the active one

  - Scratchpads:

    - Press `<CR>` on the `new` node to create a new scratchpad.
    - When you try to save it to disk (`:w`), the path is automatically filled for you. You can
      change the name to anything you want, if you save it to the suggested directory, it will load
      the next time you open DBee.
    - Press `cw` to rename the scratchpad.
    - Press `dd` to delete it (also from disk).
    - Pressing `<CR>` on an existing scratchpad in the drawer will open it in the editor pane.

  - Help:

    - Just view the key bindings.

- Once you selected the connection and created a scratchpad, you can navigate to the editor pane
  (top-right by default) and start writing queries. In editor pane, you can use the following
  actions:

  - Highlight some text in visual mode and press `BB` - this will run the selected query on the
    active connection.
  - If you press `BB` in normal mode, you run the whole scratchpad on the active connection.

- If the request was successful, the results should appear in the "result" buffer (bottom right by
  default). If the total number of results was lower than the `page_size` parameter in config (100
  by default), all results should already be present. If there are more than `page_size` results,
  you can "page" through them using one of the following:

| Navigation using lua script <br/> (even if your cursor is outside the result buffer) |       Description       | Default key mapping <br/> (cursor should be inside result buffer) |
| ------------------------------------------------------------------------------------ | :---------------------: | :---------------------------------------------------------------: |
| `require("dbee").api.ui.result_page_next()`                                          |     Go to next page     |                                 L                                 |
| `require("dbee").api.ui.result_page_prev()`                                          | Go to the previous page |                                 H                                 |
| `require("dbee").api.ui.result_page_last()`                                          |   Go to the last page   |                                 E                                 |
| `require("dbee").api.ui.result_page_first()`                                         |  Go to the first page   |                                 F                                 |

- Once in the "result" buffer, you can yank the results with the following keys:

  - `yaj` yank current row as json (or row range in visual mode)
  - `yac` yank current row as CSV (or row range in visual mode)
  - `yaJ` to yank all rows as json
  - `yaC` to yank all rows as CSV

- The current result (of the active connection) can also be saved to a file, yank-register or buffer
  using `require("dbee").store()` lua function or `:Dbee store` Ex command. Here are some examples:

  ```lua
  -- All rows as CSV to current buffer:
  require("dbee").store("csv", "buffer", { extra_arg = 0 })
  -- Results from row 2 to row 7 as json to file (index is zero based):
  require("dbee").store("json", "file", { from = 2, to = 7, extra_arg = "path/to/file.json"  })
  -- Yank the first row as table
  require("dbee").store("table", "yank", { from = 0, to = 1 })
  -- Yank the last 2 rows as CSV
  -- (negative indices are interpreted as length+1+index - same as nvim_buf_get_lines())
  -- Be aware that using negative indices requires for the
  -- iterator of the result to be drained completely, which might affect large result sets.
  require("dbee").store("csv", "yank", { from = -3, to = -1 })
  ```

- Once you are done or you want to go back to where you were, you can call
  `require("dbee").close()`.

### Specifying Connections

Connection represents an instance of the database client (i.e. one database). This is how it looks
like:

```lua
{
  id = "optional_identifier" -- only mandatory if you edit a file by hand. IT'S YOUR JOB TO KEEP THESE UNIQUE!
  name = "My Database",
  type = "sqlite", -- type of database driver
  url = "~/path/to/mydb.db",
}
```

The connections are loaded to dbee using so-called "sources". They can be added to dbee using the
`setup()` function:

```lua
  require("dbee").setup {
    sources = {
      require("dbee.sources").MemorySource:new({
        {
          name = "...",
          type = "...",
          url = "...",
        },
        -- ...
      }),
      require("dbee.sources").EnvSource:new("DBEE_CONNECTIONS"),
      require("dbee.sources").FileSource:new(vim.fn.stdpath("cache") .. "/dbee/persistence.json"),
    },
    -- ...
  },
```

The above sources are just built-ins. Here is a short description of them:

- `MemorySource` just loads the connections you give it as an argument.

- `EnvSource` loads connection from an environment variable Just export the variable you gave to the
  loader and you are good to go:

  ```sh
    export DBEE_CONNECTIONS='[
        {
            "name": "DB from env",
            "url": "username:password@tcp(host)/database-name",
            "type": "mysql"
        }
    ]'
  ```

- `FileSource` loads connections from a given json file. It also supports editing and adding
  connections interactively

If the source supports saving and editing you can add connections manually using the "add" item in
the drawer. Fill in the values and write the buffer (`:w`) to save the connection. By default, this
will save the connection to the global connections file and will persist over restarts (because
default `FileSource` supports saving)

Another option is to use "edit" item in the tree and just edit the source manually.

If you aren't satisfied with the default capabilities, you can implement your own source. You just
need to fill the `Source` interface and pass it to config at setup (`:h dbee.sources`).

#### Secrets

If you don't want to have secrets laying around your disk in plain text, you can use the special
placeholders in connection strings (this works using any method for specifying connections).

Each connection parameter is passed through go templating engine, which has two available functions:

- `env` for retrieving environment variables and
- `exec` for evaluating shell commands.

The template syntax for functions is the following: `{{ <func> "<param>" }}`. If you are dealing
with json, you need to escape double quotes, so it's sometimes better to use backticks instead
(`` {{ <func> `<param>` }} ``).

Example:

Using the `DBEE_CONNECTIONS` environment variable for specifying connections and exporting secrets
to environment:

```sh
# Define connections
export DBEE_CONNECTIONS='[
    {
        "name": "{{ exec `echo Hidden Database` }}",
        "url": "postgres://{{ env \"SECRET_DB_USER\" }}:{{ env `SECRET_DB_PASS` }}@localhost:5432/{{ env `SECRET_DB_NAME` }}?sslmode=disable",
        "type": "postgres"
    }
]'

# Export secrets
export SECRET_DB_NAME="secretdb"
export SECRET_DB_USER="secretuser"
export SECRET_DB_PASS="secretpass"
```

If you start neovim in the same shell, this will evaluate to the following connection:

```lua
{ {
  name = "Hidden Database",
  url = "postgres://secretuser:secretpass@localhost:5432/secretdb?sslmode=disable",
  type = "postgres",
} }
```

## API

Dbee comes with it's own API interface. It is split into two parts:

- core (interacting with core of the plugin),
- ui (interacting with ui of the plugin).

You can access it like this:

```lua
require("dbee").api.core.some_func()
require("dbee").api.ui.some_func()
```

## Extensions

- [`nvim-projector`](https://github.com/kndndrj/nvim-projector) To use dbee with projector, use
  [this extension](https://github.com/kndndrj/projector-dbee).

- [`nvim-cmp`](https://github.com/hrsh7th/nvim-cmp) [Here](https://github.com/MattiasMTS/cmp-dbee),
  a completion plugin with dbee support.

<!-- DOCGEN_IGNORE_START -->

## Development

Reffer to [ARCHITECTURE.md](ARCHITECTURE.md) for a brief overview of the architecture.

## Contributing

Contributions are more than welcome!

Do so by creating an `issue` or a `PR`.

You are also welcome to join the [#nvim-dbee:matrix.org](https://matrix.to/#/#nvim-dbee:matrix.org)
chat for more direct communication.

<!-- DOCGEN_IGNORE_END -->
