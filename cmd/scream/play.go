package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/JamesPrial/go-scream/internal/app"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/scream"
)

var playCmd = &cobra.Command{
	Use:   "play <guildID> [channelID]",
	Short: "Generate and play a scream in a Discord voice channel",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPlay,
}

func init() {
	rootCmd.AddCommand(playCmd)
	playCmd.Flags().StringVar(&tokenFlag, "token", "", "Discord bot token")
	addAudioFlags(playCmd)
	playCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "generate and encode but do not play")
}

func runPlay(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig(cmd)
	if err != nil {
		return err
	}

	cfg.GuildID = args[0]

	channelID := ""
	if len(args) > 1 {
		channelID = args[1]
	}

	if cfg.Token == "" {
		return config.ErrMissingToken
	}

	if err := config.Validate(cfg); err != nil {
		return err
	}

	logger := app.SetupLogger(cfg)
	logger.Info("playing scream", "guild", cfg.GuildID, "channel", channelID)

	return runWithService(cfg, logger, func(ctx context.Context, svc *scream.Service) error {
		return svc.Play(ctx, cfg.GuildID, channelID)
	})
}
