// Package cli implements the mdp command-line interface.
package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for mdp.
func NewRootCmd() *cobra.Command {
	var verbose bool

	root := &cobra.Command{
		Use:   "mdp",
		Short: "Markdown preview server for Neovim",
		Long:  "mdp is a Go-based markdown preview server with live reload and scroll sync.",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			level := slog.LevelInfo
			if verbose {
				level = slog.LevelDebug
			}
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: level,
			})))
		},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose (debug) logging")
	root.AddCommand(newServeCmd())

	return root
}
