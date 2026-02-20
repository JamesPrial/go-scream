// Package config provides configuration loading, merging, and validation
// for the go-scream bot.
package config

import "errors"

// Sentinel errors returned by the config package.
var (
	// ErrConfigNotFound is returned when the config file does not exist.
	ErrConfigNotFound = errors.New("config: file not found")

	// ErrConfigParse is returned when the config file cannot be parsed.
	ErrConfigParse = errors.New("config: failed to parse config file")

	// ErrInvalidBackend is returned when the backend is not "native" or "ffmpeg".
	ErrInvalidBackend = errors.New("config: backend must be 'native' or 'ffmpeg'")

	// ErrInvalidPreset is returned when the preset name is not known.
	ErrInvalidPreset = errors.New("config: unknown preset name")

	// ErrInvalidDuration is returned when the duration is not positive.
	ErrInvalidDuration = errors.New("config: duration must be positive")

	// ErrInvalidVolume is returned when the volume is outside [0.0, 1.0].
	ErrInvalidVolume = errors.New("config: volume must be between 0.0 and 1.0")

	// ErrInvalidFormat is returned when the format is not "ogg" or "wav".
	ErrInvalidFormat = errors.New("config: format must be 'ogg' or 'wav'")

	// ErrMissingToken is returned when the Discord token is not set.
	// Used by the service layer and CLI for context-specific validation.
	ErrMissingToken = errors.New("config: discord token is required")

	// ErrMissingGuildID is returned when the guild ID is not set.
	// Used by the service layer and CLI for context-specific validation.
	ErrMissingGuildID = errors.New("config: guild ID is required")

	// ErrMissingOutput is returned when the output file path is not set.
	// Used by the CLI generate subcommand for context-specific validation.
	ErrMissingOutput = errors.New("config: output file path is required")

	// ErrInvalidLogLevel is returned when the log level is not one of the
	// accepted values: debug, info, warn, or error.
	ErrInvalidLogLevel = errors.New("config: invalid log level (must be debug, info, warn, or error)")
)
