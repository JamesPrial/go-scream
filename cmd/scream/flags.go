package main

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/JamesPrial/go-scream/internal/config"
)

var (
	tokenFlag    string
	presetFlag   string
	durationFlag time.Duration
	volumeFlag   float64
	backendFlag  string
	formatFlag   string
	outputFlag   string
	dryRunFlag   bool
)

// buildConfig constructs a Config via: Default -> YAML -> env -> CLI flags.
func buildConfig(cmd *cobra.Command) (config.Config, error) {
	cfg := config.Default()

	if configPath != "" {
		fileCfg, err := config.Load(configPath)
		if err != nil {
			return cfg, err
		}
		cfg = config.Merge(cfg, fileCfg)
	}

	config.ApplyEnv(&cfg)

	// Apply only explicitly-set CLI flags.
	if cmd.Flags().Changed("token") {
		cfg.Token = tokenFlag
	}
	if cmd.Flags().Changed("preset") {
		cfg.Preset = presetFlag
	}
	if cmd.Flags().Changed("duration") {
		cfg.Duration = durationFlag
	}
	if cmd.Flags().Changed("volume") {
		cfg.Volume = volumeFlag
	}
	if cmd.Flags().Changed("backend") {
		cfg.Backend = config.BackendType(backendFlag)
	}
	if cmd.Flags().Changed("format") {
		cfg.Format = config.FormatType(formatFlag)
	}
	if cmd.Flags().Changed("output") {
		cfg.OutputFile = outputFlag
	}
	if cmd.Flags().Changed("dry-run") {
		cfg.DryRun = dryRunFlag
	}
	if cmd.Flags().Changed("verbose") {
		cfg.Verbose = verbose
	}

	return cfg, nil
}

// addAudioFlags adds shared audio flags to a command.
func addAudioFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&presetFlag, "preset", "", "scream preset name")
	cmd.Flags().DurationVar(&durationFlag, "duration", 0, "scream duration (e.g. 3s, 500ms)")
	cmd.Flags().Float64Var(&volumeFlag, "volume", 0, "volume multiplier [0.0-1.0]")
	cmd.Flags().StringVar(&backendFlag, "backend", "", "audio backend (native|ffmpeg)")
}
