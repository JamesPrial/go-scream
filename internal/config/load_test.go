package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Load()
// ---------------------------------------------------------------------------

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `token: "my-token"
guild_id: "guild-999"
backend: "ffmpeg"
preset: "banshee"
duration: 5s
volume: 0.75
output_file: "out.ogg"
format: "wav"
dry_run: true
verbose: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Token != "my-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "my-token")
	}
	if cfg.GuildID != "guild-999" {
		t.Errorf("GuildID = %q, want %q", cfg.GuildID, "guild-999")
	}
	if cfg.Backend != BackendFFmpeg {
		t.Errorf("Backend = %q, want %q", cfg.Backend, BackendFFmpeg)
	}
	if cfg.Preset != "banshee" {
		t.Errorf("Preset = %q, want %q", cfg.Preset, "banshee")
	}
	if cfg.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want %v", cfg.Duration, 5*time.Second)
	}
	if cfg.Volume != 0.75 {
		t.Errorf("Volume = %f, want %f", cfg.Volume, 0.75)
	}
	if cfg.OutputFile != "out.ogg" {
		t.Errorf("OutputFile = %q, want %q", cfg.OutputFile, "out.ogg")
	}
	if cfg.Format != FormatWAV {
		t.Errorf("Format = %q, want %q", cfg.Format, FormatWAV)
	}
	if cfg.DryRun != true {
		t.Errorf("DryRun = %v, want true", cfg.DryRun)
	}
	if cfg.Verbose != true {
		t.Errorf("Verbose = %v, want true", cfg.Verbose)
	}
}

func TestLoad_PartialYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `token: "partial-token"
backend: "native"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Token != "partial-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "partial-token")
	}
	if cfg.Backend != BackendNative {
		t.Errorf("Backend = %q, want %q", cfg.Backend, BackendNative)
	}

	// Unset fields should be zero values.
	if cfg.GuildID != "" {
		t.Errorf("GuildID = %q, want empty", cfg.GuildID)
	}
	if cfg.Duration != 0 {
		t.Errorf("Duration = %v, want 0", cfg.Duration)
	}
	if cfg.Volume != 0 {
		t.Errorf("Volume = %f, want 0", cfg.Volume)
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/to/config.yaml")
	if err == nil {
		t.Fatal("Load() expected error for nonexistent file, got nil")
	}
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("Load() error = %v, want wrapping ErrConfigNotFound", err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// Invalid YAML: tab indentation with mismatched structure.
	content := `:::not valid yaml at all [[[{{{
token: [unclosed bracket
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
	if !errors.Is(err, ErrConfigParse) {
		t.Errorf("Load() error = %v, want wrapping ErrConfigParse", err)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error for empty file: %v", err)
	}

	// Empty YAML produces zero-value Config.
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
	if cfg.Backend != "" {
		t.Errorf("Backend = %q, want empty", cfg.Backend)
	}
	if cfg.Duration != 0 {
		t.Errorf("Duration = %v, want 0", cfg.Duration)
	}
}

func TestLoad_UnknownFieldsSilentlyIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extra.yaml")
	content := `token: "my-token"
unknown_field: "should be ignored"
another_bogus: 42
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error for unknown fields: %v", err)
	}
	if cfg.Token != "my-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "my-token")
	}
}

func TestLoad_DurationFormats(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantDur time.Duration
		wantErr bool
	}{
		{
			name:    "Go duration string 3s",
			yaml:    `duration: 3s`,
			wantDur: 3 * time.Second,
		},
		{
			name:    "Go duration string 500ms",
			yaml:    `duration: 500ms`,
			wantDur: 500 * time.Millisecond,
		},
		{
			name:    "Go duration string 1m30s",
			yaml:    `duration: 1m30s`,
			wantDur: 90 * time.Second,
		},
		{
			name:    "invalid duration string",
			yaml:    `duration: notaduration`,
			wantErr: true,
		},
		{
			name:    "bare integer without unit",
			yaml:    `duration: "5"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "dur.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Load() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if cfg.Duration != tt.wantDur {
				t.Errorf("Duration = %v, want %v", cfg.Duration, tt.wantDur)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ApplyEnv()
// ---------------------------------------------------------------------------

func TestApplyEnv_AllVariables(t *testing.T) {
	cfg := Config{}

	t.Setenv("DISCORD_TOKEN", "env-token")
	t.Setenv("SCREAM_GUILD_ID", "env-guild")
	t.Setenv("SCREAM_BACKEND", "ffmpeg")
	t.Setenv("SCREAM_PRESET", "death-metal")
	t.Setenv("SCREAM_DURATION", "7s")
	t.Setenv("SCREAM_VOLUME", "0.42")
	t.Setenv("SCREAM_FORMAT", "wav")
	t.Setenv("SCREAM_VERBOSE", "true")

	ApplyEnv(&cfg)

	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "env-token")
	}
	if cfg.GuildID != "env-guild" {
		t.Errorf("GuildID = %q, want %q", cfg.GuildID, "env-guild")
	}
	if cfg.Backend != BackendFFmpeg {
		t.Errorf("Backend = %q, want %q", cfg.Backend, BackendFFmpeg)
	}
	if cfg.Preset != "death-metal" {
		t.Errorf("Preset = %q, want %q", cfg.Preset, "death-metal")
	}
	if cfg.Duration != 7*time.Second {
		t.Errorf("Duration = %v, want %v", cfg.Duration, 7*time.Second)
	}
	if cfg.Volume != 0.42 {
		t.Errorf("Volume = %f, want %f", cfg.Volume, 0.42)
	}
	if cfg.Format != FormatWAV {
		t.Errorf("Format = %q, want %q", cfg.Format, FormatWAV)
	}
	if cfg.Verbose != true {
		t.Errorf("Verbose = %v, want true", cfg.Verbose)
	}
}

func TestApplyEnv_EmptyEnvVarsUnset(t *testing.T) {
	cfg := Config{
		Token:   "original-token",
		GuildID: "original-guild",
		Backend: BackendNative,
		Volume:  0.9,
	}

	// Set env vars to empty strings - these should be treated as unset.
	t.Setenv("DISCORD_TOKEN", "")
	t.Setenv("SCREAM_GUILD_ID", "")
	t.Setenv("SCREAM_BACKEND", "")
	t.Setenv("SCREAM_PRESET", "")
	t.Setenv("SCREAM_DURATION", "")
	t.Setenv("SCREAM_VOLUME", "")
	t.Setenv("SCREAM_FORMAT", "")
	t.Setenv("SCREAM_VERBOSE", "")

	ApplyEnv(&cfg)

	// Original values should be preserved.
	if cfg.Token != "original-token" {
		t.Errorf("Token = %q, want %q (empty env should not override)", cfg.Token, "original-token")
	}
	if cfg.GuildID != "original-guild" {
		t.Errorf("GuildID = %q, want %q", cfg.GuildID, "original-guild")
	}
	if cfg.Backend != BackendNative {
		t.Errorf("Backend = %q, want %q", cfg.Backend, BackendNative)
	}
	if cfg.Volume != 0.9 {
		t.Errorf("Volume = %f, want %f", cfg.Volume, 0.9)
	}
}

func TestApplyEnv_InvalidDurationSilentlyIgnored(t *testing.T) {
	cfg := Config{Duration: 3 * time.Second}
	t.Setenv("SCREAM_DURATION", "not-a-duration")

	ApplyEnv(&cfg)

	if cfg.Duration != 3*time.Second {
		t.Errorf("Duration = %v, want %v (invalid value should be silently ignored)", cfg.Duration, 3*time.Second)
	}
}

func TestApplyEnv_InvalidVolumeSilentlyIgnored(t *testing.T) {
	cfg := Config{Volume: 0.8}
	t.Setenv("SCREAM_VOLUME", "not-a-number")

	ApplyEnv(&cfg)

	if cfg.Volume != 0.8 {
		t.Errorf("Volume = %f, want %f (invalid value should be silently ignored)", cfg.Volume, 0.8)
	}
}

func TestApplyEnv_InvalidVerboseSilentlyIgnored(t *testing.T) {
	cfg := Config{Verbose: false}
	t.Setenv("SCREAM_VERBOSE", "not-a-bool")

	ApplyEnv(&cfg)

	if cfg.Verbose != false {
		t.Errorf("Verbose = %v, want false (invalid value should be silently ignored)", cfg.Verbose)
	}
}

func TestApplyEnv_OverridesExistingValues(t *testing.T) {
	cfg := Config{
		Token:   "file-token",
		Backend: BackendNative,
		Volume:  1.0,
	}

	t.Setenv("DISCORD_TOKEN", "env-token")
	t.Setenv("SCREAM_BACKEND", "ffmpeg")
	t.Setenv("SCREAM_VOLUME", "0.5")

	ApplyEnv(&cfg)

	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "env-token")
	}
	if cfg.Backend != BackendFFmpeg {
		t.Errorf("Backend = %q, want %q", cfg.Backend, BackendFFmpeg)
	}
	if cfg.Volume != 0.5 {
		t.Errorf("Volume = %f, want %f", cfg.Volume, 0.5)
	}
}

func TestApplyEnv_IndividualVariables(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		envVal  string
		initial Config
		check   func(t *testing.T, cfg Config)
	}{
		{
			name:    "DISCORD_TOKEN",
			envKey:  "DISCORD_TOKEN",
			envVal:  "tok-123",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Token != "tok-123" {
					t.Errorf("Token = %q, want %q", cfg.Token, "tok-123")
				}
			},
		},
		{
			name:    "SCREAM_GUILD_ID",
			envKey:  "SCREAM_GUILD_ID",
			envVal:  "g-456",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.GuildID != "g-456" {
					t.Errorf("GuildID = %q, want %q", cfg.GuildID, "g-456")
				}
			},
		},
		{
			name:    "SCREAM_BACKEND native",
			envKey:  "SCREAM_BACKEND",
			envVal:  "native",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Backend != BackendNative {
					t.Errorf("Backend = %q, want %q", cfg.Backend, BackendNative)
				}
			},
		},
		{
			name:    "SCREAM_PRESET",
			envKey:  "SCREAM_PRESET",
			envVal:  "robot",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Preset != "robot" {
					t.Errorf("Preset = %q, want %q", cfg.Preset, "robot")
				}
			},
		},
		{
			name:    "SCREAM_DURATION",
			envKey:  "SCREAM_DURATION",
			envVal:  "10s",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Duration != 10*time.Second {
					t.Errorf("Duration = %v, want %v", cfg.Duration, 10*time.Second)
				}
			},
		},
		{
			name:    "SCREAM_VOLUME",
			envKey:  "SCREAM_VOLUME",
			envVal:  "0.33",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Volume != 0.33 {
					t.Errorf("Volume = %f, want %f", cfg.Volume, 0.33)
				}
			},
		},
		{
			name:    "SCREAM_FORMAT ogg",
			envKey:  "SCREAM_FORMAT",
			envVal:  "ogg",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Format != FormatOGG {
					t.Errorf("Format = %q, want %q", cfg.Format, FormatOGG)
				}
			},
		},
		{
			name:    "SCREAM_VERBOSE true",
			envKey:  "SCREAM_VERBOSE",
			envVal:  "true",
			initial: Config{},
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Verbose != true {
					t.Errorf("Verbose = %v, want true", cfg.Verbose)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.initial
			t.Setenv(tt.envKey, tt.envVal)
			ApplyEnv(&cfg)
			tt.check(t, cfg)
		})
	}
}

func TestApplyEnv_VerboseVariants(t *testing.T) {
	// Test various truthy/falsy strings for SCREAM_VERBOSE.
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"TRUE", "TRUE", true},
		{"True", "True", true},
		{"false", "false", false},
		{"0", "0", false},
		{"FALSE", "FALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{}
			t.Setenv("SCREAM_VERBOSE", tt.val)
			ApplyEnv(&cfg)
			if cfg.Verbose != tt.want {
				t.Errorf("Verbose = %v, want %v for SCREAM_VERBOSE=%q", cfg.Verbose, tt.want, tt.val)
			}
		})
	}
}
