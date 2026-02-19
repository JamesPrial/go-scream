package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// BackendType identifies the audio generation backend.
type BackendType string

const (
	// BackendNative uses the native Go audio generator.
	BackendNative BackendType = "native"

	// BackendFFmpeg uses the FFmpeg audio generator.
	BackendFFmpeg BackendType = "ffmpeg"
)

// FormatType identifies the output audio file format.
type FormatType string

const (
	// FormatOGG produces Ogg/Opus encoded output.
	FormatOGG FormatType = "ogg"

	// FormatWAV produces WAV encoded output.
	FormatWAV FormatType = "wav"
)

// Config holds all configuration values for the go-scream bot.
type Config struct {
	Token      string        `yaml:"token"`
	GuildID    string        `yaml:"guild_id"`
	Backend    BackendType   `yaml:"backend"`
	Preset     string        `yaml:"preset"`
	Duration   time.Duration `yaml:"duration"`
	Volume     float64       `yaml:"volume"`
	OutputFile string        `yaml:"output_file"`
	Format     FormatType    `yaml:"format"`
	DryRun     bool          `yaml:"dry_run"`
	Verbose    bool          `yaml:"verbose"`
}

// rawConfig is an intermediate struct used for YAML unmarshaling. It captures
// the duration field as a yaml.Node so we can parse Go duration strings like
// "5s", "500ms", "1m30s" rather than treating them as integer nanoseconds.
type rawConfig struct {
	Token      string      `yaml:"token"`
	GuildID    string      `yaml:"guild_id"`
	Backend    BackendType `yaml:"backend"`
	Preset     string      `yaml:"preset"`
	Duration   yaml.Node   `yaml:"duration"`
	Volume     float64     `yaml:"volume"`
	OutputFile string      `yaml:"output_file"`
	Format     FormatType  `yaml:"format"`
	DryRun     bool        `yaml:"dry_run"`
	Verbose    bool        `yaml:"verbose"`
}

// UnmarshalYAML implements yaml.Unmarshaler so that duration fields are parsed
// from Go duration strings (e.g. "5s", "500ms", "1m30s") rather than integer
// nanoseconds.
func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	var raw rawConfig
	if err := value.Decode(&raw); err != nil {
		return err
	}

	c.Token = raw.Token
	c.GuildID = raw.GuildID
	c.Backend = raw.Backend
	c.Preset = raw.Preset
	c.Volume = raw.Volume
	c.OutputFile = raw.OutputFile
	c.Format = raw.Format
	c.DryRun = raw.DryRun
	c.Verbose = raw.Verbose

	// Parse duration from the raw YAML node when present.
	if raw.Duration.Value != "" {
		d, err := time.ParseDuration(raw.Duration.Value)
		if err != nil {
			return fmt.Errorf("config: invalid duration %q: %w", raw.Duration.Value, err)
		}
		c.Duration = d
	}

	return nil
}

// Default returns a Config with sensible default values.
// Backend defaults to "native", Preset to "classic", Duration to 3 seconds,
// Volume to 1.0, and Format to "ogg". All other fields are zero values.
func Default() Config {
	return Config{
		Backend:  BackendNative,
		Preset:   "classic",
		Duration: 3 * time.Second,
		Volume:   1.0,
		Format:   FormatOGG,
	}
}

// Merge combines base and overlay into a new Config. Non-zero overlay fields
// replace the corresponding base fields. Zero values (empty string, 0 duration,
// 0.0 float64, false bool) are treated as unset and the base value is kept.
// Neither base nor overlay is mutated.
func Merge(base, overlay Config) Config {
	result := base

	if overlay.Token != "" {
		result.Token = overlay.Token
	}
	if overlay.GuildID != "" {
		result.GuildID = overlay.GuildID
	}
	if overlay.Backend != "" {
		result.Backend = overlay.Backend
	}
	if overlay.Preset != "" {
		result.Preset = overlay.Preset
	}
	if overlay.Duration != 0 {
		result.Duration = overlay.Duration
	}
	if overlay.Volume != 0 {
		result.Volume = overlay.Volume
	}
	if overlay.OutputFile != "" {
		result.OutputFile = overlay.OutputFile
	}
	if overlay.Format != "" {
		result.Format = overlay.Format
	}
	if overlay.DryRun {
		result.DryRun = overlay.DryRun
	}
	if overlay.Verbose {
		result.Verbose = overlay.Verbose
	}

	return result
}
