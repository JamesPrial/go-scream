package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Load reads a YAML config file at path and returns the parsed Config.
// If the file does not exist, ErrConfigNotFound is returned (wrapped).
// If the file cannot be parsed, ErrConfigParse is returned (wrapped).
// An empty file returns a zero-value Config with no error.
// Unknown YAML fields are silently ignored.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("%w: %s: %w", ErrConfigNotFound, path, err)
		}
		return Config{}, fmt.Errorf("%w: %w", ErrConfigParse, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrConfigParse, err)
	}

	return cfg, nil
}

// ApplyEnv overlays environment variable values onto cfg. Only non-empty
// environment variable values are applied; empty strings leave the field
// unchanged. Parse errors for numeric or boolean values are silently ignored.
//
// Supported variables:
//   - DISCORD_TOKEN  -> cfg.Token
//   - SCREAM_GUILD_ID -> cfg.GuildID
//   - SCREAM_BACKEND  -> cfg.Backend
//   - SCREAM_PRESET   -> cfg.Preset
//   - SCREAM_DURATION -> cfg.Duration (Go duration string, e.g. "5s")
//   - SCREAM_VOLUME   -> cfg.Volume (float64)
//   - SCREAM_FORMAT   -> cfg.Format
//   - SCREAM_VERBOSE  -> cfg.Verbose (bool)
func ApplyEnv(cfg *Config) {
	if v := os.Getenv("DISCORD_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("SCREAM_GUILD_ID"); v != "" {
		cfg.GuildID = v
	}
	if v := os.Getenv("SCREAM_BACKEND"); v != "" {
		cfg.Backend = BackendType(v)
	}
	if v := os.Getenv("SCREAM_PRESET"); v != "" {
		cfg.Preset = v
	}
	if v := os.Getenv("SCREAM_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Duration = d
		}
	}
	if v := os.Getenv("SCREAM_VOLUME"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Volume = f
		}
	}
	if v := os.Getenv("SCREAM_FORMAT"); v != "" {
		cfg.Format = FormatType(v)
	}
	if v := os.Getenv("SCREAM_VERBOSE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Verbose = b
		}
	}
	if v := os.Getenv("SCREAM_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
}
