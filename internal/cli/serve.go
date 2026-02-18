package cli

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/mdp/internal/server"
	"github.com/donaldgifford/mdp/internal/watcher"
)

const debounceInterval = 50 * time.Millisecond

func newServeCmd() *cobra.Command {
	var (
		port    int
		browser bool
	)

	cmd := &cobra.Command{
		Use:   "serve <file>",
		Short: "Start a preview server for a markdown file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			slog.Info("starting preview server", "file", file, "port", port, "browser", browser)

			cfg := server.Config{
				File:        file,
				Port:        port,
				OpenBrowser: browser,
			}

			srv, err := server.New(cfg)
			if err != nil {
				return fmt.Errorf("creating server: %w", err)
			}
			defer srv.Close()

			// Start file watcher.
			w, err := watcher.New(file, debounceInterval, func() {
				if broadcastErr := srv.BroadcastFile(); broadcastErr != nil {
					slog.Error("broadcast failed", "error", broadcastErr)
				}
			})
			if err != nil {
				return fmt.Errorf("creating watcher: %w", err)
			}
			defer func() {
				if closeErr := w.Close(); closeErr != nil {
					slog.Error("closing watcher", "error", closeErr)
				}
			}()
			go w.Start()

			return srv.ListenAndServe()
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Port to listen on (0 = auto-assign)")
	cmd.Flags().BoolVar(&browser, "browser", true, "Open browser automatically")

	return cmd
}
