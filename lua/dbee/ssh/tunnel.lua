local M = {}

local uv = vim.loop

--- Start an SSH tunnel.
---@param ssh_config table SSH configuration table
---@return table|nil, string|nil Process handle or nil, error message or nil
function M.start_tunnel(ssh_config)
  local executable
  local args = {}

  -- Check if we're using password authentication
  if ssh_config and ssh_config.password then
    executable = "sshpass"
    table.insert(args, "-p")
    table.insert(args, ssh_config.password)
    table.insert(args, "ssh")
  elseif ssh_config and ssh_config.ssh_file then
    executable = "ssh"
    table.insert(args, "-i")
    table.insert(args, ssh_config.ssh_file)
  else
    executable = "ssh"
  end

  -- Ensure required flags for tunneling are present
  local has_L = false
  local has_N = false

  -- Add ssh options
  if ssh_config and ssh_config.options then
    for _, opt in ipairs(ssh_config.options) do
      if opt == "-L" then
        has_L = true
      end
      if opt == "-N" then
        has_N = true
      end
      table.insert(args, opt)
    end
  end

  -- Add ssh key file if provided
  if ssh_config and ssh_config.ssh_file then
    table.insert(args, 1, "-i")
    table.insert(args, 2, ssh_config.ssh_file)
  end

  -- Add -N flag if not present (prevents executing remote commands, good for tunneling)
  if not has_N then
    table.insert(args, "-N")
  end

  -- Make sure we have -L for port forwarding if not already in options
  if not has_L then
    table.insert(args, "-L")
  end

  -- Add local and remote port forwarding
  table.insert(args, string.format("%d:localhost:%d", ssh_config.local_port, ssh_config.remote_port))

  -- Add user and host
  local user_host = ""
  if ssh_config and ssh_config.user and ssh_config.host then
    user_host = string.format("%s@%s", ssh_config.user, ssh_config.host)
  else
    return nil, "SSH configuration is missing user or host"
  end
  table.insert(args, user_host)

  -- Create pipes for stdout and stderr
  local stdout = uv.new_pipe(false)
  local stderr = uv.new_pipe(false)

  -- Store output for error reporting
  local stdout_data = ""
  local stderr_data = ""

  -- Spawn the process
  local handle, err = uv.spawn(executable, {
    args = args,
    stdio = { nil, stdout, stderr },
    detached = false,
    hide = false,
  }, function(code, signal)
    print("SSH tunnel exited with code:", code, "and signal:", signal)

    -- Close the pipes
    if stdout then
      stdout:close()
    end
    if stderr then
      stderr:close()
    end

    if code ~= 0 then
      print("SSH tunnel error:", stderr_data)
    end
  end)

  if not handle then
    stdout:close()
    stderr:close()
    return nil, "Failed to start SSH tunnel: " .. (err or "unknown error")
  end

  -- Read stdout
  stdout:read_start(function(stdout_err, data)
    if stdout_err then
      print("Error reading from stdout:", stdout_err)
    end
    if data then
      stdout_data = stdout_data .. data
    end
  end)

  -- Read stderr
  stderr:read_start(function(stderr_err, data)
    if stderr_err then
      print("Error reading from stderr:", stderr_err)
    end
    if data then
      stderr_data = stderr_data .. data

      -- Check for common error patterns
      if
        stderr_data:match("Connection refused")
        or stderr_data:match("Permission denied")
        or stderr_data:match("Address already in use")
      then
        print("SSH tunnel error detected:", stderr_data)
      end
    end
  end)

  -- Set a timeout to verify the tunnel is established
  local timeout_timer = uv.new_timer()
  timeout_timer:start(5000, 0, function()
    timeout_timer:stop()
    timeout_timer:close()

    -- If we already have error data, log it to help with debugging
    if stderr_data and stderr_data ~= "" then
      print("SSH tunnel stderr output:", stderr_data)
      -- Common SSH error detection
      if stderr_data:match("Permission denied") then
        print("SSH authentication failed. Check your username, password or SSH key.")
      elseif stderr_data:match("No route to host") then
        print("Cannot reach SSH host. Check your network connection and host address.")
      elseif stderr_data:match("Connection refused") then
        print("SSH connection refused. Ensure the SSH server is running and accessible.")
      elseif stderr_data:match("Host key verification failed") then
        print("SSH host key verification failed. You may need to add this host to known_hosts.")
      end
    end

    -- Check if the tunnel is working by attempting to connect to the local port
    local client = uv.new_tcp()
    local connection_check_completed = false

    client:connect("127.0.0.1", ssh_config.local_port, function(connect_err)
      if connect_err then
        print("SSH tunnel verification failed: could not connect to local port", ssh_config.local_port, connect_err)

        -- If we have process info, check if it's still running
        if handle then
          local pid = uv.process_get_pid(handle)
          if pid then
            print("SSH process is still running with PID:", pid)

            -- Run netstat/ss command to check if the port is actually bound
            local netstat_cmd = "ss -lnptu | grep " .. ssh_config.local_port
            local cmd_handle, cmd_err = uv.spawn("bash", {
              args = { "-c", netstat_cmd },
              stdio = { nil, stdout, stderr },
            }, function(netstat_code)
              if netstat_code ~= 0 then
                print("Debugging: Port " .. ssh_config.local_port .. " is not bound by any process.")

                -- Check common ssh client errors
                if stderr_data:match("Permission denied") then
                  print("Authentication failed: check your password/ssh key")
                elseif stderr_data:match("Connection closed by") then
                  print("Connection was accepted but immediately closed by server")
                elseif stderr_data:match("Bad configuration option") then
                  print("SSH configuration error in the command arguments")
                elseif stderr_data and stderr_data ~= "" then
                  print("SSH error output:", stderr_data)
                end

                -- Kill the process since tunnel failed to establish
                pcall(function()
                  if handle then
                    uv.process_kill(handle, "sigterm")
                  end
                end)
              end
            end)

            if not cmd_handle and cmd_err then
              print("Failed to run network diagnostics:", cmd_err)
            end
          else
            print("SSH process is no longer running")
          end
        end

        -- Don't close stderr or stdout pipes here, wait for the process to exit naturally
      else
        print("SSH tunnel verification successful: local port", ssh_config.local_port, "is accessible")
      end

      if not connection_check_completed then
        connection_check_completed = true
        pcall(function()
          if client then
            client:close()
          end
        end)
      end
    end)

    -- Set a short timeout for the connection attempt
    local connection_timer = uv.new_timer()
    connection_timer:start(1000, 0, function()
      if not connection_check_completed then
        connection_check_completed = true
        pcall(function()
          client:close()
        end)
      end
      connection_timer:close()
    end)
  end)

  return handle, nil
end

--- Stop an SSH tunnel.
---@param handle table Process handle of the SSH tunnel
---@param force boolean Optional, if true uses SIGKILL instead of SIGTERM
function M.stop_tunnel(handle, force)
  if handle then
    local signal = force and "sigkill" or "sigterm"
    local success = uv.process_kill(handle, signal)

    if not success then
      print("Failed to stop SSH tunnel with signal", signal)
      -- If SIGTERM failed and force wasn't specified, try SIGKILL
      if not force then
        return M.stop_tunnel(handle, true)
      end
    end
  end
end

return M
