-- build.lua — Auto-detected by lazy.nvim when the plugin is installed/updated.
-- Downloads a pre-built binary from GitHub releases.
-- Falls back to building from source if no release is available.

local repo = "donaldgifford/mdp"
local binary_name = "mdp"

local plugin_dir = vim.fn.fnamemodify(
  debug.getinfo(1, "S").source:sub(2),
  ":p:h"
)
local bin_dir = plugin_dir .. "/bin"

--- Detect OS and architecture.
---@return string|nil platform e.g. "darwin_arm64"
local function detect_platform()
  local os_name = vim.loop.os_uname().sysname:lower()
  local arch = vim.loop.os_uname().machine

  if os_name == "darwin" then
    os_name = "darwin"
  elseif os_name == "linux" then
    os_name = "linux"
  else
    return nil
  end

  if arch == "x86_64" or arch == "amd64" then
    arch = "amd64"
  elseif arch == "arm64" or arch == "aarch64" then
    arch = "arm64"
  else
    return nil
  end

  return os_name .. "_" .. arch
end

--- Get the latest release tag from GitHub API.
---@return string|nil tag
local function get_latest_release()
  local url = "https://api.github.com/repos/"
    .. repo
    .. "/releases/latest"
  local result = vim.fn.system({ "curl", "-fsSL", url })
  if vim.v.shell_error ~= 0 then
    return nil
  end

  local tag = result:match('"tag_name"%s*:%s*"([^"]+)"')
  return tag
end

--- Download and extract the release binary.
---@return boolean success
local function download_binary()
  local platform = detect_platform()
  if not platform then
    return false
  end

  coroutine.yield("Detected platform: " .. platform)

  local tag = get_latest_release()
  if not tag then
    coroutine.yield("No release found, will build from source")
    return false
  end

  coroutine.yield("Downloading " .. tag .. "...")

  local url = string.format(
    "https://github.com/%s/releases/download/%s/%s_%s.tar.gz",
    repo,
    tag,
    binary_name,
    platform
  )

  local tmp_dir = vim.fn.tempname()
  vim.fn.mkdir(tmp_dir, "p")

  local archive = tmp_dir .. "/archive.tar.gz"
  vim.fn.system({ "curl", "-fsSL", url, "-o", archive })
  if vim.v.shell_error ~= 0 then
    vim.fn.delete(tmp_dir, "rf")
    return false
  end

  vim.fn.system({
    "tar",
    "-xzf",
    archive,
    "-C",
    tmp_dir,
  })
  if vim.v.shell_error ~= 0 then
    vim.fn.delete(tmp_dir, "rf")
    return false
  end

  vim.fn.mkdir(bin_dir, "p")

  local binary_path = tmp_dir .. "/" .. binary_name
  if vim.fn.filereadable(binary_path) ~= 1 then
    -- Search for it (GoReleaser may nest the binary).
    local found = vim.fn.glob(tmp_dir .. "/**/" .. binary_name, false, true)
    if #found == 0 then
      vim.fn.delete(tmp_dir, "rf")
      return false
    end
    binary_path = found[1]
  end

  vim.fn.rename(binary_path, bin_dir .. "/" .. binary_name)
  vim.fn.system({ "chmod", "+x", bin_dir .. "/" .. binary_name })
  vim.fn.delete(tmp_dir, "rf")

  coroutine.yield("Installed " .. binary_name .. " " .. tag)
  return true
end

--- Build the binary from source using Go.
---@return boolean success
local function build_from_source()
  if vim.fn.executable("go") ~= 1 then
    coroutine.yield(
      "Go not found. Install from https://go.dev/dl/"
    )
    return false
  end

  coroutine.yield("Building from source...")
  vim.fn.mkdir(bin_dir, "p")

  local output = vim.fn.system(string.format(
    "cd %s && go build -o %s/%s ./cmd/mdp",
    vim.fn.shellescape(plugin_dir),
    vim.fn.shellescape(bin_dir),
    binary_name
  ))

  if vim.v.shell_error ~= 0 then
    coroutine.yield("Build failed: " .. output)
    return false
  end

  coroutine.yield("Built " .. binary_name .. " from source")
  return true
end

--- Main build entry point (runs as a coroutine by lazy.nvim).
return function()
  if download_binary() then
    return
  end

  coroutine.yield("Download failed, falling back to source...")
  if not build_from_source() then
    coroutine.yield("Install failed. Run :MdpInstall! to retry.")
  end
end
