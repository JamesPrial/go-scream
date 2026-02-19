package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/scream"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a scream and save to a file",
	RunE:  runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output file path (required)")
	_ = generateCmd.MarkFlagRequired("output")
	addAudioFlags(generateCmd)
	generateCmd.Flags().StringVar(&formatFlag, "format", "", "output format (ogg|wav)")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig(cmd)
	if err != nil {
		return err
	}

	if err := config.Validate(cfg); err != nil {
		return err
	}

	if cfg.Verbose {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Generating scream to %s (format: %s)\n", cfg.OutputFile, cfg.Format)
	}

	return runWithService(cfg, func(ctx context.Context, svc *scream.Service) error {
		f, err := os.Create(cfg.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to close output file: %v\n", cerr)
			}
		}()
		return svc.Generate(ctx, f)
	})
}
