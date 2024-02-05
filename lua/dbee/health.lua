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

-- Gets git hash of the install manifest
---@return string
local function get_manifest_hash()
  return install.version()
end

function M.check()
  vim.health.start("DBee report")

  if vim.fn.executable(install.bin()) ~= 1 then
    vim.health.error("Binary not executable: " .. install.bin() .. ".")
    return
  end

  if vim.fn.executable("git") ~= 1 then
    vim.health.warn("Git not installed -- could not determine binary version.")
    return
  end

  local go_hash = get_go_hash()
  local current_hash = get_current_hash()
  local manifest_hash = get_manifest_hash()

  if go_hash == "unknown" then
    vim.health.error("Could not determine binary version.")
    return
  end

  if go_hash == current_hash then
    vim.health.ok("Binary version matches version of current HEAD.")
    return
  elseif go_hash == manifest_hash then
    vim.health.ok("Binary version matches version of install manifest.")
    return
  end

  vim.health.error(
    string.format(
      "Binary version %q doesn't match either:\n  - current hash: %q or\n  - hash of install manifest %q.",
      go_hash,
      current_hash,
      manifest_hash
    )
  )
end

return M
