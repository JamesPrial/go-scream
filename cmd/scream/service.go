package main

import (
	"context"
	"io"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/JamesPrial/go-scream/internal/app"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/discord"
	"github.com/JamesPrial/go-scream/internal/encoding"
	"github.com/JamesPrial/go-scream/internal/scream"
)

// newServiceFromConfig constructs a scream.Service and an optional io.Closer
// (the Discord session) from the provided configuration. The caller must
// close the returned closer when done.
func newServiceFromConfig(cfg config.Config, logger *slog.Logger) (*scream.Service, io.Closer, error) {
	gen, err := app.NewGenerator(string(cfg.Backend), logger)
	if err != nil {
		return nil, nil, err
	}

	frameEnc := encoding.NewGopusFrameEncoder(logger)
	fileEnc := app.NewFileEncoder(string(cfg.Format), logger)

	var player discord.VoicePlayer
	var closer io.Closer
	if cfg.Token != "" {
		player, closer, err = app.NewDiscordDeps(cfg.Token, logger)
		if err != nil {
			return nil, nil, err
		}
	}

	svc := scream.NewServiceWithDeps(cfg, gen, fileEnc, frameEnc, player, logger)
	return svc, closer, nil
}

// runWithService creates a signal-notifying context, builds the service from
// cfg, defers closing the session with a warning log on error, then
// delegates to fn.
func runWithService(cfg config.Config, logger *slog.Logger, fn func(ctx context.Context, svc *scream.Service) error) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc, closer, err := newServiceFromConfig(cfg, logger)
	if err != nil {
		return err
	}
	if closer != nil {
		defer func() {
			if cerr := closer.Close(); cerr != nil {
				slog.Warn("failed to close discord session", "error", cerr)
			}
		}()
	}
	return fn(ctx, svc)
}
