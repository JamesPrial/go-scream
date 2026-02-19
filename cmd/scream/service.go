package main

import (
	"context"
	"fmt"
	"io"
	"os"
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
func newServiceFromConfig(cfg config.Config) (*scream.Service, io.Closer, error) {
	gen, err := app.NewGenerator(string(cfg.Backend))
	if err != nil {
		return nil, nil, err
	}

	frameEnc := encoding.NewGopusFrameEncoder()
	fileEnc := app.NewFileEncoder(string(cfg.Format))

	var player discord.VoicePlayer
	var closer io.Closer
	if cfg.Token != "" {
		player, closer, err = app.NewDiscordDeps(cfg.Token)
		if err != nil {
			return nil, nil, err
		}
	}

	svc := scream.NewServiceWithDeps(cfg, gen, fileEnc, frameEnc, player)
	return svc, closer, nil
}

// runWithService creates a signal-notifying context, builds the service from
// cfg, defers closing the session with a stderr warning on error, then
// delegates to fn.
func runWithService(cfg config.Config, fn func(ctx context.Context, svc *scream.Service) error) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc, closer, err := newServiceFromConfig(cfg)
	if err != nil {
		return err
	}
	if closer != nil {
		defer func() {
			if cerr := closer.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to close session: %v\n", cerr)
			}
		}()
	}
	return fn(ctx, svc)
}
