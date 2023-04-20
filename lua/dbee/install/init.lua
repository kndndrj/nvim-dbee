local M = {}

function M.path()
  return vim.fn.stdpath("data") .. "/dbee/bin"
end

---@param osys string operating system in format of uv.os_uname()
---@param arch string architecture in format of uv.os_uname()
---@return string url address of compiled binary
local function get_url(osys, arch)
  local arch_aliases = {
    ["arm"] = "arm",
    ["aarch64_be"] = "arm64",
    ["aarch64"] = "arm64",
    ["armv8b"] = "arm64", -- compat
    ["armv8l"] = "arm64", -- compat
    ["mips"] = "mips",
    ["mips64"] = "mips64",
    ["ppc64"] = "ppc64",
    ["ppc64le"] = "ppc64le",
    ["s390"] = "s390x", -- compat
    ["s390x"] = "s390x",
    ["i386"] = "386",
    ["i686"] = "386", -- compat
    ["x86_64"] = "amd64",
  }
  -- TODO:
  local os_aliases = {
    ["win"] = "windows",
  }

  if not osys or not arch then
    error("no operating system and arch provided")
  end
  local ok, m = pcall(require, "dbee.install.__manifest")
  if not ok or type(m) ~= "table" or vim.tbl_isempty(m) then
    error('error reading install manifest. try installing with go directly: require("dbee").install("go")')
  end

  local a = arch_aliases[arch] or arch
  local o = os_aliases[string.lower(osys)] or string.lower(osys)
  local key = string.format("dbee_%s_%s", o, a)

  local url = m.urls[key]
  if not url then
    error("no compiled binary found for " .. osys .. "/" .. arch)
  end

  return url
end

---@return string version
local function get_version()
  local version = "latest"
  local ok, m = pcall(require, "dbee.install.__manifest")
  if not ok or type(m) ~= "table" or vim.tbl_isempty(m) then
    print("error reading install manifest. using version: " .. version)
    return version
  end

  return m.version or version
end

---@param command? "wget"|"curl"|"bitsadmin"|"go"
---@return { cmd: string, args: string[], env: { string: string } }
local function get_job(command)
  local uname = vim.loop.os_uname()
  local arch = uname.machine
  local osys = uname.sysname
  local install_dir = M.path()
  local install_binary = install_dir .. "/dbee"

  -- make install dir
  vim.fn.mkdir(install_dir, "p")

  local jobs = {
    wget = function()
      return {
        cmd = "wget",
        args = { "-O", install_binary, get_url(osys, arch) },
        env = {},
      }
    end,
    curl = function()
      return {
        cmd = "curl",
        args = { "-sfLo", install_binary, get_url(osys, arch) },
        env = {},
      }
    end,
    bitsadmin = function()
      return {
        cmd = "bitsadmin",
        args = { "TODO" },
        env = {},
      }
    end,
    go = function()
      return {
        cmd = "go",
        args = { "install", "github.com/kndndrj/nvim-dbee/dbee@" .. get_version() },
        env = { GOBIN = install_dir },
      }
    end,
  }
  -- priority list
  local prio = { "wget", "curl", "bitsadmin", "go" }

  -- if command is provided use it
  if command then
    local job = jobs[command]()
    if job and vim.fn.executable(job.cmd) == 1 then
      return job
    end
    error('"' .. command .. '" is not a supported installation method')
  end

  -- else find the first suitable command
  for _, cmd in ipairs(prio) do
    local job = jobs[cmd]()
    if job and vim.fn.executable(job.cmd) == 1 then
      return job
    end
  end

  error("no suitable command found")
end

---@param command? "wget"|"curl"|"bitsadmin"|"go" preffered command
function M.exec(command)
  -- find a suitable install command
  local job = get_job(command)

  print("installing dbee with: " .. job.cmd)

  -- run the command
  local uv = vim.loop

  -- set env and save the previous values
  -- for some reason setting env on uv.spawn doesnt work
  local saved_env = {}
  for k, v in pairs(job.env) do
    local save = uv.os_getenv(k)
    if not save then
      save = ""
    end
    saved_env[k] = save
    uv.os_setenv(k, v)
  end
  local function cleanup()
    -- restore previous env variables
    for k, v in pairs(saved_env) do
      uv.os_setenv(k, v)
    end
  end

  local handle
  handle = uv.spawn(job.cmd, {
    args = job.args,
    stdio = { nil, 1, 2 },
  }, function(code, _)
    handle:close()
    if code == 0 then
      print("successfully installed")
    else
      print("installation error")
    end
    cleanup()
  end)

  if not handle then
    print("installation error")
    cleanup()
  end
end

return M
