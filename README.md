# go-scream

A Discord voice bot that generates unique synthetic screams. Produces audio from pure Go synthesis or FFmpeg, streams it to Discord voice channels, or saves it to OGG/WAV files.

## Installation

Requires Go 1.24+ and libopus (for Opus encoding via CGO).

```bash
# macOS
brew install opus

# Ubuntu/Debian
sudo apt-get install libopus-dev

# Build
go build -o scream ./cmd/scream
```

## Usage

### Play in Discord

```bash
# Play a scream in a voice channel
scream play --token $DISCORD_TOKEN <guildID> [channelID]

# Use a preset
scream play --token $DISCORD_TOKEN --preset death-metal <guildID>

# Customize duration and volume
scream play --token $DISCORD_TOKEN --duration 5s --volume 0.8 <guildID>
```

If `channelID` is omitted, the bot auto-detects the first populated voice channel in the guild.

### Generate to file

```bash
# Generate an OGG file
scream generate --output scream.ogg

# Generate a WAV file with a specific preset
scream generate --output scream.wav --format wav --preset banshee
```

### List presets

```bash
scream presets
```

Available presets: `classic`, `whisper`, `death-metal`, `glitch`, `banshee`, `robot`. If no preset is specified, parameters are randomized.

## Configuration

Settings are resolved in order (last wins): defaults, YAML config file, environment variables, CLI flags.

### YAML config

```bash
scream play --config scream.yaml --token $DISCORD_TOKEN <guildID>
```

### Environment variables

| Variable | Description |
|---|---|
| `DISCORD_TOKEN` | Discord bot token |
| `SCREAM_BACKEND` | `native` (default) or `ffmpeg` |
| `SCREAM_PRESET` | Preset name |
| `SCREAM_DURATION` | Duration (e.g. `3s`, `500ms`) |
| `SCREAM_VOLUME` | Volume `0.0`-`1.0` |
| `SCREAM_FORMAT` | Output format: `ogg` (default) or `wav` |

## Audio backends

- **native** (default): Pure Go synthesis. Five layered oscillators (primary scream, harmonic sweep, high shriek, noise bursts, background noise) processed through a filter chain (high-pass, low-pass, bit-crusher, compressor, limiter).
- **ffmpeg**: Delegates synthesis to an FFmpeg subprocess. Requires `ffmpeg` on `$PATH`.

## OpenClaw skill

A second binary (`cmd/skill`) wraps go-scream as an [OpenClaw](https://github.com/openclaw) skill. See [SKILL.md](SKILL.md) for details.

```bash
go build -o skill ./cmd/skill
skill <guildID> [channelID]
```

## License

MIT - see [LICENSE](LICENSE).
