// Package cli implements the mdp command-line interface.
package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for mdp.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mdp",
		Short: "Markdown preview server for Neovim",
		Long:  "mdp is a Go-based markdown preview server with live reload and scroll sync.",
	}

	root.AddCommand(newServeCmd())

	return root
}
