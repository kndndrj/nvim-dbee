<!-- Any html tags, badges etc. go before this tag. -->

<!--docgen-start-->

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

Using Packer:

```lua
use {
  "kndndrj/nvim-dbee",
  requires = {
    "MunifTanjim/nui.nvim",
  },
  config = function()
    require("dbee").setup()
  end
}
```

## Quick Start

Call the `setup()` function with an optional config parameter.
If you are not using your plugin manager to lazy load for you, make sure to
specify `{ lazy = true }` in the config.

Here is a brief refference of the most useful functions:

```lua
require("dbee").open() -- open UI
require("dbee").close() -- close UI
require("dbee").next() -- next page when results are ready
require("dbee").prev() -- previous page when results are ready
require("dbee").execute(query) -- run a query on the active connection directly
require("dbee").save(format, file) -- save the current result to file (format is either "csv" or "json" for now).
```

## Configuration

As mentioned, you can pass an optional table parameter to `setup()` function.

Here are the defaults:

<!--docgen-config-start-->

<!-- Contents from lua/dbee/config.lua are inserted between these tags. -->

[`config.lua`](lua/dbee/config.lua)

<!--docgen-config-end-->

## Projector Integration

DBee is compatible with my other plugin [nvim-projector](https://github.com/kndndrj/nvim-projector), a code-runner/project-configurator.

To use dbee with it, simply use `"dbee"` as one of it's outputs.

## Development

Reffer to [ARCHITECTURE.md](ARCHITECTURE.md) for a brief overview of the architecture.
