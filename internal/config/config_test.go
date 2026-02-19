package config

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Default()
// ---------------------------------------------------------------------------

func TestDefault_ReturnsExpectedValues(t *testing.T) {
	cfg := Default()

	// String fields default to empty.
	if cfg.Token != "" {
		t.Errorf("Default().Token = %q, want %q", cfg.Token, "")
	}
	if cfg.GuildID != "" {
		t.Errorf("Default().GuildID = %q, want %q", cfg.GuildID, "")
	}
	if cfg.OutputFile != "" {
		t.Errorf("Default().OutputFile = %q, want %q", cfg.OutputFile, "")
	}

	// Backend defaults to native.
	if cfg.Backend != BackendNative {
		t.Errorf("Default().Backend = %q, want %q", cfg.Backend, BackendNative)
	}

	// Preset defaults to classic.
	if cfg.Preset != "classic" {
		t.Errorf("Default().Preset = %q, want %q", cfg.Preset, "classic")
	}

	// Duration defaults to 3s.
	if cfg.Duration != 3*time.Second {
		t.Errorf("Default().Duration = %v, want %v", cfg.Duration, 3*time.Second)
	}

	// Volume defaults to 1.0.
	if cfg.Volume != 1.0 {
		t.Errorf("Default().Volume = %f, want %f", cfg.Volume, 1.0)
	}

	// Format defaults to ogg.
	if cfg.Format != FormatOGG {
		t.Errorf("Default().Format = %q, want %q", cfg.Format, FormatOGG)
	}

	// Bool fields default to false.
	if cfg.DryRun != false {
		t.Errorf("Default().DryRun = %v, want false", cfg.DryRun)
	}
	if cfg.Verbose != false {
		t.Errorf("Default().Verbose = %v, want false", cfg.Verbose)
	}
}

func TestDefault_BackendConstant(t *testing.T) {
	if BackendNative != "native" {
		t.Errorf("BackendNative = %q, want %q", BackendNative, "native")
	}
	if BackendFFmpeg != "ffmpeg" {
		t.Errorf("BackendFFmpeg = %q, want %q", BackendFFmpeg, "ffmpeg")
	}
}

func TestDefault_FormatConstant(t *testing.T) {
	if FormatOGG != "ogg" {
		t.Errorf("FormatOGG = %q, want %q", FormatOGG, "ogg")
	}
	if FormatWAV != "wav" {
		t.Errorf("FormatWAV = %q, want %q", FormatWAV, "wav")
	}
}

// ---------------------------------------------------------------------------
// Merge()
// ---------------------------------------------------------------------------

func TestMerge_ZeroOverlayPreservesBase(t *testing.T) {
	base := Config{
		Token:      "base-token",
		GuildID:    "guild-123",
		Backend:    BackendFFmpeg,
		Preset:     "classic",
		Duration:   5 * time.Second,
		Volume:     0.8,
		OutputFile: "output.ogg",
		Format:     FormatWAV,
		DryRun:     true,
		Verbose:    true,
	}
	overlay := Config{} // all zero values

	got := Merge(base, overlay)

	if got.Token != base.Token {
		t.Errorf("Merge Token = %q, want %q", got.Token, base.Token)
	}
	if got.GuildID != base.GuildID {
		t.Errorf("Merge GuildID = %q, want %q", got.GuildID, base.GuildID)
	}
	if got.Backend != base.Backend {
		t.Errorf("Merge Backend = %q, want %q", got.Backend, base.Backend)
	}
	if got.Preset != base.Preset {
		t.Errorf("Merge Preset = %q, want %q", got.Preset, base.Preset)
	}
	if got.Duration != base.Duration {
		t.Errorf("Merge Duration = %v, want %v", got.Duration, base.Duration)
	}
	if got.Volume != base.Volume {
		t.Errorf("Merge Volume = %f, want %f", got.Volume, base.Volume)
	}
	if got.OutputFile != base.OutputFile {
		t.Errorf("Merge OutputFile = %q, want %q", got.OutputFile, base.OutputFile)
	}
	if got.Format != base.Format {
		t.Errorf("Merge Format = %q, want %q", got.Format, base.Format)
	}
	if got.DryRun != base.DryRun {
		t.Errorf("Merge DryRun = %v, want %v", got.DryRun, base.DryRun)
	}
	if got.Verbose != base.Verbose {
		t.Errorf("Merge Verbose = %v, want %v", got.Verbose, base.Verbose)
	}
}

func TestMerge_NonZeroOverlayWins(t *testing.T) {
	base := Default()
	overlay := Config{
		Token:      "overlay-token",
		GuildID:    "overlay-guild",
		Backend:    BackendFFmpeg,
		Preset:     "banshee",
		Duration:   10 * time.Second,
		Volume:     0.5,
		OutputFile: "scream.wav",
		Format:     FormatWAV,
		DryRun:     true,
		Verbose:    true,
	}

	got := Merge(base, overlay)

	if got.Token != "overlay-token" {
		t.Errorf("Merge Token = %q, want %q", got.Token, "overlay-token")
	}
	if got.GuildID != "overlay-guild" {
		t.Errorf("Merge GuildID = %q, want %q", got.GuildID, "overlay-guild")
	}
	if got.Backend != BackendFFmpeg {
		t.Errorf("Merge Backend = %q, want %q", got.Backend, BackendFFmpeg)
	}
	if got.Preset != "banshee" {
		t.Errorf("Merge Preset = %q, want %q", got.Preset, "banshee")
	}
	if got.Duration != 10*time.Second {
		t.Errorf("Merge Duration = %v, want %v", got.Duration, 10*time.Second)
	}
	if got.Volume != 0.5 {
		t.Errorf("Merge Volume = %f, want %f", got.Volume, 0.5)
	}
	if got.OutputFile != "scream.wav" {
		t.Errorf("Merge OutputFile = %q, want %q", got.OutputFile, "scream.wav")
	}
	if got.Format != FormatWAV {
		t.Errorf("Merge Format = %q, want %q", got.Format, FormatWAV)
	}
	if got.DryRun != true {
		t.Errorf("Merge DryRun = %v, want true", got.DryRun)
	}
	if got.Verbose != true {
		t.Errorf("Merge Verbose = %v, want true", got.Verbose)
	}
}

func TestMerge_PartialOverlay(t *testing.T) {
	base := Config{
		Token:    "base-token",
		GuildID:  "base-guild",
		Backend:  BackendNative,
		Preset:   "classic",
		Duration: 3 * time.Second,
		Volume:   1.0,
		Format:   FormatOGG,
	}
	overlay := Config{
		Token:  "new-token",
		Volume: 0.7,
	}

	got := Merge(base, overlay)

	// Overlay fields should win.
	if got.Token != "new-token" {
		t.Errorf("Merge Token = %q, want %q", got.Token, "new-token")
	}
	if got.Volume != 0.7 {
		t.Errorf("Merge Volume = %f, want %f", got.Volume, 0.7)
	}

	// Base fields should be preserved where overlay is zero.
	if got.GuildID != "base-guild" {
		t.Errorf("Merge GuildID = %q, want %q", got.GuildID, "base-guild")
	}
	if got.Backend != BackendNative {
		t.Errorf("Merge Backend = %q, want %q", got.Backend, BackendNative)
	}
	if got.Preset != "classic" {
		t.Errorf("Merge Preset = %q, want %q", got.Preset, "classic")
	}
	if got.Duration != 3*time.Second {
		t.Errorf("Merge Duration = %v, want %v", got.Duration, 3*time.Second)
	}
	if got.Format != FormatOGG {
		t.Errorf("Merge Format = %q, want %q", got.Format, FormatOGG)
	}
}

func TestMerge_FieldTypes(t *testing.T) {
	tests := []struct {
		name    string
		base    Config
		overlay Config
		check   func(t *testing.T, got Config)
	}{
		{
			name:    "string field: Token override",
			base:    Config{Token: "old"},
			overlay: Config{Token: "new"},
			check: func(t *testing.T, got Config) {
				t.Helper()
				if got.Token != "new" {
					t.Errorf("Token = %q, want %q", got.Token, "new")
				}
			},
		},
		{
			name:    "BackendType field override",
			base:    Config{Backend: BackendNative},
			overlay: Config{Backend: BackendFFmpeg},
			check: func(t *testing.T, got Config) {
				t.Helper()
				if got.Backend != BackendFFmpeg {
					t.Errorf("Backend = %q, want %q", got.Backend, BackendFFmpeg)
				}
			},
		},
		{
			name:    "FormatType field override",
			base:    Config{Format: FormatOGG},
			overlay: Config{Format: FormatWAV},
			check: func(t *testing.T, got Config) {
				t.Helper()
				if got.Format != FormatWAV {
					t.Errorf("Format = %q, want %q", got.Format, FormatWAV)
				}
			},
		},
		{
			name:    "Duration field override",
			base:    Config{Duration: 2 * time.Second},
			overlay: Config{Duration: 7 * time.Second},
			check: func(t *testing.T, got Config) {
				t.Helper()
				if got.Duration != 7*time.Second {
					t.Errorf("Duration = %v, want %v", got.Duration, 7*time.Second)
				}
			},
		},
		{
			name:    "float64 field: Volume override",
			base:    Config{Volume: 1.0},
			overlay: Config{Volume: 0.3},
			check: func(t *testing.T, got Config) {
				t.Helper()
				if got.Volume != 0.3 {
					t.Errorf("Volume = %f, want %f", got.Volume, 0.3)
				}
			},
		},
		{
			name:    "bool field: DryRun override true",
			base:    Config{},
			overlay: Config{DryRun: true},
			check: func(t *testing.T, got Config) {
				t.Helper()
				if got.DryRun != true {
					t.Errorf("DryRun = %v, want true", got.DryRun)
				}
			},
		},
		{
			name:    "bool field: false overlay preserves base true",
			base:    Config{DryRun: true},
			overlay: Config{DryRun: false},
			check: func(t *testing.T, got Config) {
				t.Helper()
				// false is zero value for bool, so base should be preserved.
				if got.DryRun != true {
					t.Errorf("DryRun = %v, want true (false is zero, should preserve base)", got.DryRun)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.base, tt.overlay)
			tt.check(t, got)
		})
	}
}

func TestMerge_BothZero(t *testing.T) {
	got := Merge(Config{}, Config{})

	if got.Token != "" {
		t.Errorf("Token = %q, want empty", got.Token)
	}
	if got.Backend != "" {
		t.Errorf("Backend = %q, want empty", got.Backend)
	}
	if got.Duration != 0 {
		t.Errorf("Duration = %v, want 0", got.Duration)
	}
	if got.Volume != 0 {
		t.Errorf("Volume = %f, want 0", got.Volume)
	}
}

func TestMerge_DoesNotMutateInputs(t *testing.T) {
	base := Config{Token: "base", Volume: 0.5}
	overlay := Config{Token: "overlay"}

	baseCopy := base
	overlayCopy := overlay

	_ = Merge(base, overlay)

	if base.Token != baseCopy.Token || base.Volume != baseCopy.Volume {
		t.Error("Merge mutated the base config")
	}
	if overlay.Token != overlayCopy.Token {
		t.Error("Merge mutated the overlay config")
	}
}
