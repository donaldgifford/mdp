package cli

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/mdp/internal/server"
	"github.com/donaldgifford/mdp/internal/watcher"
)

const debounceInterval = 50 * time.Millisecond

func newServeCmd() *cobra.Command {
	var (
		port          int
		browser       bool
		theme         string
		hljsTheme     string
		scrollSync    bool
		stdin         bool
		customCSS     string
		openToNetwork bool
		idleTimeout   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "serve <file>",
		Short: "Start a preview server for a markdown file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			slog.Info(
				"starting preview server",
				"version",
				version,
				"commit",
				commit,
				"built",
				date,
				"file",
				file,
				"port",
				port,
				"browser",
				browser,
				"theme",
				theme,
			)

			cfg := server.Config{
				File:          file,
				Port:          port,
				OpenBrowser:   browser,
				Theme:         theme,
				HljsTheme:     hljsTheme,
				ScrollSync:    scrollSync,
				CustomCSS:     customCSS,
				OpenToNetwork: openToNetwork,
				IdleTimeout:   idleTimeout,
			}

			srv, err := server.New(cfg)
			if err != nil {
				return fmt.Errorf("creating server: %w", err)
			}
			defer srv.Close()

			if stdin {
				go srv.ReadStdin(os.Stdin)
			} else {
				cleanup, watchErr := startWatcher(file, srv)
				if watchErr != nil {
					return watchErr
				}
				defer cleanup()
			}

			return srv.ListenAndServe()
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Port to listen on (0 = auto-assign)")
	cmd.Flags().BoolVar(&browser, "browser", true, "Open browser automatically")
	cmd.Flags().StringVar(&theme, "theme", "auto",
		`Preview theme name or path to CSS file (e.g. "auto", "tokyo-night", "/path/to/custom.css")`)
	cmd.Flags().StringVar(&hljsTheme, "hljs-theme", "",
		`Vendored hljs stylesheet for custom theme files (github, github-dark). Only valid with --theme=<file>.`)
	cmd.Flags().BoolVar(&scrollSync, "scroll-sync", true, "Enable scroll sync with cursor position")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read content/cursor updates from stdin (for editor plugins)")
	cmd.Flags().StringVar(&customCSS, "css", "", "Path to custom CSS file to inject after default styles")
	cmd.Flags().BoolVar(&openToNetwork, "open-to-network", false, "Listen on 0.0.0.0 instead of localhost")
	cmd.Flags().DurationVar(&idleTimeout, "idle-timeout", 30*time.Second, "Shut down after no clients connected for this duration (0 = disabled)")

	return cmd
}

// startWatcher creates and starts a file watcher that broadcasts on change.
// It returns a cleanup function and any error.
func startWatcher(file string, srv *server.Server) (cleanup func(), err error) {
	w, err := watcher.New(file, debounceInterval, func() {
		if broadcastErr := srv.BroadcastFile(); broadcastErr != nil {
			slog.Error("broadcast failed", "error", broadcastErr)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	go w.Start()

	return func() {
		if closeErr := w.Close(); closeErr != nil {
			slog.Error("closing watcher", "error", closeErr)
		}
	}, nil
}
