local install = require("dbee.install")

local M = {}

---@param cmd string
---@return string
local function run_cmd(cmd)
  local handle = assert(io.popen(cmd))
  local result = handle:read("*all")
  handle:close()

  return string.gsub(result, "\n", "") or ""
end

---@return string _ path of git repo
local function repo()
  local p, _ = debug.getinfo(1).source:sub(2):gsub("/lua/dbee/health.lua$", "/")
  return p
end

-- Gets a git hash from which the go binary is compiled.
---@return string
local function get_go_hash()
  return run_cmd(string.format("%s -version", install.bin()))
end

-- Gets currently checked out git hash.
---@return string
local function get_current_hash()
  return run_cmd(string.format("git -C %q rev-parse HEAD", repo()))
end

-- Gets git hash of last tagged commit.
---@return string
local function get_last_tag_hash()
  local last_tag = run_cmd(string.format("git -C %q describe --tags --abbrev=0", repo()))
  return run_cmd(string.format("git -C %q rev-list -n 1 tags/%s", repo(), last_tag))
end

function M.check()
  vim.health.report_start("DBee report")

  if vim.fn.executable(install.bin()) ~= 1 then
    vim.health.report_error("Binary not executable: " .. install.bin() .. ".")
    return
  end

  if vim.fn.executable("git") ~= 1 then
    vim.health.report_warn("Git not installed -- could not determine binary version.")
    return
  end

  local go_hash = get_go_hash()
  local current_hash = get_current_hash()
  local last_tag_hash = get_last_tag_hash()

  if go_hash == current_hash or go_hash == last_tag_hash then
    vim.health.report_ok("Binary versions match")
    return
  end

  vim.health.report_error(
    string.format(
      "Binary version %q doesn't match either:\n  - current hash: %q or\n  - hash of latest tag %q.",
      go_hash,
      current_hash,
      last_tag_hash
    )
  )
end

return M
