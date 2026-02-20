package scream

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/JamesPrial/go-scream/internal/audio"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/discord"
	"github.com/JamesPrial/go-scream/internal/encoding"
)

// Service orchestrates audio generation, encoding, and Discord voice playback.
type Service struct {
	cfg       config.Config
	generator audio.Generator
	fileEnc   encoding.FileEncoder
	frameEnc  encoding.OpusFrameEncoder
	player    discord.VoicePlayer
	logger    *slog.Logger
}

// NewServiceWithDeps constructs a Service with all dependencies explicitly
// injected. It never returns nil. The player argument may be nil when the
// service is used in DryRun mode or for file generation only. Callers must
// pass an untyped nil (not a typed-nil interface value) when no player is needed.
func NewServiceWithDeps(
	cfg config.Config,
	gen audio.Generator,
	fileEnc encoding.FileEncoder,
	frameEnc encoding.OpusFrameEncoder,
	player discord.VoicePlayer,
	logger *slog.Logger,
) *Service {
	return &Service{
		cfg:       cfg,
		generator: gen,
		fileEnc:   fileEnc,
		frameEnc:  frameEnc,
		player:    player,
		logger:    logger,
	}
}

// Play generates a scream and streams it to the specified Discord voice channel.
// It validates guildID, checks for a configured player (unless DryRun is set),
// and checks for a pre-cancelled context before proceeding.
func (s *Service) Play(ctx context.Context, guildID, channelID string) error {
	if guildID == "" {
		return config.ErrMissingGuildID
	}

	if !s.cfg.DryRun && s.player == nil {
		return ErrNoPlayer
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	s.logger.Debug("resolving audio params", "preset", s.cfg.Preset, "duration", s.cfg.Duration, "volume", s.cfg.Volume)

	params, err := resolveParams(s.cfg)
	if err != nil {
		return err
	}

	s.logger.Debug("generating audio")

	pcm, err := s.generator.Generate(params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGenerateFailed, err)
	}

	s.logger.Debug("encoding frames")

	frameCh, errCh := s.frameEnc.EncodeFrames(pcm, params.SampleRate, params.Channels)

	if s.cfg.DryRun {
		s.logger.Info("dry-run: encoding and discarding frames")
		for range frameCh {
		}
		encErr := <-errCh
		if encErr != nil {
			return fmt.Errorf("%w: %w", ErrEncodeFailed, encErr)
		}
		return nil
	}

	s.logger.Debug("streaming to discord", "guild", guildID, "channel", channelID)

	playErr := s.player.Play(ctx, guildID, channelID, frameCh)
	encErr := <-errCh

	if playErr != nil {
		return fmt.Errorf("%w: %w", ErrPlayFailed, playErr)
	}
	if encErr != nil {
		return fmt.Errorf("%w: %w", ErrEncodeFailed, encErr)
	}

	return nil
}

// Generate creates a scream and writes it to dst using the configured file encoder.
// It does not require a Discord token or player.
func (s *Service) Generate(ctx context.Context, dst io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.logger.Debug("resolving audio params", "preset", s.cfg.Preset, "duration", s.cfg.Duration, "volume", s.cfg.Volume)

	params, err := resolveParams(s.cfg)
	if err != nil {
		return err
	}

	s.logger.Debug("generating audio")

	pcm, err := s.generator.Generate(params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGenerateFailed, err)
	}

	s.logger.Debug("encoding to file")

	if err := s.fileEnc.Encode(dst, pcm, params.SampleRate, params.Channels); err != nil {
		return fmt.Errorf("%w: %w", ErrEncodeFailed, err)
	}

	return nil
}

// ListPresets returns the names of all available scream presets.
func ListPresets() []string {
	names := audio.AllPresets()
	result := make([]string, len(names))
	for i, n := range names {
		result[i] = string(n)
	}
	return result
}

// resolveParams derives audio.ScreamParams from the provided Config.
// If cfg.Preset is set, it looks up the named preset and returns an error
// if the name is unknown. If cfg.Preset is empty, Randomize is used to
// generate random parameters. In either case, a positive cfg.Duration
// overrides the duration from the preset or random params.
//
// cfg.Volume is a linear multiplier where 1.0 means no change. It is
// converted to decibels and applied as an offset to FilterParams.VolumeBoostDB
// so that the existing preset/random boost is scaled by the user's intent.
func resolveParams(cfg config.Config) (audio.ScreamParams, error) {
	var params audio.ScreamParams

	if cfg.Preset != "" {
		p, ok := audio.GetPreset(audio.PresetName(cfg.Preset))
		if !ok {
			return audio.ScreamParams{}, ErrUnknownPreset
		}
		params = p
	} else {
		params = audio.Randomize(0)
	}

	if cfg.Duration > 0 {
		params.Duration = cfg.Duration
	}

	// Apply volume: cfg.Volume is a linear multiplier (1.0 = no change).
	// Convert to dB and add to the existing VolumeBoostDB so that the preset
	// or randomized boost is offset by the user's intent. When Volume == 1.0,
	// log10(1.0) == 0, so this is a no-op and remains backward-compatible.
	if cfg.Volume > 0 {
		params.Filter.VolumeBoostDB += 20 * math.Log10(cfg.Volume)
	}

	return params, nil
}
