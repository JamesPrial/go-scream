package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JamesPrial/go-scream/internal/scream"
)

var presetsCmd = &cobra.Command{
	Use:   "presets",
	Short: "List available scream presets",
	Run:   runPresets,
}

func init() {
	rootCmd.AddCommand(presetsCmd)
}

func runPresets(cmd *cobra.Command, args []string) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Available presets:")
	for _, name := range scream.ListPresets() {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", name)
	}
}
