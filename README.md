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

![Screenshot](assets/screenshot.jpg)

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

## Quick Start

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
