// Package main is the entry point for mdp.
package main

import (
	"fmt"
	"os"

	"github.com/donaldgifford/mdp/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
