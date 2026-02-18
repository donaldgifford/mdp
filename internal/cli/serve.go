package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/mdp/internal/server"
)

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

			return srv.ListenAndServe()
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Port to listen on (0 = auto-assign)")
	cmd.Flags().BoolVar(&browser, "browser", true, "Open browser automatically")

	return cmd
}
