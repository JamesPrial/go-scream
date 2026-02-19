// Package main is the entry point for the scream CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JamesPrial/go-scream/pkg/version"
)

var (
	configPath string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:     "scream",
	Short:   "A Discord voice bot that generates synthetic screams",
	Version: version.String(),
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "path to config file (YAML)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
