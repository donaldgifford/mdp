# Logging

mdp writes all server and plugin activity to a single log file. This makes it
straightforward to confirm which binary is running, trace connection events, and
diagnose issues without enabling verbose mode.

## Default location

```
~/.local/state/nvim/mdp.log
```

This follows the XDG Base Directory specification, placing mdp logs alongside
other Neovim state files (e.g. shada, undo history). The directory is created
automatically on first use.

## Watching logs in real time

```bash
tail -f ~/.local/state/nvim/mdp.log
```

## What gets logged

### Session boundaries

Each server run is delimited by a start and end marker so multiple sessions in
the same log file are easy to distinguish:

```
=== mdp session start: /path/to/file.md ===
...
=== mdp session end (exit 0) ===
```

The start marker is written by the Lua plugin before launching the process. The
end marker is written in the `on_exit` job callback after the process exits.

### Server startup

The first log line after the session marker is written by the Go binary itself
and includes the version, commit hash, build date, and effective configuration:

```
time=2026-02-24T14:00:00Z level=INFO msg="starting preview server" version=v0.2.0 commit=abc1234 built=2026-02-24T13:00:00Z file=/path/to/file.md port=0 browser=true
time=2026-02-24T14:00:00Z level=INFO msg=serving addr=http://127.0.0.1:52691 file=/path/to/file.md
```

The `version` field is populated via ldflags at build time:

- **Release binary** (downloaded by `build.lua` / `install.sh`): shows the
  tagged release version, e.g. `v0.2.0`
- **Source build** (`:MdpInstall!` or fallback): shows `git describe` output,
  e.g. `v0.2.0-4-gabc1234-dirty`, including commit distance and dirty state

This makes it immediately clear which binary is active without running
`--version` separately.

### Idle shutdown

When the Lua plugin detects that the last markdown buffer has been closed, it
logs the countdown before starting the timer:

```
[mdp] no markdown buffers open, idle shutdown in 30s
```

The server-side idle watcher (active when browser tabs close) logs separately
through the Go binary's structured logger:

```
time=... level=INFO msg="no clients connected, idle timer started" timeout=30s
time=... level=INFO msg="idle timeout reached, shutting down"
```

Both paths lead to `=== mdp session end ===` in the log.

## How logging works

mdp runs as a child process of Neovim. The Go binary writes structured log
output (`log/slog`, text handler) to **stderr**. The Lua plugin captures
`on_stderr` from the Neovim job and appends each line to the log file using
`io.open` in append mode.

The Lua plugin also writes its own plaintext entries directly (session markers,
idle shutdown messages) using the same `write_log` helper. All output is
interleaved in arrival order.

```
Neovim (Lua)                   mdp (Go)
    |                               |
    |-- write_log("session start") -|
    |-- jobstart(cmd) ------------> |-- slog --> stderr
    |<- on_stderr (captured) -------|
    |-- write_log(stderr lines) ----|
    |                               |
    |  [buffer closed]              |
    |-- write_log("idle shutdown") -|
    |-- timer (30s) --------------> |
    |-- jobstop() ---------------> SIGTERM
    |<- on_exit ------------------- |-- slog --> stderr (session end)
    |-- write_log("session end") ---|
```

## Configuration

### Change log path

```lua
require("mdp").setup({
  log_file = vim.fn.expand("~/logs/mdp.log"),
})
```

Or in a lazy.nvim spec:

```lua
{
  "donaldgifford/mdp",
  opts = {
    log_file = vim.fn.expand("~/logs/mdp.log"),
  },
}
```

### Disable logging

Set `log_file` to an empty string:

```lua
opts = {
  log_file = "",
}
```

### Enable verbose (debug) logging

The Go binary supports a `--verbose` / `-v` flag that enables `DEBUG` level
structured logs. This is not currently exposed as a plugin option. To use it,
run the binary directly:

```bash
mdp serve --verbose --stdin file.md
```

Or temporarily pass it by modifying the `cmd` table in `M.start()` inside
`lua/mdp/init.lua`.
