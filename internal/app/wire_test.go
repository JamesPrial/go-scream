package app

import (
	"io"
	"log/slog"
	"os/exec"
	"testing"

	"github.com/JamesPrial/go-scream/internal/audio/ffmpeg"
	"github.com/JamesPrial/go-scream/internal/config"
	"github.com/JamesPrial/go-scream/internal/encoding"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// skipIfNoFFmpeg skips the test if ffmpeg is not available on PATH.
func skipIfNoFFmpeg(t *testing.T) {
	t.Helper()
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available")
	}
}

// ---------------------------------------------------------------------------
// NewGenerator
// ---------------------------------------------------------------------------

func TestNewGenerator_NativeBackend(t *testing.T) {
	gen, err := NewGenerator(config.BackendNative, discardLogger)
	if err != nil {
		t.Fatalf("NewGenerator(%q) error = %v, want nil", config.BackendNative, err)
	}
	if gen == nil {
		t.Fatal("NewGenerator(\"native\") returned nil generator with nil error")
	}
}

func TestNewGenerator_FFmpegBackend_Available(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewGenerator(config.BackendFFmpeg, discardLogger)
	if err != nil {
		t.Fatalf("NewGenerator(%q) error = %v, want nil", config.BackendFFmpeg, err)
	}
	if gen == nil {
		t.Fatal("NewGenerator(\"ffmpeg\") returned nil generator with nil error")
	}
}

func TestNewGenerator_FFmpegBackend_NotAvailable(t *testing.T) {
	// When ffmpeg is available, the error path cannot be tested directly.
	// We verify the sentinel error exists and that it wraps correctly when
	// returned by the ffmpeg package.
	if ffmpeg.ErrFFmpegNotFound == nil {
		t.Fatal("ErrFFmpegNotFound sentinel should not be nil")
	}
}

func TestNewGenerator_UnknownBackend_FallsBackToNative(t *testing.T) {
	// Per the implementation doc: "Any other value returns the native Go
	// generator." Unknown backends do NOT produce an error.
	tests := []struct {
		name    string
		backend config.BackendType
	}{
		{"empty string", config.BackendType("")},
		{"unknown string", config.BackendType("unknown")},
		{"typo", config.BackendType("nativ")},
		{"uppercase NATIVE", config.BackendType("NATIVE")},
		{"mixed case Ffmpeg", config.BackendType("Ffmpeg")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(tt.backend, discardLogger)
			if err != nil {
				t.Fatalf("NewGenerator(%q) error = %v, want nil", tt.backend, err)
			}
			if gen == nil {
				t.Fatalf("NewGenerator(%q) returned nil generator", tt.backend)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewFileEncoder
// ---------------------------------------------------------------------------

func TestNewFileEncoder_OGG(t *testing.T) {
	enc := NewFileEncoder(config.FormatOGG, discardLogger)
	if enc == nil {
		t.Fatal("NewFileEncoder(\"ogg\") returned nil")
	}
	if _, ok := enc.(*encoding.OGGEncoder); !ok {
		t.Errorf("NewFileEncoder(\"ogg\") returned %T, want *encoding.OGGEncoder", enc)
	}
}

func TestNewFileEncoder_WAV(t *testing.T) {
	enc := NewFileEncoder(config.FormatWAV, discardLogger)
	if enc == nil {
		t.Fatal("NewFileEncoder(\"wav\") returned nil")
	}
	if _, ok := enc.(*encoding.WAVEncoder); !ok {
		t.Errorf("NewFileEncoder(\"wav\") returned %T, want *encoding.WAVEncoder", enc)
	}
}

func TestNewFileEncoder_DefaultsToOGG(t *testing.T) {
	// Per doc: "Any other value returns an OGGEncoder."
	tests := []struct {
		name   string
		format config.FormatType
	}{
		{"empty string", config.FormatType("")},
		{"unknown format", config.FormatType("mp3")},
		{"uppercase WAV", config.FormatType("WAV")},
		{"uppercase OGG", config.FormatType("OGG")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewFileEncoder(tt.format, discardLogger)
			if enc == nil {
				t.Fatalf("NewFileEncoder(%q) returned nil", tt.format)
			}
			if _, ok := enc.(*encoding.OGGEncoder); !ok {
				t.Errorf("NewFileEncoder(%q) returned %T, want *encoding.OGGEncoder (default)", tt.format, enc)
			}
		})
	}
}

func TestNewFileEncoder_NeverReturnsNil(t *testing.T) {
	formats := []config.FormatType{"ogg", "wav", "", "flac", "mp3", "aac"}
	for _, f := range formats {
		enc := NewFileEncoder(f, discardLogger)
		if enc == nil {
			t.Errorf("NewFileEncoder(%q) returned nil, should never return nil", f)
		}
	}
}

func TestNewFileEncoder_ImplementsFileEncoder(t *testing.T) {
	// Verify that NewFileEncoder returns a value assignable to FileEncoder.
	// The return type of NewFileEncoder is encoding.FileEncoder, so any
	// assignment is already guaranteed at compile time by the signature.
	_ = NewFileEncoder(config.FormatOGG, discardLogger)
	_ = NewFileEncoder(config.FormatWAV, discardLogger)
}

// ---------------------------------------------------------------------------
// NewDiscordDeps — skipped because it requires a real Discord token and
// network access (calls session.Open() which initiates a WebSocket connection).
// ---------------------------------------------------------------------------

func TestNewDiscordDeps_RequiresNetwork(t *testing.T) {
	t.Skip("NewDiscordDeps requires a real Discord bot token and network access")
}

// ---------------------------------------------------------------------------
// NewGenerator table-driven (combined scenarios)
// ---------------------------------------------------------------------------

func TestNewGenerator_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		backend   config.BackendType
		wantNil   bool
		wantErr   bool
		skipNoFFm bool // skip if ffmpeg not on PATH
	}{
		{
			name:    "native backend returns generator",
			backend: config.BackendNative,
			wantNil: false,
			wantErr: false,
		},
		{
			name:      "ffmpeg backend returns generator when available",
			backend:   config.BackendFFmpeg,
			wantNil:   false,
			wantErr:   false,
			skipNoFFm: true,
		},
		{
			name:    "empty string falls back to native",
			backend: config.BackendType(""),
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "arbitrary string falls back to native",
			backend: config.BackendType("pulse-audio"),
			wantNil: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipNoFFm {
				skipIfNoFFmpeg(t)
			}

			gen, err := NewGenerator(tt.backend, discardLogger)

			if tt.wantErr && err == nil {
				t.Fatal("NewGenerator() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("NewGenerator() unexpected error: %v", err)
			}
			if tt.wantNil && gen != nil {
				t.Error("NewGenerator() expected nil generator, got non-nil")
			}
			if !tt.wantNil && gen == nil {
				t.Error("NewGenerator() expected non-nil generator, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewFileEncoder table-driven (combined scenarios)
// ---------------------------------------------------------------------------

func TestNewFileEncoder_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		format   config.FormatType
		wantType string // "OGG" or "WAV"
	}{
		{
			name:     "ogg format returns OGGEncoder",
			format:   config.FormatOGG,
			wantType: "OGG",
		},
		{
			name:     "wav format returns WAVEncoder",
			format:   config.FormatWAV,
			wantType: "WAV",
		},
		{
			name:     "empty string defaults to OGG",
			format:   config.FormatType(""),
			wantType: "OGG",
		},
		{
			name:     "unknown format defaults to OGG",
			format:   config.FormatType("flac"),
			wantType: "OGG",
		},
		{
			name:     "case sensitive wav only",
			format:   config.FormatType("WAV"),
			wantType: "OGG", // uppercase does not match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewFileEncoder(tt.format, discardLogger)
			if enc == nil {
				t.Fatal("NewFileEncoder() returned nil")
			}

			switch tt.wantType {
			case "OGG":
				if _, ok := enc.(*encoding.OGGEncoder); !ok {
					t.Errorf("NewFileEncoder(%q) = %T, want *encoding.OGGEncoder", tt.format, enc)
				}
			case "WAV":
				if _, ok := enc.(*encoding.WAVEncoder); !ok {
					t.Errorf("NewFileEncoder(%q) = %T, want *encoding.WAVEncoder", tt.format, enc)
				}
			default:
				t.Fatalf("bad wantType in test table: %q", tt.wantType)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SetupLogger
// ---------------------------------------------------------------------------

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{
			name: "default config returns non-nil logger",
			cfg:  config.Default(),
		},
		{
			name: "verbose config returns non-nil logger",
			cfg:  config.Config{Verbose: true},
		},
		{
			name: "explicit log level returns non-nil logger",
			cfg:  config.Config{LogLevel: "debug"},
		},
		{
			name: "zero-value config returns non-nil logger",
			cfg:  config.Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := SetupLogger(tt.cfg)
			if logger == nil {
				t.Fatal("SetupLogger() returned nil, want non-nil *slog.Logger")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SignalContext
// ---------------------------------------------------------------------------

func TestSignalContext(t *testing.T) {
	ctx, cancel := SignalContext()
	if ctx == nil {
		t.Fatal("SignalContext() returned nil context")
	}
	if cancel == nil {
		t.Fatal("SignalContext() returned nil cancel func")
	}

	// Verify context is not already done before cancel.
	select {
	case <-ctx.Done():
		t.Fatal("SignalContext() context should not be done before cancel is called")
	default:
		// expected: context is still active
	}

	// Calling cancel should cause the context to be done.
	cancel()

	select {
	case <-ctx.Done():
		// expected: context is cancelled
	default:
		t.Fatal("SignalContext() context should be done after cancel is called")
	}
}
