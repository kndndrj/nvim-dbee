local M = {}

---@alias install_command "wget"|"curl"|"bitsadmin"|"go"|"cgo"

-- NOTE: don't use vim.notify in loop callbacks
local function log_error(mes)
  print("[dbee install - error]: " .. mes)
end
local function log_info(mes)
  print("[dbee install]: " .. mes)
end

function M.path()
  return vim.fn.stdpath("data") .. "/dbee/bin"
end

function M.source_path()
  return debug.getinfo(1).source:sub(2):gsub("/lua/dbee/install/init.lua$", "/dbee")
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
    ["windows_nt"] = "windows",
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
  local key = string.format("%s/%s", o, a)

  local url = m.urls[key]
  if not url then
    error("no compiled binary found for " .. osys .. "/" .. arch)
  end

  return url
end

---@param command? install_command
---@return { cmd: string, args: string[], env: { string: string } }[]
local function get_job(command)
  local uname = vim.loop.os_uname()
  local arch = uname.machine
  local osys = uname.sysname
  local install_dir = M.path()
  local install_binary = install_dir .. "/dbee"

  -- make install dir
  vim.fn.mkdir(install_dir, "p")

  local chmod = {
    cmd = "chmod",
    args = { "+x", install_binary },
    env = {},
  }

  local jobs_list = {
    wget = function()
      return {
        {
          cmd = "wget",
          args = { "-qO", install_binary, get_url(osys, arch) },
          env = {},
        },
        chmod,
      }
    end,
    curl = function()
      return {
        {
          cmd = "curl",
          args = { "-sfLo", install_binary, get_url(osys, arch) },
          env = {},
        },
        chmod,
      }
    end,
    bitsadmin = function()
      return {
        {
          cmd = "bitsadmin",
          args = { "TODO" },
          env = {},
        },
      }
    end,
    go = function()
      return {
        {
          cmd = "go",
          args = { "build", "-C", M.source_path(), "-o", install_binary },
          env = {},
        },
      }
    end,
    cgo = function()
      return {
        {
          cmd = "go",
          args = { "build", "-C", M.source_path(), "-o", install_binary },
          env = { CGO_ENABLED = "1" },
        },
      }
    end,
  }
  -- priority list
  local prio_job_list = { "wget", "curl", "bitsadmin", "go" }

  -- if command is provided use it
  if command then
    local jobs = jobs_list[command]() or {}
    for _, j in ipairs(jobs) do
      if vim.fn.executable(j.cmd) ~= 1 then
        error('"' .. command .. '" is not a supported installation method')
      end
    end
    return jobs
  end

  -- else find the first suitable command
  for _, cmd in ipairs(prio_job_list) do
    local jobs = jobs_list[cmd]() or {}
    local ignore = false
    for _, j in ipairs(jobs) do
      if vim.fn.executable(j.cmd) ~= 1 then
        ignore = true
        break
      end
    end
    if not ignore then
      return jobs
    end
  end

  error("no suitable installation method found")
end

---@param jobs table jobs to run in order
---@param index integer index of the job in jobs table
local function run_jobs(jobs, index)
  local job = jobs[index]
  if not job then
    return
  end
  log_info("running command: " .. job.cmd)
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
      if index >= #jobs then
        log_info("successfully installed")
        return
      end
      run_jobs(jobs, index + 1)
    else
      log_error("command: " .. job.cmd .. " exited with code " .. tostring(code))
    end
    cleanup()
  end)

  if not handle then
    log_error("could not spawn command: " .. job.cmd)
    cleanup()
  end
end

---@param command? install_command preffered command
function M.exec(command)
  -- find a suitable install command
  local jobs = get_job(command)

  local msg = ""
  for _, j in ipairs(jobs) do
    msg = msg .. " " .. j.cmd
  end
  log_info("installing dbee with: " .. msg)

  run_jobs(jobs, 1)
end

return M
