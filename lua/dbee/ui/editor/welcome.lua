local M = {}

function M.banner()
  return {
    "-- [ Enter insert mode to clear ]",
    "",
    "",
    "-- Welcome to",
    "-- ",
    "--  ██████████   ███████████",
    "-- ░░███░░░░███ ░░███░░░░░███",
    "--  ░███   ░░███ ░███    ░███  ██████   ██████",
    "--  ░███    ░███ ░██████████  ███░░███ ███░░███",
    "--  ░███    ░███ ░███░░░░░███░███████ ░███████",
    "--  ░███    ███  ░███    ░███░███░░░  ░███░░░",
    "--  ██████████   ███████████ ░░██████ ░░██████",
    "-- ░░░░░░░░░░   ░░░░░░░░░░░   ░░░░░░   ░░░░░░",
    "",
    "",
    '-- Type ":h dbee.txt" to learn more about the plugin.',
    "",
    '-- Report issues to: "github.com/kndndrj/nvim-dbee/issues".',
    "",
    "-- Existing users: DO NOT PANIC:",
    "-- Your notes and connections were moved from:",
    '-- "' .. vim.fn.stdpath("cache") .. '/dbee/notes" and',
    '-- "' .. vim.fn.stdpath("cache") .. '/dbee/persistence.json"',
    "-- to:",
    '-- "' .. vim.fn.stdpath("state") .. '/dbee/notes" and',
    '-- "' .. vim.fn.stdpath("state") .. '/dbee/persistence.json"',
    "-- Move them manually or adjust the config accordingly.",
    '-- see the "Breaking Changes" issue on github for more info.',
  }
end

return M
