local M = {}

function M.path()
  return vim.fn.stdpath("data") .. "/dbee/bin"
end

---@param command? "wget"|"curl"|"bitsadmin"|"go"
---@return { cmd: string, args: string[], env: { string: string } }|nil
local function get_job(command)
  local ok, v = pcall(require, "dbee.__version")
  local version = "latest"
  if ok and type(v) == "string" and v ~= "" then
    version = v
  end
  -- TODO: detect
  local arch = "x86"
  local osys = "linux"

  local remote_binary = string.format("nvim-dbee_%s_%s.bin", arch, osys)
  local link = string.format("https://github.com/kndndrj/nvim-dbee/releases/download/%s/%s", version, remote_binary)
  local install_dir = M.path()
  local install_binary = install_dir .. "/dbee"

  -- make install dir
  vim.fn.mkdir(install_dir, "p")

  local jobs = {
    wget = {
      cmd = "wget",
      args = { "-O", install_binary, link },
      env = {},
    },
    curl = {
      cmd = "curl",
      args = { "-sfLo", install_binary, link },
      env = {},
    },
    bitsadmin = {
      cmd = "bitsadmin",
      args = { "TODO" },
      env = {},
    },
    go = {
      cmd = "go",
      args = { "install", "github.com/kndndrj/nvim-dbee/dbee@" .. version },
      env = { GOBIN = install_dir },
    },
  }

  -- if command is provided use it
  if command then
    if command and jobs[command] and vim.fn.executable(jobs[command].cmd) == 1 then
      return jobs[command]
    end
    return
  end

  -- else find the first suitable command
  for _, params in pairs(jobs) do
    if vim.fn.executable(params.cmd) == 1 then
      return params
    end
  end
end

---@param command? "wget"|"curl"|"bitsadmin"|"go" preffered command
function M.exec(command)
  -- find a suitable install command
  local job = get_job(command)
  if not job then
    local prefix = ""
    if command then
      prefix = command .. ": "
    end
    print(prefix .."command not found for installation")
    return
  end

  print("installing with: " .. job.cmd)

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
