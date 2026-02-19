package config

// knownPresets lists every valid preset name accepted by Validate.
// This list must be kept in sync with the preset constants defined in
// internal/audio/presets.go (audio.AllPresets).
var knownPresets = []string{
	"classic",
	"whisper",
	"death-metal",
	"glitch",
	"banshee",
	"robot",
}

// Validate checks that cfg contains valid values for all fields.
// It returns the first validation error encountered, or nil if cfg is valid.
//
// Rules:
//   - Backend must be BackendNative or BackendFFmpeg
//   - Preset, if non-empty, must be one of the known preset names
//   - Duration must be > 0
//   - Volume must be >= 0.0 and <= 1.0
//   - Format must be FormatOGG or FormatWAV
func Validate(cfg Config) error {
	if cfg.Backend != BackendNative && cfg.Backend != BackendFFmpeg {
		return ErrInvalidBackend
	}

	if cfg.Preset != "" {
		if !isValidPreset(cfg.Preset) {
			return ErrInvalidPreset
		}
	}

	if cfg.Duration <= 0 {
		return ErrInvalidDuration
	}

	if cfg.Volume < 0.0 || cfg.Volume > 1.0 {
		return ErrInvalidVolume
	}

	if cfg.Format != FormatOGG && cfg.Format != FormatWAV {
		return ErrInvalidFormat
	}

	return nil
}

// isValidPreset reports whether name matches one of the known preset names.
func isValidPreset(name string) bool {
	for _, p := range knownPresets {
		if p == name {
			return true
		}
	}
	return false
}
