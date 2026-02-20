// Package main is the OpenClaw skill wrapper binary for go-scream.
// It resolves a Discord token, builds configuration, and wires up the
// scream service to play audio in a Discord voice channel.
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/JamesPrial/go-scream/internal/app"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/encoding"
	"github.com/JamesPrial/go-scream/internal/scream"
)

// openclawConfig mirrors the relevant fields of the ~/.openclaw/openclaw.json
// file. Extra fields in the JSON are silently ignored.
type openclawConfig struct {
	Channels struct {
		Discord struct {
			Token string `json:"token"`
		} `json:"discord"`
	} `json:"channels"`
}

// parseOpenClawConfig reads the JSON file at path and returns the Discord
// token found at .channels.discord.token. It returns an error if the file
// cannot be read or the content is not valid JSON. If the token field is
// absent or empty, ("", nil) is returned.
func parseOpenClawConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("skill: failed to read config file %s: %w", path, err)
	}

	var cfg openclawConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("skill: failed to parse config file %s: %w", path, err)
	}

	return cfg.Channels.Discord.Token, nil
}

// resolveToken determines the Discord token by checking sources in priority
// order:
//  1. DISCORD_TOKEN environment variable (if non-empty)
//  2. parseOpenClawConfig(openclawPath) (errors are silently ignored)
//
// Returns an empty string if no source yields a token.
func resolveToken(openclawPath string) string {
	if v := os.Getenv("DISCORD_TOKEN"); v != "" {
		return v
	}

	token, err := parseOpenClawConfig(openclawPath)
	if err != nil {
		// Silently ignore file parse errors and fall through to empty.
		return ""
	}

	return token
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: skill <guildID> [channelID]")
		os.Exit(1)
	}

	guildID := os.Args[1]

	channelID := ""
	if len(os.Args) > 2 {
		channelID = os.Args[2]
	}

	openclawPath := filepath.Join(os.Getenv("HOME"), ".openclaw", "openclaw.json")
	token := resolveToken(openclawPath)
	if token == "" {
		fmt.Fprintln(os.Stderr, "skill: discord token is required (set DISCORD_TOKEN or configure ~/.openclaw/openclaw.json)")
		os.Exit(1)
	}

	cfg := config.Default()
	// ApplyEnv loads audio parameter overrides (SCREAM_PRESET, SCREAM_DURATION,
	// SCREAM_VOLUME, SCREAM_BACKEND). Token and GuildID are set explicitly below
	// from skill-specific sources, overriding any env values ApplyEnv may set.
	config.ApplyEnv(&cfg)
	cfg.Token = token
	cfg.GuildID = guildID

	logger := app.SetupLogger(cfg)

	if err := config.Validate(cfg); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := app.SignalContext()
	defer stop()

	gen, err := app.NewGenerator(cfg.Backend, logger)
	if err != nil {
		slog.Error("failed to create generator", "error", err)
		os.Exit(1)
	}

	frameEnc := encoding.NewGopusFrameEncoder(logger)
	fileEnc := app.NewFileEncoder(cfg.Format, logger)

	player, sessionCloser, err := app.NewDiscordDeps(cfg.Token, logger)
	if err != nil {
		slog.Error("failed to create discord session", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := sessionCloser.Close(); cerr != nil {
			slog.Warn("failed to close discord session", "error", cerr)
		}
	}()

	svc := scream.NewServiceWithDeps(cfg, gen, fileEnc, frameEnc, player, logger)
	if err := svc.Play(ctx, cfg.GuildID, channelID); err != nil {
		slog.Error("playback failed", "error", err)
		os.Exit(1)
	}
}
