package server

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
)

// openBrowser opens the given URL in the user's default browser.
func openBrowser(url string) {
	// Respect $BROWSER if set.
	if browser := os.Getenv("BROWSER"); browser != "" {
		run(browser, url)
		return
	}

	switch runtime.GOOS {
	case "darwin":
		run("open", url)
	case "linux":
		run("xdg-open", url)
	case "windows":
		run("cmd", "/c", "start", url)
	default:
		slog.Warn("unsupported platform for auto-open, open manually", "url", url)
	}
}

func run(name string, args ...string) {
	// Use a detached context — the browser outlives the server process.
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		slog.Warn("failed to open browser", "error", err)
	}
}
