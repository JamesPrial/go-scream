// Package app provides shared wiring helpers used by the go-scream binaries.
// It centralises generator selection, encoder creation, Discord session
// construction, logger setup, and signal context creation so that cmd/scream
// and cmd/skill do not duplicate that logic.
package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"github.com/JamesPrial/go-scream/internal/audio"
	"github.com/JamesPrial/go-scream/internal/audio/ffmpeg"
	"github.com/JamesPrial/go-scream/internal/audio/native"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/discord"
	"github.com/JamesPrial/go-scream/internal/encoding"
)

// NewGenerator selects and returns an audio.Generator based on the backend
// type. When backend is config.BackendFFmpeg, it locates the ffmpeg binary on
// PATH and returns an ffmpeg-backed generator. Any other value returns the
// native Go generator. Returns an error only when the ffmpeg backend is
// requested but the ffmpeg binary cannot be found.
func NewGenerator(backend config.BackendType, logger *slog.Logger) (audio.Generator, error) {
	if backend == config.BackendFFmpeg {
		g, err := ffmpeg.NewGenerator(logger)
		if err != nil {
			return nil, err
		}
		return g, nil
	}
	return native.NewGenerator(logger), nil
}

// NewFileEncoder returns a FileEncoder for the given format type. When format
// is config.FormatWAV, a WAVEncoder is returned. Any other value returns an
// OGGEncoder. NewFileEncoder never returns nil.
func NewFileEncoder(format config.FormatType, logger *slog.Logger) encoding.FileEncoder {
	if format == config.FormatWAV {
		return encoding.NewWAVEncoder(logger)
	}
	return encoding.NewOGGEncoder(logger)
}

// NewDiscordDeps creates a discordgo session for the given bot token, opens
// the WebSocket connection, and returns a ready-to-use VoicePlayer together
// with an io.Closer that must be called to close the session when done.
// On any error both returned values are nil.
//
// The discordgo session's log level is derived from logger so that internal
// DAVE E2EE diagnostics are visible when go-scream runs with --log-level debug.
func NewDiscordDeps(token string, logger *slog.Logger) (discord.VoicePlayer, io.Closer, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	// Bridge discordgo's logging into slog so DAVE diagnostics appear in
	// go-scream's structured log output.
	session.LogLevel = discordLogLevel(logger)
	discordgo.Logger = func(msgL, caller int, format string, a ...interface{}) {
		var lvl slog.Level
		switch msgL {
		case discordgo.LogError:
			lvl = slog.LevelError
		case discordgo.LogWarning:
			lvl = slog.LevelWarn
		case discordgo.LogInformational:
			lvl = slog.LevelInfo
		default:
			lvl = slog.LevelDebug
		}
		logger.Log(context.Background(), lvl, fmt.Sprintf(format, a...), "source", "discordgo")
	}

	if err := session.Open(); err != nil {
		return nil, nil, fmt.Errorf("failed to open discord session: %w", err)
	}
	sess := &discord.GoSession{S: session, Logger: logger}
	player := discord.NewPlayer(sess, logger)
	return player, session, nil
}

// discordLogLevel maps the slog logger's effective level to a discordgo
// LogLevel constant. It probes from most verbose to least so that the
// discordgo session emits messages at the matching threshold.
func discordLogLevel(logger *slog.Logger) int {
	ctx := context.Background()
	switch {
	case logger.Enabled(ctx, slog.LevelDebug):
		return discordgo.LogDebug
	case logger.Enabled(ctx, slog.LevelInfo):
		return discordgo.LogInformational
	case logger.Enabled(ctx, slog.LevelWarn):
		return discordgo.LogWarning
	default:
		return discordgo.LogError
	}
}

// SetupLogger creates a *slog.Logger whose level is resolved from cfg via
// config.ParseLogLevel, writing to stderr with a text handler. It also sets
// the returned logger as the default slog logger.
func SetupLogger(cfg config.Config) *slog.Logger {
	level := config.ParseLogLevel(cfg)
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// SignalContext returns a context that is cancelled when SIGINT or SIGTERM is
// received, together with a stop function that releases the signal resources.
// Callers must invoke the returned cancel function (e.g. via defer) to avoid
// leaking resources.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
