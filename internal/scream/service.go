package scream

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/JamesPrial/go-scream/internal/audio"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/discord"
	"github.com/JamesPrial/go-scream/internal/encoding"
)

// Service orchestrates audio generation, encoding, and Discord voice playback.
type Service struct {
	cfg       config.Config
	generator audio.AudioGenerator
	fileEnc   encoding.FileEncoder
	frameEnc  encoding.OpusFrameEncoder
	player    discord.VoicePlayer
	closer    io.Closer
}

// NewServiceWithDeps constructs a Service with all dependencies explicitly
// injected. It never returns nil. The player argument may be nil when the
// service is used in DryRun mode or for file generation only.
func NewServiceWithDeps(
	cfg config.Config,
	gen audio.AudioGenerator,
	fileEnc encoding.FileEncoder,
	frameEnc encoding.OpusFrameEncoder,
	player discord.VoicePlayer,
) *Service {
	// Normalize typed-nil interfaces (e.g. (*ConcreteType)(nil)) to untyped nil
	// so that s.player == nil works correctly in Play().
	if player != nil {
		v := reflect.ValueOf(player)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			player = nil
		}
	}
	return &Service{
		cfg:       cfg,
		generator: gen,
		fileEnc:   fileEnc,
		frameEnc:  frameEnc,
		player:    player,
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

	params, err := resolveParams(s.cfg)
	if err != nil {
		return err
	}

	pcm, err := s.generator.Generate(params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGenerateFailed, err)
	}

	frameCh, errCh := s.frameEnc.EncodeFrames(pcm, params.SampleRate, params.Channels)

	if s.cfg.DryRun {
		for range frameCh {
		}
		encErr := <-errCh
		if encErr != nil {
			return fmt.Errorf("%w: %w", ErrEncodeFailed, encErr)
		}
		return nil
	}

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

	params, err := resolveParams(s.cfg)
	if err != nil {
		return err
	}

	pcm, err := s.generator.Generate(params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGenerateFailed, err)
	}

	if err := s.fileEnc.Encode(dst, pcm, params.SampleRate, params.Channels); err != nil {
		return fmt.Errorf("%w: %w", ErrEncodeFailed, err)
	}

	return nil
}

// Close releases any resources held by the service. If no closer was set,
// it returns nil.
func (s *Service) Close() error {
	if s.closer == nil {
		return nil
	}
	return s.closer.Close()
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
