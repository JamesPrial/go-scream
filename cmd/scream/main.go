// Package main is the entry point for the scream CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JamesPrial/go-scream/pkg/version"
)

var (
	configPath   string
	verbose      bool
	logLevelFlag string
)

var rootCmd = &cobra.Command{
	Use:     "scream",
	Short:   "A Discord voice bot that generates synthetic screams",
	Version: version.String(),
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "path to config file (YAML)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "", "log level (debug|info|warn|error)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
