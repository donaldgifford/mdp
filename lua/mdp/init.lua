-- mdp.nvim — Markdown preview plugin for Neovim
-- Provides :MdpStart, :MdpStop, :MdpToggle, :MdpOpen commands.

local M = {}

--- Default log file path: ~/.local/state/nvim/mdp.log (XDG-compliant).
local default_log_file = vim.fn.stdpath("log") .. "/mdp.log"

--- Default configuration.
local defaults = {
  port = 0,
  browser = true,
  theme = "auto",
  scroll_sync = true,
  idle_timeout_secs = 30, -- Shut down after this many seconds with no browser tab open (0 = disabled).
  log_file = default_log_file, -- Server log output. Empty string disables logging.
  binary = "", -- Empty means auto-detect via exepath.
  debounce_ms = 300,
}

--- Active state.
local state = {
  job_id = nil,
  addr = nil,
  augroup = nil,
  cursor_timer = nil,
  content_timer = nil,
  shutdown_timer = nil, -- Lua-side idle timer started when last markdown buffer closes.
}

--- Merged user configuration.
local config = {}

--- Plugin root directory (3 levels up from lua/mdp/init.lua).
local plugin_dir = vim.fn.fnamemodify(debug.getinfo(1, "S").source:sub(2), ":h:h:h")

--- Append lines to the configured log file, if one is set.
---@param lines string[]
local function write_log(lines)
  if not config.log_file or config.log_file == "" then
    return
  end
  local f = io.open(config.log_file, "a")
  if not f then
    return
  end
  for _, line in ipairs(lines) do
    if line ~= "" then
      f:write(line .. "\n")
    end
  end
  f:close()
end

--- Resolve the mdp binary path.
--- Search order: config.binary > plugin bin/ dir > PATH
---@return string|nil path, string|nil error
local function resolve_binary()
  if config.binary ~= "" then
    if vim.fn.executable(config.binary) == 1 then
      return config.binary, nil
    end
    return nil, "mdp binary not found at: " .. config.binary
  end

  -- Check the plugin's own bin/ directory (populated by install.sh).
  local local_bin = plugin_dir .. "/bin/mdp"
  if vim.fn.executable(local_bin) == 1 then
    return local_bin, nil
  end

  -- Check PATH (e.g., from go install).
  local path = vim.fn.exepath("mdp")
  if path ~= "" then
    return path, nil
  end

  return nil, "mdp binary not found. Run :MdpInstall or install with: go install github.com/donaldgifford/mdp/cmd/mdp@latest"
end

--- Send a JSON message to the mdp process via stdin.
---@param msg table
local function send_message(msg)
  if not state.job_id then
    return
  end

  local ok, encoded = pcall(vim.fn.json_encode, msg)
  if not ok then
    vim.notify("[mdp] Failed to encode message", vim.log.levels.ERROR)
    return
  end

  vim.fn.chansend(state.job_id, encoded .. "\n")
end

--- Send the current buffer content to mdp.
local function send_content()
  local lines = vim.api.nvim_buf_get_lines(0, 0, -1, false)
  local content = table.concat(lines, "\n") .. "\n"
  local file = vim.api.nvim_buf_get_name(0)

  send_message({
    type = "content",
    data = content,
    file = file,
  })
end

--- Send the current cursor position to mdp.
local function send_cursor()
  local line = vim.api.nvim_win_get_cursor(0)[1]
  send_message({
    type = "cursor",
    line = line,
  })
end

--- Debounced content send for TextChangedI.
local function debounced_content()
  if state.content_timer then
    vim.fn.timer_stop(state.content_timer)
  end
  state.content_timer = vim.fn.timer_start(config.debounce_ms, function()
    state.content_timer = nil
    send_content()
  end)
end

--- Throttled cursor send (~60fps = ~16ms, but 50ms is sufficient).
local function throttled_cursor()
  if state.cursor_timer then
    return
  end
  state.cursor_timer = vim.fn.timer_start(50, function()
    state.cursor_timer = nil
    send_cursor()
  end)
end

--- Check if the current buffer is a markdown file.
---@return boolean
local function is_markdown_buffer()
  local ft = vim.bo.filetype
  return ft == "markdown" or ft == "mdx"
end

--- Start the shutdown timer if no markdown buffers are loaded.
--- Called via vim.schedule so the buffer list reflects the final state
--- after a delete/unload event has fully completed.
local function maybe_start_shutdown_timer()
  vim.schedule(function()
    if not state.job_id or config.idle_timeout_secs <= 0 then
      return
    end
    if state.shutdown_timer then
      return
    end
    for _, buf in ipairs(vim.api.nvim_list_bufs()) do
      if vim.api.nvim_buf_is_loaded(buf) then
        local ft = vim.api.nvim_get_option_value("filetype", { buf = buf })
        if ft == "markdown" or ft == "mdx" then
          return
        end
      end
    end
    -- No markdown buffers remain — start the shutdown timer.
    local captured_job = state.job_id
    state.shutdown_timer = vim.fn.timer_start(
      config.idle_timeout_secs * 1000,
      function()
        state.shutdown_timer = nil
        if state.job_id == captured_job then
          M.stop()
        end
      end
    )
  end)
end

--- Set up autocmds for buffer sync and cursor tracking.
local function setup_autocmds()
  if state.augroup then
    return
  end

  state.augroup = vim.api.nvim_create_augroup("MdpPreview", { clear = true })

  -- Content sync on save.
  vim.api.nvim_create_autocmd("BufWritePost", {
    group = state.augroup,
    pattern = { "*.md", "*.mdx", "*.markdown" },
    callback = send_content,
  })

  -- Content sync on insert mode changes (debounced).
  vim.api.nvim_create_autocmd("TextChangedI", {
    group = state.augroup,
    pattern = { "*.md", "*.mdx", "*.markdown" },
    callback = debounced_content,
  })

  -- Cursor sync.
  if config.scroll_sync then
    vim.api.nvim_create_autocmd({ "CursorMoved", "CursorMovedI" }, {
      group = state.augroup,
      pattern = { "*.md", "*.mdx", "*.markdown" },
      callback = throttled_cursor,
    })
  end

  -- Switch preview when entering a different markdown buffer.
  vim.api.nvim_create_autocmd("BufEnter", {
    group = state.augroup,
    pattern = { "*.md", "*.mdx", "*.markdown" },
    callback = function()
      if state.job_id then
        -- Cancel any pending buffer-close shutdown — user is back in markdown.
        if state.shutdown_timer then
          vim.fn.timer_stop(state.shutdown_timer)
          state.shutdown_timer = nil
        end
        send_content()
        if config.scroll_sync then
          send_cursor()
        end
      end
    end,
  })

  -- Start idle shutdown when the last markdown buffer is closed.
  -- BufDelete fires before the buffer is removed; vim.schedule defers the scan
  -- to after the event completes so the buffer list is in its final state.
  -- Listening to all three events covers delete, wipeout, and unload paths used
  -- by buffer management plugins (bufferline.nvim, etc.).
  vim.api.nvim_create_autocmd({ "BufDelete", "BufWipeout", "BufUnload" }, {
    group = state.augroup,
    callback = maybe_start_shutdown_timer,
  })
end

--- Remove autocmds.
local function teardown_autocmds()
  if state.augroup then
    vim.api.nvim_del_augroup_by_id(state.augroup)
    state.augroup = nil
  end

  if state.cursor_timer then
    vim.fn.timer_stop(state.cursor_timer)
    state.cursor_timer = nil
  end
  if state.content_timer then
    vim.fn.timer_stop(state.content_timer)
    state.content_timer = nil
  end
  if state.shutdown_timer then
    vim.fn.timer_stop(state.shutdown_timer)
    state.shutdown_timer = nil
  end
end

--- Start the mdp preview server.
function M.start()
  if state.job_id then
    vim.notify("[mdp] Already running", vim.log.levels.WARN)
    return
  end

  if not is_markdown_buffer() then
    vim.notify("[mdp] Not a markdown buffer", vim.log.levels.WARN)
    return
  end

  local binary, err = resolve_binary()
  if not binary then
    vim.notify("[mdp] " .. err, vim.log.levels.ERROR)
    return
  end

  local file = vim.api.nvim_buf_get_name(0)
  if file == "" then
    vim.notify("[mdp] Buffer has no file name. Save first.", vim.log.levels.WARN)
    return
  end

  local cmd = {
    binary, "serve",
    "--stdin",
    "--scroll-sync=" .. tostring(config.scroll_sync),
    "--theme", config.theme,
    "--idle-timeout=" .. tostring(config.idle_timeout_secs) .. "s",
  }

  if config.port > 0 then
    table.insert(cmd, "--port")
    table.insert(cmd, tostring(config.port))
  end

  if config.browser then
    table.insert(cmd, "--browser")
  else
    table.insert(cmd, "--browser=false")
  end

  table.insert(cmd, file)

  write_log({ "=== mdp session start: " .. file .. " ===" })

  local job_id = vim.fn.jobstart(cmd, {
    stdin_mode = "pipe",
    on_stdout = function(_, data)
      write_log(data)
      for _, line in ipairs(data) do
        local addr = line:match("addr=http://([%w%.%-:]+)")
        if addr then
          state.addr = addr
        end
      end
    end,
    on_stderr = function(_, data)
      write_log(data)
      for _, line in ipairs(data) do
        if line ~= "" then
          local addr = line:match("addr=http://([%w%.%-:]+)")
          if addr then
            state.addr = addr
          end
        end
      end
    end,
    on_exit = function(_, exit_code)
      write_log({ "=== mdp session end (exit " .. exit_code .. ") ===" })
      if exit_code ~= 0 and state.job_id then
        vim.schedule(function()
          vim.notify("[mdp] Process exited with code " .. exit_code, vim.log.levels.WARN)
        end)
      end
      state.job_id = nil
      state.addr = nil
      teardown_autocmds()
    end,
  })

  if job_id <= 0 then
    vim.notify("[mdp] Failed to start process", vim.log.levels.ERROR)
    return
  end

  state.job_id = job_id
  setup_autocmds()

  -- Send initial content after a brief delay for server startup.
  vim.defer_fn(function()
    if state.job_id then
      send_content()
    end
  end, 200)

  vim.notify("[mdp] Preview started")
end

--- Stop the mdp preview server.
function M.stop()
  if not state.job_id then
    vim.notify("[mdp] Not running", vim.log.levels.WARN)
    return
  end

  vim.fn.jobstop(state.job_id)
  state.job_id = nil
  state.addr = nil
  teardown_autocmds()

  vim.notify("[mdp] Preview stopped")
end

--- Toggle the mdp preview server.
function M.toggle()
  if state.job_id then
    M.stop()
  else
    M.start()
  end
end

--- Show the preview for the current buffer.
--- If the server is not running, starts it (opens browser automatically).
--- If already running, pushes the current buffer and opens a browser tab.
--- Stopping is handled by idle timeout or :MdpStop — this command never stops.
function M.preview()
  if not state.job_id then
    if not is_markdown_buffer() then
      vim.notify("[mdp] Not a markdown buffer", vim.log.levels.WARN)
      return
    end
    M.start()
    return
  end

  -- Already running: sync current buffer if markdown, then open browser.
  if is_markdown_buffer() then
    send_content()
    if config.scroll_sync then
      send_cursor()
    end
  end
  M.open()
end

--- Re-open the browser without restarting the server.
function M.open()
  if not state.addr then
    vim.notify("[mdp] Not running", vim.log.levels.WARN)
    return
  end

  local url = "http://" .. state.addr
  local open_cmd

  if vim.fn.has("mac") == 1 then
    open_cmd = { "open", url }
  elseif vim.fn.has("wsl") == 1 then
    open_cmd = { "wslview", url }
  elseif vim.fn.executable("xdg-open") == 1 then
    open_cmd = { "xdg-open", url }
  else
    vim.notify("[mdp] Cannot detect browser opener", vim.log.levels.ERROR)
    return
  end

  vim.fn.jobstart(open_cmd, { detach = true })
end

--- Check if mdp is currently running.
---@return boolean
function M.is_running()
  return state.job_id ~= nil
end

--- Run the install script. Pass "--source" to force a build from source.
---@param args table|nil Command arguments from nvim_create_user_command.
function M.install(args)
  local install_script = plugin_dir .. "/scripts/install.sh"
  if vim.fn.filereadable(install_script) ~= 1 then
    vim.notify("[mdp] Install script not found: " .. install_script, vim.log.levels.ERROR)
    return
  end

  local cmd = { "bash", install_script }
  local bang = args and args.bang
  if bang then
    table.insert(cmd, "--source")
  end

  vim.notify("[mdp] Installing... " .. (bang and "(from source)" or "(downloading release)"))

  vim.fn.jobstart(cmd, {
    cwd = plugin_dir,
    on_exit = function(_, exit_code)
      vim.schedule(function()
        if exit_code == 0 then
          vim.notify("[mdp] Install complete")
        else
          vim.notify("[mdp] Install failed (exit " .. exit_code .. ")", vim.log.levels.ERROR)
        end
      end)
    end,
  })
end

--- Plugin setup function.
---@param opts table|nil User configuration.
function M.setup(opts)
  config = vim.tbl_deep_extend("force", defaults, opts or {})

  -- Ensure the log directory exists.
  if config.log_file and config.log_file ~= "" then
    vim.fn.mkdir(vim.fn.fnamemodify(config.log_file, ":h"), "p")
  end

  -- Register commands.
  vim.api.nvim_create_user_command("MdpPreview", M.preview, { desc = "Show markdown preview (start or switch buffer)" })
  vim.api.nvim_create_user_command("MdpStart", M.start, { desc = "Start mdp preview" })
  vim.api.nvim_create_user_command("MdpStop", M.stop, { desc = "Stop mdp preview" })
  vim.api.nvim_create_user_command("MdpToggle", M.toggle, { desc = "Toggle mdp preview" })
  vim.api.nvim_create_user_command("MdpOpen", M.open, { desc = "Re-open mdp preview in browser" })
  vim.api.nvim_create_user_command("MdpInstall", M.install, {
    desc = "Install/update mdp binary (use ! for source build)",
    bang = true,
  })

  -- Clean up on Neovim exit.
  vim.api.nvim_create_autocmd("VimLeavePre", {
    callback = function()
      if state.job_id then
        vim.fn.jobstop(state.job_id)
      end
    end,
  })
end

return M
