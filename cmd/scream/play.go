package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

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

	if cfg.Verbose {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Playing scream in guild %s", cfg.GuildID)
		if channelID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), " channel %s", channelID)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}

	return runWithService(cfg, func(ctx context.Context, svc *scream.Service) error {
		return svc.Play(ctx, cfg.GuildID, channelID)
	})
}
