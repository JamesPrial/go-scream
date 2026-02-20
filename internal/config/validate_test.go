package config

import (
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Validate()
// ---------------------------------------------------------------------------

func TestValidate_DefaultConfigPasses(t *testing.T) {
	cfg := Default()
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate(Default()) unexpected error: %v", err)
	}
}

func TestValidate_Backend(t *testing.T) {
	tests := []struct {
		name    string
		backend BackendType
		wantErr error
	}{
		{
			name:    "native is valid",
			backend: BackendNative,
			wantErr: nil,
		},
		{
			name:    "ffmpeg is valid",
			backend: BackendFFmpeg,
			wantErr: nil,
		},
		{
			name:    "empty is invalid",
			backend: "",
			wantErr: ErrInvalidBackend,
		},
		{
			name:    "unknown backend is invalid",
			backend: "pulse",
			wantErr: ErrInvalidBackend,
		},
		{
			name:    "case sensitive: Native is invalid",
			backend: "Native",
			wantErr: ErrInvalidBackend,
		},
		{
			name:    "case sensitive: FFMPEG is invalid",
			backend: "FFMPEG",
			wantErr: ErrInvalidBackend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Backend = tt.backend
			err := Validate(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidate_Preset(t *testing.T) {
	tests := []struct {
		name    string
		preset  string
		wantErr error
	}{
		{
			name:    "classic is valid",
			preset:  "classic",
			wantErr: nil,
		},
		{
			name:    "whisper is valid",
			preset:  "whisper",
			wantErr: nil,
		},
		{
			name:    "death-metal is valid",
			preset:  "death-metal",
			wantErr: nil,
		},
		{
			name:    "glitch is valid",
			preset:  "glitch",
			wantErr: nil,
		},
		{
			name:    "banshee is valid",
			preset:  "banshee",
			wantErr: nil,
		},
		{
			name:    "robot is valid",
			preset:  "robot",
			wantErr: nil,
		},
		{
			name:    "empty is valid (no preset selected)",
			preset:  "",
			wantErr: nil,
		},
		{
			name:    "unknown preset is invalid",
			preset:  "dubstep",
			wantErr: ErrInvalidPreset,
		},
		{
			name:    "case sensitive: Classic is invalid",
			preset:  "Classic",
			wantErr: ErrInvalidPreset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Preset = tt.preset
			err := Validate(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidate_Duration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantErr  error
	}{
		{
			name:     "positive duration is valid",
			duration: 3 * time.Second,
			wantErr:  nil,
		},
		{
			name:     "1ms is valid",
			duration: time.Millisecond,
			wantErr:  nil,
		},
		{
			name:     "zero duration is invalid",
			duration: 0,
			wantErr:  ErrInvalidDuration,
		},
		{
			name:     "negative duration is invalid",
			duration: -1 * time.Second,
			wantErr:  ErrInvalidDuration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Duration = tt.duration
			err := Validate(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidate_Volume(t *testing.T) {
	tests := []struct {
		name    string
		volume  float64
		wantErr error
	}{
		{
			name:    "volume 1.0 is valid",
			volume:  1.0,
			wantErr: nil,
		},
		{
			name:    "volume 0.0 is valid",
			volume:  0.0,
			wantErr: nil,
		},
		{
			name:    "volume 0.5 is valid",
			volume:  0.5,
			wantErr: nil,
		},
		{
			name:    "volume above 1.0 is invalid",
			volume:  1.01,
			wantErr: ErrInvalidVolume,
		},
		{
			name:    "volume below 0.0 is invalid",
			volume:  -0.1,
			wantErr: ErrInvalidVolume,
		},
		{
			name:    "volume way above range",
			volume:  5.0,
			wantErr: ErrInvalidVolume,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Volume = tt.volume
			err := Validate(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidate_Format(t *testing.T) {
	tests := []struct {
		name    string
		format  FormatType
		wantErr error
	}{
		{
			name:    "ogg is valid",
			format:  FormatOGG,
			wantErr: nil,
		},
		{
			name:    "wav is valid",
			format:  FormatWAV,
			wantErr: nil,
		},
		{
			name:    "empty format is invalid",
			format:  "",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "mp3 is invalid",
			format:  "mp3",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "case sensitive: OGG is invalid",
			format:  "OGG",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "case sensitive: WAV is invalid",
			format:  "WAV",
			wantErr: ErrInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Format = tt.format
			err := Validate(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Validate() — LogLevel field
// ---------------------------------------------------------------------------

func TestValidate_LogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		wantErr  error
	}{
		{
			name:     "empty is valid (no log level set)",
			logLevel: "",
			wantErr:  nil,
		},
		{
			name:     "debug is valid",
			logLevel: "debug",
			wantErr:  nil,
		},
		{
			name:     "info is valid",
			logLevel: "info",
			wantErr:  nil,
		},
		{
			name:     "warn is valid",
			logLevel: "warn",
			wantErr:  nil,
		},
		{
			name:     "error is valid",
			logLevel: "error",
			wantErr:  nil,
		},
		{
			name:     "INFO is valid (case-insensitive)",
			logLevel: "INFO",
			wantErr:  nil,
		},
		{
			name:     "Debug is valid (mixed case)",
			logLevel: "Debug",
			wantErr:  nil,
		},
		{
			name:     "WARN is valid (uppercase)",
			logLevel: "WARN",
			wantErr:  nil,
		},
		{
			name:     "ERROR is valid (uppercase)",
			logLevel: "ERROR",
			wantErr:  nil,
		},
		{
			name:     "trace is invalid",
			logLevel: "trace",
			wantErr:  ErrInvalidLogLevel,
		},
		{
			name:     "invalid is invalid",
			logLevel: "invalid",
			wantErr:  ErrInvalidLogLevel,
		},
		{
			name:     "fatal is invalid",
			logLevel: "fatal",
			wantErr:  ErrInvalidLogLevel,
		},
		{
			name:     "verbose is invalid (not a log level)",
			logLevel: "verbose",
			wantErr:  ErrInvalidLogLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.LogLevel = tt.logLevel
			err := Validate(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidate_MultipleInvalidFields(t *testing.T) {
	// When multiple fields are invalid, Validate should return an error.
	// We do not prescribe which error it returns first; we just confirm
	// it is one of the expected sentinel errors.
	cfg := Config{
		Backend:  "invalid-backend",
		Preset:   "invalid-preset",
		Duration: -1 * time.Second,
		Volume:   -1.0,
		Format:   "mp3",
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() expected error for fully invalid config, got nil")
	}

	// The error should match at least one of the expected sentinels.
	isExpected := errors.Is(err, ErrInvalidBackend) ||
		errors.Is(err, ErrInvalidPreset) ||
		errors.Is(err, ErrInvalidDuration) ||
		errors.Is(err, ErrInvalidVolume) ||
		errors.Is(err, ErrInvalidFormat)

	if !isExpected {
		t.Errorf("Validate() error = %v, want one of the known sentinel errors", err)
	}
}

func TestValidate_SentinelErrorsExist(t *testing.T) {
	// Verify that all sentinel errors are non-nil distinct values.
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrConfigNotFound", ErrConfigNotFound},
		{"ErrConfigParse", ErrConfigParse},
		{"ErrInvalidBackend", ErrInvalidBackend},
		{"ErrInvalidPreset", ErrInvalidPreset},
		{"ErrInvalidDuration", ErrInvalidDuration},
		{"ErrInvalidVolume", ErrInvalidVolume},
		{"ErrInvalidFormat", ErrInvalidFormat},
		{"ErrInvalidLogLevel", ErrInvalidLogLevel},
	}

	for _, s := range sentinels {
		t.Run(s.name, func(t *testing.T) {
			if s.err == nil {
				t.Errorf("%s is nil", s.name)
			}
			if s.err.Error() == "" {
				t.Errorf("%s.Error() is empty", s.name)
			}
		})
	}
}

func TestValidate_ErrorWrapping(t *testing.T) {
	// Verify that Load() errors wrap the sentinel properly (testable with errors.Is).
	// We do not test the exact message, just that wrapping is correct.

	// ErrConfigNotFound: tested in load_test.go
	// ErrConfigParse: tested in load_test.go

	// Validate errors: tested above via errors.Is.
	// This test just double-checks that the sentinel errors are usable with errors.Is.
	cfg := Default()
	cfg.Backend = "bogus"
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidBackend) {
		t.Errorf("errors.Is(err, ErrInvalidBackend) = false, want true; err = %v", err)
	}
}

func TestValidate_LogLevel_ErrorWrapping(t *testing.T) {
	// Verify that ErrInvalidLogLevel is returned and detectable via errors.Is.
	cfg := Default()
	cfg.LogLevel = "trace"
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !errors.Is(err, ErrInvalidLogLevel) {
		t.Errorf("errors.Is(err, ErrInvalidLogLevel) = false, want true; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkValidate_ValidConfig(b *testing.B) {
	cfg := Default()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Validate(cfg)
	}
}

func BenchmarkDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Default()
	}
}

func BenchmarkMerge(b *testing.B) {
	base := Default()
	overlay := Config{
		Token:   "tok",
		Backend: BackendFFmpeg,
		Volume:  0.5,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Merge(base, overlay)
	}
}
