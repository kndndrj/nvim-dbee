local M = {}

---@param new_scratchpad_dir string
function M.banner(new_scratchpad_dir)
  local old_scratchpad_dir = vim.fn.stdpath("cache") .. "/dbee/scratches"

  return {
    "[ Enter insert mode to clear ]",
    "",
    "",
    "Welcome to",
    "",
    " ██████████   ███████████",
    "░░███░░░░███ ░░███░░░░░███",
    " ░███   ░░███ ░███    ░███  ██████   ██████",
    " ░███    ░███ ░██████████  ███░░███ ███░░███",
    " ░███    ░███ ░███░░░░░███░███████ ░███████",
    " ░███    ███  ░███    ░███░███░░░  ░███░░░",
    " ██████████   ███████████ ░░██████ ░░██████",
    "░░░░░░░░░░   ░░░░░░░░░░░   ░░░░░░   ░░░░░░",
    "",
    "",
    'Type ":h dbee.txt" to learn more about the plugin.',
    "",
    'Report issues to: "github.com/kndndrj/nvim-dbee/issues".',
    "",
    "",
    "NOTE TO EXISTING USERS:",
    "Your scratchpads are safe, scratchpad directory just changed.",
    "Now you have global scratchpads (which you had before) and local, which are",
    "available per-connection.",
    "",
    'I advise you to look into the old directory: "' .. old_scratchpad_dir .. '"',
    'and move the scratchpads to the new global one: "' .. new_scratchpad_dir .. '".',
    "",
    "If you are confident, you can just execute the following command:",
    "",
    '"mv ' .. old_scratchpad_dir .. "/* " .. new_scratchpad_dir .. '/global/"',
    "",
    "I rather didn't automatize this migration, because I believe you preffer to know where",
    "your data is stored.",
    'Oh BTW, If you haven\'t figured out yet, scratchpads are now called "notes".',
  }
end

return M
