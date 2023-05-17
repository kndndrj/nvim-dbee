<!-- Any html tags, badges etc. go before this tag. -->

![Linting Status](https://img.shields.io/github/actions/workflow/status/kndndrj/nvim-dbee/lint.yml?label=linting&style=for-the-badge)
![Docgen Status](https://img.shields.io/github/actions/workflow/status/kndndrj/nvim-dbee/docgen.yml?label=docgen&logo=neovim&logoColor=white&style=for-the-badge)
![Backend](https://img.shields.io/badge/go-backend-lightblue?style=for-the-badge&logo=go&logoColor=white)
![Frontend](https://img.shields.io/badge/lua-frontend-blue?style=for-the-badge&logo=lua&logoColor=white)

<!--DOCGEN_START-->

# Neovim DBee

**Database Client for NeoVim!**

**Execute Your Favourite Queries From the Comfort of Your Editor!**

**Backend in Go!**

**Frontend in Lua!**

**Get Results FAST With Under-the-hood Iterator!**

**Integrates with nvim-projector!**

**Bees Love It!**

***Alpha Software - Expect Breaking Changes!***

## Installation

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

This project aims to be as cross-platform as possible, but there are some
limitations (for example some of the go dependencies only work on certain
platforms).

The CI pipeline tries building the binary for all possible GOARCH/GOOS
combinations - the ones that succeed are stored in a
[remote bucket](https://github.com/kndndrj/nvim-dbee-bucket) on it's own branch.
Additionally, the [install manifest](lua/dbee/install/__manifest.lua) gets
created.

So to check if your platform is currently supported, check out the mentioned
manifest

### Manual Binary Installation

The installation examples include the `build`/`run` functions, which get
triggered once the plugin updates. This should be sufficient for the majority of
users. If that doesn't include you, then you have a few options:

- just install with the `"go"` option (this performs `go install` under the
  hood):
  ```lua
  require("dbee").install("go")
  ```
- Download an already compiled binary from one of urls in the
  [install manifest](lua/dbee/install/__manifest.lua)
- `go install` (the install location will vary depending on your local go
  configuration):
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

## Usage

Call the `setup()` function with an optional config parameter. If you are not
using your plugin manager to lazy load for you, make sure to specify
`{ lazy = true }` in the config.

Here is a brief refference of the most useful functions:

```lua
-- Open/close the UI.
require("dbee").open()
require("dbee").close()
-- Next/previou page of the results (there are the same mappings that work just inside the results buffer
-- available in config).
require("dbee").next()
require("dbee").prev()
-- Run a query on the active connection directly.
require("dbee").execute(query)
-- Save the current result to file (format is either "csv" or "json" for now).
require("dbee").save(format, file)
```

### Specifying Connections

Connection represents an instance of the database client (i.e. one database).
This is how it looks like:

```lua
{
  id = "optional_identifier" -- useful to set manually if you want to remove from the file (see below)
                             -- IT'S YOUR JOB TO KEEP THESE UNIQUE!
  name = "My Database",
  type = "sqlite", -- type of database driver
  url = "~/path/to/mydb.db",
}
```

There are a few different ways you can use to specify the connection parameters
for DBee:

- Using the `setup()` function:

  The most straightforward (but probably the most useless) way is to just add
  them to your configuration in `init.lua` like this:

  ```lua
  require("dbee").setup {
  connections = {
    {
      name = "My Database",
      type = "sqlite", -- type of database driver
      url = "~/path/to/mydb.db",
    },
    -- ...
  },
  -- ... the rest of your config
  }
  ```

- Use the prompt at runtime:

  You can add connections manually using the "add connection" item in the
  drawer. Fill in the values and write the buffer (`:w`) to save the connection.
  By default, this will save the connection to the global connections file and
  will persist over restarts.

- Use an env variable. This variable is `DBEE_CONNECTIONS` by default:

  You can export an environment variable with connections from your shell like
  this:

  ```sh
    export DBEE_CONNECTIONS='[
        {
            "name": "DB from env",
            "url": "mysql://...",
            "type": "mysql"
        }
    ]'
  ```

- Use a custom load function:

  If you aren't satisfied with the default capabilities, you can provide your
  own `load` function in the config at setup. This example uses a
  project-specific connections config file:

  ```lua
  local file = vim.fn.getcwd() .. "/.dbee.json"

  require("dbee").setup {
    loader = {
      -- this function must return a list of connections and it doesn't
      -- care about anything else
      load = function()
        return require("dbee.loader").load_from_file(file)
      end,
      -- just as an example you can also specify this function to save any
      -- connections from the prompt input to the same file as they are being loaded from
      add = function(connections)
        require("dbee.loader").add_to_file(file)
      end,
      -- and this to remove them
      remove = function(connections)
        require("dbee.loader").remove_from_file(file)
      end,
    },
    -- ... the rest of your config
  }
  ```

#### Secrets

If you don't want to have secrets laying around your disk in plain text, you can
use the special placeholders in connection strings (this works using any method
for specifying connections).

NOTE: *Currently only envirnoment variables are supported*

Example:

Using the `DBEE_CONNECTIONS` environment variable for specifying connections and
exporting secrets to environment:

```sh
# Define connections
export DBEE_CONNECTIONS='[
    {
        "name": "{{ env.SECRET_DB_NAME }}",
        "url": "postgres://{{ env.SECRET_DB_USER }}:{{ env.SECRET_DB_PASS }}@localhost:5432/{{ env.SECRET_DB_NAME }}?sslmode=disable",
        "type": "postgres"
    }
]'

# Export secrets
export SECRET_DB_NAME="secretdb"
export SECRET_DB_USER="secretuser"
export SECRET_DB_PASS="secretpass"
```

If you start neovim in the same shell, this will evaluate to the following
connection:

```lua
{ {
  name = "secretdb",
  url = "postgres://secretuser:secretpass@localhost:5432/secretdb?sslmode=disable",
  type = "postgres",
} }
```

## Configuration

As mentioned, you can pass an optional table parameter to `setup()` function.

Here are the defaults:

<!--DOCGEN_CONFIG_START-->

<!-- Contents from lua/dbee/config.lua are inserted between these tags for docgen. -->

[`config.lua`](lua/dbee/config.lua)

<!--DOCGEN_CONFIG_END-->

## Projector Integration

DBee is compatible with my other plugin
[nvim-projector](https://github.com/kndndrj/nvim-projector), a
code-runner/project-configurator.

To use dbee with it, simply use `"dbee"` as one of it's outputs.

## Development

Reffer to [ARCHITECTURE.md](ARCHITECTURE.md) for a brief overview of the
architecture.
