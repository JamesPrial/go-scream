package main

import (
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"

	"github.com/JamesPrial/go-scream/internal/audio"
	"github.com/JamesPrial/go-scream/internal/audio/ffmpeg"
	"github.com/JamesPrial/go-scream/internal/audio/native"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/discord"
	"github.com/JamesPrial/go-scream/internal/encoding"
	"github.com/JamesPrial/go-scream/internal/scream"
)

// newServiceFromConfig constructs a scream.Service and an optional io.Closer
// (the Discord session) from the provided configuration. The caller must
// close the returned closer when done.
func newServiceFromConfig(cfg config.Config) (*scream.Service, io.Closer, error) {
	// Select audio generator based on backend.
	var gen audio.AudioGenerator
	if cfg.Backend == config.BackendFFmpeg {
		g, err := ffmpeg.NewFFmpegGenerator()
		if err != nil {
			return nil, nil, err
		}
		gen = g
	} else {
		gen = native.NewNativeGenerator()
	}

	// Create Opus frame encoder.
	frameEnc := encoding.NewGopusFrameEncoder()

	// Create file encoder based on format.
	var fileEnc encoding.FileEncoder
	if cfg.Format == config.FormatWAV {
		fileEnc = encoding.NewWAVEncoder()
	} else {
		fileEnc = encoding.NewOGGEncoder()
	}

	// Create Discord player if token is available.
	var player discord.VoicePlayer
	var closer io.Closer
	if cfg.Token != "" {
		session, err := discordgo.New("Bot " + cfg.Token)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create discord session: %w", err)
		}
		if err := session.Open(); err != nil {
			return nil, nil, fmt.Errorf("failed to open discord session: %w", err)
		}
		closer = session
		sess := &discord.DiscordGoSession{S: session}
		player = discord.NewDiscordPlayer(sess)
	}

	svc := scream.NewServiceWithDeps(cfg, gen, fileEnc, frameEnc, player)
	return svc, closer, nil
}
