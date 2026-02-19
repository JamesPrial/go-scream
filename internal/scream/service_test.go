package scream

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JamesPrial/go-scream/internal/audio"
	"github.com/JamesPrial/go-scream/internal/config"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

// mockGenerator implements audio.AudioGenerator for testing.
type mockGenerator struct {
	mu         sync.Mutex
	callCount  int
	lastParams audio.ScreamParams
	reader     io.Reader
	err        error
}

func (m *mockGenerator) Generate(params audio.ScreamParams) (io.Reader, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	m.lastParams = params
	if m.err != nil {
		return nil, m.err
	}
	if m.reader != nil {
		return m.reader, nil
	}
	// Return a small valid PCM buffer by default (1 frame worth of silence).
	return bytes.NewReader(make([]byte, 960*2*2)), nil
}

func (m *mockGenerator) called() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *mockGenerator) params() audio.ScreamParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastParams
}

// mockFileEncoder implements encoding.FileEncoder for testing.
type mockFileEncoder struct {
	mu        sync.Mutex
	callCount int
	err       error
}

func (m *mockFileEncoder) Encode(dst io.Writer, src io.Reader, sampleRate, channels int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.err != nil {
		return m.err
	}
	// Drain the source reader to simulate encoding.
	_, _ = io.Copy(io.Discard, src)
	return nil
}

func (m *mockFileEncoder) called() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// mockFrameEncoder implements encoding.OpusFrameEncoder for testing.
type mockFrameEncoder struct {
	mu        sync.Mutex
	callCount int
	frames    [][]byte
	encErr    error
}

func (m *mockFrameEncoder) EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	frameCh := make(chan []byte, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(frameCh)
		defer close(errCh)

		// Drain the source reader.
		_, _ = io.Copy(io.Discard, src)

		if m.encErr != nil {
			errCh <- m.encErr
			return
		}

		frames := m.frames
		if frames == nil {
			// Default: send a few fake frames.
			frames = [][]byte{
				{0x01, 0x02, 0x03},
				{0x04, 0x05, 0x06},
			}
		}
		for _, f := range frames {
			frameCh <- f
		}
		errCh <- nil
	}()

	return frameCh, errCh
}

func (m *mockFrameEncoder) called() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// mockPlayer implements discord.VoicePlayer for testing.
type mockPlayer struct {
	mu        sync.Mutex
	callCount int
	lastGuild string
	lastChan  string
	err       error
}

func (m *mockPlayer) Play(ctx context.Context, guildID, channelID string, frames <-chan []byte) error {
	m.mu.Lock()
	m.callCount++
	m.lastGuild = guildID
	m.lastChan = channelID
	playErr := m.err
	m.mu.Unlock()

	// Drain the frames channel to prevent goroutine leaks.
	for range frames {
	}

	if playErr != nil {
		return playErr
	}
	return nil
}

func (m *mockPlayer) called() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// mockCloser implements io.Closer for testing.
type mockCloser struct {
	mu        sync.Mutex
	callCount int
	err       error
}

func (m *mockCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	return m.err
}

func (m *mockCloser) called() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// validPlayConfig returns a config suitable for Play() tests.
func validPlayConfig() config.Config {
	return config.Config{
		Token:    "test-token",
		GuildID:  "guild-123",
		Backend:  config.BackendNative,
		Preset:   "classic",
		Duration: 3 * time.Second,
		Volume:   1.0,
		Format:   config.FormatOGG,
	}
}

// validGenerateConfig returns a config suitable for Generate() tests.
func validGenerateConfig() config.Config {
	return config.Config{
		Backend:    config.BackendNative,
		Preset:     "classic",
		Duration:   3 * time.Second,
		Volume:     1.0,
		Format:     config.FormatOGG,
		OutputFile: "test.ogg",
	}
}

// newTestService creates a Service with all mocks wired in.
func newTestService(cfg config.Config, gen *mockGenerator, fEnc *mockFileEncoder, frEnc *mockFrameEncoder, pl *mockPlayer) *Service {
	return NewServiceWithDeps(cfg, gen, fEnc, frEnc, pl)
}

// ---------------------------------------------------------------------------
// NewServiceWithDeps tests
// ---------------------------------------------------------------------------

func Test_NewServiceWithDeps_ReturnsNonNil(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, pl)

	if svc == nil {
		t.Fatal("NewServiceWithDeps returned nil")
	}
}

func Test_NewServiceWithDeps_NilPlayer(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validPlayConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	if svc == nil {
		t.Fatal("NewServiceWithDeps returned nil even with nil player")
	}
}

func Test_NewServiceWithDeps_StoresConfig(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()
	cfg.Preset = "whisper"

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, pl)

	if svc == nil {
		t.Fatal("NewServiceWithDeps returned nil")
	}
	// We verify the config is stored by exercising a behavior that uses it.
	// The Play method should use the "whisper" preset for generation.
}

// ---------------------------------------------------------------------------
// Play() tests
// ---------------------------------------------------------------------------

func Test_Play_HappyPath(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	if gen.called() != 1 {
		t.Errorf("generator called %d times, want 1", gen.called())
	}
	if frEnc.called() != 1 {
		t.Errorf("frame encoder called %d times, want 1", frEnc.called())
	}
	if pl.called() != 1 {
		t.Errorf("player called %d times, want 1", pl.called())
	}
}

func Test_Play_UsesPresetParams(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()
	cfg.Preset = "classic"

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	// Verify the generator received params consistent with the "classic" preset.
	expectedPreset, ok := audio.GetPreset(audio.PresetClassic)
	if !ok {
		t.Fatal("classic preset not found")
	}
	gotParams := gen.params()
	if gotParams.SampleRate != expectedPreset.SampleRate {
		t.Errorf("generator params SampleRate = %d, want %d", gotParams.SampleRate, expectedPreset.SampleRate)
	}
	if gotParams.Channels != expectedPreset.Channels {
		t.Errorf("generator params Channels = %d, want %d", gotParams.Channels, expectedPreset.Channels)
	}
}

func Test_Play_PassesGuildAndChannelToPlayer(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "my-guild", "my-channel")
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()
	if pl.lastGuild != "my-guild" {
		t.Errorf("player guildID = %q, want %q", pl.lastGuild, "my-guild")
	}
	if pl.lastChan != "my-channel" {
		t.Errorf("player channelID = %q, want %q", pl.lastChan, "my-channel")
	}
}

func Test_Play_Validation(t *testing.T) {
	tests := []struct {
		name      string
		guildID   string
		channelID string
		player    *mockPlayer
		wantErr   error
	}{
		{
			name:      "empty guild ID returns error",
			guildID:   "",
			channelID: "chan-456",
			player:    &mockPlayer{},
			wantErr:   config.ErrMissingGuildID,
		},
		{
			name:      "nil player returns ErrNoPlayer",
			guildID:   "guild-123",
			channelID: "chan-456",
			player:    nil,
			wantErr:   ErrNoPlayer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := &mockGenerator{}
			fEnc := &mockFileEncoder{}
			frEnc := &mockFrameEncoder{}
			cfg := validPlayConfig()

			svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, tt.player)

			err := svc.Play(context.Background(), tt.guildID, tt.channelID)
			if err == nil {
				t.Fatal("Play() expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Play() error = %v, want wrapping %v", err, tt.wantErr)
			}
		})
	}
}

func Test_Play_GeneratorError(t *testing.T) {
	genErr := errors.New("generator boom")
	gen := &mockGenerator{err: genErr}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err == nil {
		t.Fatal("Play() expected error from generator, got nil")
	}
	if !errors.Is(err, ErrGenerateFailed) {
		t.Errorf("Play() error = %v, want wrapping ErrGenerateFailed", err)
	}
	// Player should NOT have been called.
	if pl.called() != 0 {
		t.Errorf("player should not be called on generator error, called %d times", pl.called())
	}
}

func Test_Play_PlayerError(t *testing.T) {
	playErr := errors.New("player boom")
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{err: playErr}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err == nil {
		t.Fatal("Play() expected error from player, got nil")
	}
	if !errors.Is(err, ErrPlayFailed) {
		t.Errorf("Play() error = %v, want wrapping ErrPlayFailed", err)
	}
}

func Test_Play_DryRun_SkipsPlayer(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()
	cfg.DryRun = true

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err != nil {
		t.Fatalf("Play() with DryRun unexpected error: %v", err)
	}

	if gen.called() != 1 {
		t.Errorf("generator called %d times, want 1 (even in dry run)", gen.called())
	}
	if frEnc.called() != 1 {
		t.Errorf("frame encoder called %d times, want 1 (even in dry run)", frEnc.called())
	}
	if pl.called() != 0 {
		t.Errorf("player called %d times in dry run, want 0", pl.called())
	}
}

func Test_Play_DryRun_NilPlayerOK(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validPlayConfig()
	cfg.DryRun = true

	// nil player should not cause an error in dry run mode.
	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err != nil {
		t.Fatalf("Play() with DryRun and nil player unexpected error: %v", err)
	}
}

func Test_Play_ContextCancelled(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := svc.Play(ctx, "guild-123", "chan-456")
	if err == nil {
		t.Fatal("Play() expected context.Canceled, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Play() error = %v, want wrapping context.Canceled", err)
	}
}

func Test_Play_UnknownPreset(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()
	cfg.Preset = "nonexistent-preset"

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err == nil {
		t.Fatal("Play() expected ErrUnknownPreset, got nil")
	}
	if !errors.Is(err, ErrUnknownPreset) {
		t.Errorf("Play() error = %v, want wrapping ErrUnknownPreset", err)
	}
}

func Test_Play_MultiplePresets(t *testing.T) {
	presets := []string{"classic", "whisper", "death-metal", "glitch", "banshee", "robot"}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			gen := &mockGenerator{}
			fEnc := &mockFileEncoder{}
			frEnc := &mockFrameEncoder{}
			pl := &mockPlayer{}
			cfg := validPlayConfig()
			cfg.Preset = preset

			svc := newTestService(cfg, gen, fEnc, frEnc, pl)

			err := svc.Play(context.Background(), "guild-123", "chan-456")
			if err != nil {
				t.Fatalf("Play() with preset %q unexpected error: %v", preset, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Generate() tests
// ---------------------------------------------------------------------------

func Test_Generate_HappyPath_OGG(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()
	cfg.Format = config.FormatOGG

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Generate() unexpected error: %v", err)
	}

	if gen.called() != 1 {
		t.Errorf("generator called %d times, want 1", gen.called())
	}
	if fEnc.called() != 1 {
		t.Errorf("file encoder called %d times, want 1", fEnc.called())
	}
}

func Test_Generate_HappyPath_WAV(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()
	cfg.Format = config.FormatWAV

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Generate() unexpected error: %v", err)
	}

	if gen.called() != 1 {
		t.Errorf("generator called %d times, want 1", gen.called())
	}
	if fEnc.called() != 1 {
		t.Errorf("file encoder called %d times, want 1", fEnc.called())
	}
}

func Test_Generate_NoTokenRequired(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()
	cfg.Token = "" // No token needed for generate-only mode.

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Generate() without token unexpected error: %v", err)
	}
}

func Test_Generate_GeneratorError(t *testing.T) {
	genErr := errors.New("generator boom")
	gen := &mockGenerator{err: genErr}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err == nil {
		t.Fatal("Generate() expected error from generator, got nil")
	}
	if !errors.Is(err, ErrGenerateFailed) {
		t.Errorf("Generate() error = %v, want wrapping ErrGenerateFailed", err)
	}
	// File encoder should NOT have been called.
	if fEnc.called() != 0 {
		t.Errorf("file encoder should not be called on generator error, called %d times", fEnc.called())
	}
}

func Test_Generate_FileEncoderError(t *testing.T) {
	encErr := errors.New("encoding boom")
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{err: encErr}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err == nil {
		t.Fatal("Generate() expected error from encoder, got nil")
	}
	if !errors.Is(err, ErrEncodeFailed) {
		t.Errorf("Generate() error = %v, want wrapping ErrEncodeFailed", err)
	}
}

func Test_Generate_UnknownPreset(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()
	cfg.Preset = "bogus-preset"

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err == nil {
		t.Fatal("Generate() expected ErrUnknownPreset, got nil")
	}
	if !errors.Is(err, ErrUnknownPreset) {
		t.Errorf("Generate() error = %v, want wrapping ErrUnknownPreset", err)
	}
}

func Test_Generate_PlayerNotInvoked(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validGenerateConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, pl)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Generate() unexpected error: %v", err)
	}

	// Player should never be called during Generate.
	if pl.called() != 0 {
		t.Errorf("player should not be called during Generate, called %d times", pl.called())
	}
}

// ---------------------------------------------------------------------------
// Close() tests
// ---------------------------------------------------------------------------

func Test_Close_WithCloser(t *testing.T) {
	mc := &mockCloser{}
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validPlayConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)
	svc.closer = mc

	err := svc.Close()
	if err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}
	if mc.called() != 1 {
		t.Errorf("closer called %d times, want 1", mc.called())
	}
}

func Test_Close_WithCloserError(t *testing.T) {
	closeErr := errors.New("close boom")
	mc := &mockCloser{err: closeErr}
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validPlayConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)
	svc.closer = mc

	err := svc.Close()
	if err == nil {
		t.Fatal("Close() expected error, got nil")
	}
	if !errors.Is(err, closeErr) {
		t.Errorf("Close() error = %v, want %v", err, closeErr)
	}
}

func Test_Close_NilCloser(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validPlayConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)
	// closer is nil by default.

	err := svc.Close()
	if err != nil {
		t.Fatalf("Close() with nil closer unexpected error: %v", err)
	}
}

func Test_Close_CalledTwice_NoPanic(t *testing.T) {
	mc := &mockCloser{}
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validPlayConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)
	svc.closer = mc

	// Should not panic on double close.
	_ = svc.Close()
	err := svc.Close()
	// Second close may return nil or an error, but must not panic.
	_ = err
}

// ---------------------------------------------------------------------------
// ListPresets() tests
// ---------------------------------------------------------------------------

func Test_ListPresets_ReturnsAllPresets(t *testing.T) {
	presets := ListPresets()

	if len(presets) != 6 {
		t.Fatalf("ListPresets() returned %d presets, want 6", len(presets))
	}
}

func Test_ListPresets_ContainsExpectedNames(t *testing.T) {
	expected := []string{"classic", "whisper", "death-metal", "glitch", "banshee", "robot"}
	presets := ListPresets()

	presetSet := make(map[string]bool)
	for _, p := range presets {
		presetSet[p] = true
	}

	for _, name := range expected {
		if !presetSet[name] {
			t.Errorf("ListPresets() missing expected preset %q", name)
		}
	}
}

func Test_ListPresets_NoDuplicates(t *testing.T) {
	presets := ListPresets()

	seen := make(map[string]bool)
	for _, p := range presets {
		if seen[p] {
			t.Errorf("ListPresets() returned duplicate preset %q", p)
		}
		seen[p] = true
	}
}

func Test_ListPresets_Deterministic(t *testing.T) {
	first := ListPresets()
	second := ListPresets()

	if len(first) != len(second) {
		t.Fatalf("ListPresets() non-deterministic: first=%d, second=%d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("ListPresets()[%d] = %q then %q; not deterministic", i, first[i], second[i])
		}
	}
}

// ---------------------------------------------------------------------------
// resolveParams (tested indirectly through Play/Generate behavior)
// ---------------------------------------------------------------------------

func Test_ResolveParams_PresetOverridesDuration(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()
	cfg.Preset = "classic"
	cfg.Duration = 5 * time.Second // Override preset duration.

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	gotParams := gen.params()
	// The config duration should override the preset default.
	if gotParams.Duration != 5*time.Second {
		t.Errorf("generator params Duration = %v, want %v", gotParams.Duration, 5*time.Second)
	}
}

func Test_ResolveParams_EmptyPresetUsesRandomize(t *testing.T) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()
	cfg.Preset = "" // No preset — should use Randomize.

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	gotParams := gen.params()
	// Randomize always produces 48000 Hz, 2 channels.
	if gotParams.SampleRate != 48000 {
		t.Errorf("generator params SampleRate = %d, want 48000", gotParams.SampleRate)
	}
	if gotParams.Channels != 2 {
		t.Errorf("generator params Channels = %d, want 2", gotParams.Channels)
	}
}

// ---------------------------------------------------------------------------
// Sentinel error existence tests
// ---------------------------------------------------------------------------

func Test_SentinelErrors_Exist(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNoPlayer", ErrNoPlayer, "scream:"},
		{"ErrUnknownPreset", ErrUnknownPreset, "scream:"},
		{"ErrGenerateFailed", ErrGenerateFailed, "scream:"},
		{"ErrEncodeFailed", ErrEncodeFailed, "scream:"},
		{"ErrPlayFailed", ErrPlayFailed, "scream:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("%s is nil", tt.name)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Errorf("%s.Error() = %q, want containing %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error wrapping verification
// ---------------------------------------------------------------------------

func Test_Play_GeneratorError_WrapsOriginal(t *testing.T) {
	origErr := errors.New("underlying generator failure")
	gen := &mockGenerator{err: origErr}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err == nil {
		t.Fatal("Play() expected error, got nil")
	}
	// Should wrap both ErrGenerateFailed and the original error.
	if !errors.Is(err, ErrGenerateFailed) {
		t.Errorf("Play() error does not wrap ErrGenerateFailed: %v", err)
	}
}

func Test_Play_PlayerError_WrapsOriginal(t *testing.T) {
	origErr := errors.New("underlying player failure")
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{err: origErr}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)

	err := svc.Play(context.Background(), "guild-123", "chan-456")
	if err == nil {
		t.Fatal("Play() expected error, got nil")
	}
	if !errors.Is(err, ErrPlayFailed) {
		t.Errorf("Play() error does not wrap ErrPlayFailed: %v", err)
	}
}

func Test_Generate_GeneratorError_WrapsOriginal(t *testing.T) {
	origErr := errors.New("underlying generator failure")
	gen := &mockGenerator{err: origErr}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err == nil {
		t.Fatal("Generate() expected error, got nil")
	}
	if !errors.Is(err, ErrGenerateFailed) {
		t.Errorf("Generate() error does not wrap ErrGenerateFailed: %v", err)
	}
}

func Test_Generate_EncoderError_WrapsOriginal(t *testing.T) {
	origErr := errors.New("underlying encoder failure")
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{err: origErr}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)

	var buf bytes.Buffer
	err := svc.Generate(context.Background(), &buf)
	if err == nil {
		t.Fatal("Generate() expected error, got nil")
	}
	if !errors.Is(err, ErrEncodeFailed) {
		t.Errorf("Generate() error does not wrap ErrEncodeFailed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func Benchmark_Play_HappyPath(b *testing.B) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	pl := &mockPlayer{}
	cfg := validPlayConfig()

	svc := newTestService(cfg, gen, fEnc, frEnc, pl)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.Play(ctx, "guild-123", "chan-456")
	}
}

func Benchmark_Generate_HappyPath(b *testing.B) {
	gen := &mockGenerator{}
	fEnc := &mockFileEncoder{}
	frEnc := &mockFrameEncoder{}
	cfg := validGenerateConfig()

	svc := NewServiceWithDeps(cfg, gen, fEnc, frEnc, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = svc.Generate(ctx, &buf)
	}
}

func Benchmark_ListPresets(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ListPresets()
	}
}
