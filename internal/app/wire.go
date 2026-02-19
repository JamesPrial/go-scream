// Package app provides shared wiring helpers used by the go-scream binaries.
// It centralises generator selection, encoder creation, and Discord session
// construction so that cmd/scream and cmd/skill do not duplicate that logic.
package app

import (
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"

	"github.com/JamesPrial/go-scream/internal/audio"
	"github.com/JamesPrial/go-scream/internal/audio/ffmpeg"
	"github.com/JamesPrial/go-scream/internal/audio/native"
	"github.com/JamesPrial/go-scream/internal/discord"
	"github.com/JamesPrial/go-scream/internal/encoding"
)

// backendFFmpeg is the string value that selects the ffmpeg audio generator.
// It matches config.BackendFFmpeg without importing the config package here.
const backendFFmpeg = "ffmpeg"

// formatWAV is the string value that selects WAV output.
// It matches config.FormatWAV without importing the config package here.
const formatWAV = "wav"

// NewGenerator selects and returns an audio.Generator based on the backend
// string. When backend is "ffmpeg", it locates the ffmpeg binary on PATH and
// returns an ffmpeg-backed generator. Any other value returns the native Go
// generator. Returns an error only when the ffmpeg backend is requested but
// the ffmpeg binary cannot be found.
func NewGenerator(backend string) (audio.Generator, error) {
	if backend == backendFFmpeg {
		g, err := ffmpeg.NewGenerator()
		if err != nil {
			return nil, err
		}
		return g, nil
	}
	return native.NewGenerator(), nil
}

// NewFileEncoder returns a FileEncoder for the given format string. When
// format is "wav", a WAVEncoder is returned. Any other value returns an
// OGGEncoder. NewFileEncoder never returns nil.
func NewFileEncoder(format string) encoding.FileEncoder {
	if format == formatWAV {
		return encoding.NewWAVEncoder()
	}
	return encoding.NewOGGEncoder()
}

// NewDiscordDeps creates a discordgo session for the given bot token, opens
// the WebSocket connection, and returns a ready-to-use VoicePlayer together
// with an io.Closer that must be called to close the session when done.
// On any error both returned values are nil.
func NewDiscordDeps(token string) (discord.VoicePlayer, io.Closer, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create discord session: %w", err)
	}
	if err := session.Open(); err != nil {
		return nil, nil, fmt.Errorf("failed to open discord session: %w", err)
	}
	sess := &discord.GoSession{S: session}
	player := discord.NewPlayer(sess)
	return player, session, nil
}
