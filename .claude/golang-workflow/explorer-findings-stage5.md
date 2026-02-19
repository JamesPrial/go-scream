# Explorer Findings: Stage 5 — Configuration and CLI Implementation

**Date:** 2026-02-18
**Working Directory:** /Users/jamesprial/code/go-scream
**Go Version:** 1.25.7 (module: `github.com/JamesPrial/go-scream`)

---

## 1. Directory Status

All five target directories exist but are empty (scaffold placeholders):

| Directory | Status |
|-----------|--------|
| `/Users/jamesprial/code/go-scream/internal/config/` | EXISTS, EMPTY |
| `/Users/jamesprial/code/go-scream/internal/scream/` | EXISTS, EMPTY |
| `/Users/jamesprial/code/go-scream/cmd/scream/` | EXISTS, EMPTY |
| `/Users/jamesprial/code/go-scream/cmd/skill/` | EXISTS, EMPTY |
| `/Users/jamesprial/code/go-scream/pkg/version/` | EXISTS, EMPTY |

---

## 2. All Interfaces Stage 5 Will Consume

### 2.1 audio.AudioGenerator

File: `/Users/jamesprial/code/go-scream/internal/audio/generator.go:6`

```
package audio

import "io"

type AudioGenerator interface {
    Generate(params ScreamParams) (io.Reader, error)
}
```

The orchestrator in `internal/scream/scream.go` accepts an AudioGenerator
and calls Generate(params) to obtain raw s16le PCM.

---

### 2.2 encoding.OpusFrameEncoder

File: `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go:53`

```
type OpusFrameEncoder interface {
    EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error)
}
```

Used by the orchestrator for the play pathway. Returns two channels:
a frame channel and an error channel. Both are closed when encoding ends.

---

### 2.3 encoding.FileEncoder

File: `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go:61`

```
type FileEncoder interface {
    Encode(dst io.Writer, src io.Reader, sampleRate, channels int) error
}
```

Used by `scream generate` to write OGG or WAV files.

---

### 2.4 discord.VoicePlayer

File: `/Users/jamesprial/code/go-scream/internal/discord/player.go:16`

```
type VoicePlayer interface {
    Play(ctx context.Context, guildID, channelID string, frames <-chan []byte) error
}
```

IMPORTANT: Takes a pre-built frame channel, not ScreamParams. The orchestrator
must bridge: Generate -> EncodeFrames -> Play.

---

### 2.5 discord.Session

File: `/Users/jamesprial/code/go-scream/internal/discord/session.go:6`

```
type Session interface {
    ChannelVoiceJoin(guildID, channelID string, mute, deaf bool) (VoiceConn, error)
    GuildVoiceStates(guildID string) ([]*VoiceState, error)
}
```

In play.go: wrap *discordgo.Session as &discord.DiscordGoSession{S: dg}.

---

## 3. All Concrete Types Stage 5 Will Instantiate

### 3.1 native.NativeGenerator

File: `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go:16`

```
type NativeGenerator struct{}
func NewNativeGenerator() *NativeGenerator
```

Implements audio.AudioGenerator. No parameters. Default backend.

---

### 3.2 ffmpeg.FFmpegGenerator

File: `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go:16`

```
type FFmpegGenerator struct{ ffmpegPath string }
func NewFFmpegGenerator() (*FFmpegGenerator, error)          // auto-finds on PATH
func NewFFmpegGeneratorWithPath(path string) *FFmpegGenerator // explicit path
```

Implements audio.AudioGenerator.
Error: returns ffmpeg.ErrFFmpegNotFound if not on PATH.

---

### 3.3 encoding.GopusFrameEncoder

File: `/Users/jamesprial/code/go-scream/internal/encoding/opus.go:22`

```
type GopusFrameEncoder struct{ bitrate int }
func NewGopusFrameEncoder() *GopusFrameEncoder
func NewGopusFrameEncoderWithBitrate(bitrate int) *GopusFrameEncoder
```

Implements encoding.OpusFrameEncoder. Default bitrate: OpusBitrate = 64000.

---

### 3.4 encoding.OGGEncoder

File: `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go:13`

```
type OGGEncoder struct{ opus OpusFrameEncoder }
func NewOGGEncoder() *OGGEncoder
func NewOGGEncoderWithOpus(opus OpusFrameEncoder) *OGGEncoder
```

Implements encoding.FileEncoder. Method: Encode(dst, src, sampleRate, channels).
Default output format for `scream generate`.

---

### 3.5 encoding.WAVEncoder

File: `/Users/jamesprial/code/go-scream/internal/encoding/wav.go:28`

```
type WAVEncoder struct{}
func NewWAVEncoder() *WAVEncoder
```

Implements encoding.FileEncoder.

---

### 3.6 discord.DiscordPlayer

File: `/Users/jamesprial/code/go-scream/internal/discord/player.go:21`

```
type DiscordPlayer struct{ session Session }
func NewDiscordPlayer(session Session) *DiscordPlayer
```

Implements discord.VoicePlayer.
Compile-time check at player.go:26: var _ VoicePlayer = (*DiscordPlayer)(nil)

---

### 3.7 discord.DiscordGoSession

File: `/Users/jamesprial/code/go-scream/internal/discord/session.go:27`

```
type DiscordGoSession struct{ S *discordgo.Session }
```

Implements discord.Session.
Construction: &discord.DiscordGoSession{S: dg}
where dg is *discordgo.Session from discordgo.New("Bot " + token)

---

### 3.8 discord.FindPopulatedChannel

File: `/Users/jamesprial/code/go-scream/internal/discord/channel.go:10`

```
func FindPopulatedChannel(session Session, guildID, botUserID string) (string, error)
```

Error returns:
- discord.ErrEmptyGuildID if guildID is ""
- discord.ErrGuildStateFailed (wrapped) if guild state unavailable
- discord.ErrNoPopulatedChannel if no non-bot users in any voice channel

CRITICAL: Third param `botUserID` must be `dg.State.User.ID` (obtained after
dg.Open() returns). Excludes the bot itself from the occupancy check.

---

## 4. ScreamParams and Presets

### 4.1 audio.ScreamParams

File: `/Users/jamesprial/code/go-scream/internal/audio/params.go:8`

```
type ScreamParams struct {
    Duration   time.Duration
    SampleRate int
    Channels   int
    Seed       int64
    Layers     [5]LayerParams
    Noise      NoiseParams
    Filter     FilterParams
}

func Randomize(seed int64) ScreamParams    // seed=0 uses time.Now().UnixNano()
func (p ScreamParams) Validate() error
```

Standard values: SampleRate=48000, Channels=2, Duration=2.5-4.0s

---

### 4.2 audio.PresetName and presets

File: `/Users/jamesprial/code/go-scream/internal/audio/presets.go`

```
type PresetName string

const (
    PresetClassic    PresetName = "classic"
    PresetWhisper    PresetName = "whisper"
    PresetDeathMetal PresetName = "death-metal"
    PresetGlitch     PresetName = "glitch"
    PresetBanshee    PresetName = "banshee"
    PresetRobot      PresetName = "robot"
)

func AllPresets() []PresetName
func GetPreset(name PresetName) (ScreamParams, bool)  // bool=false if not found
```

Usage in Stage 5:
- `scream presets` calls AllPresets() to list all names
- `--preset classic` calls GetPreset("classic")
- No preset flag: call Randomize(seed) with seed from --seed flag or 0

---

## 5. Error Patterns Used Across the Codebase

Pattern: sentinel errors via errors.New, wrapping via fmt.Errorf("%w: ...", err).

internal/audio/errors.go:
```
var ErrInvalidDuration     = errors.New("duration must be positive")
var ErrInvalidSampleRate   = errors.New("sample rate must be positive")
var ErrInvalidChannels     = errors.New("channels must be 1 or 2")
var ErrInvalidAmplitude    = errors.New("amplitude must be between 0 and 1")
var ErrInvalidFilterCutoff = errors.New("filter cutoff must be non-negative")
var ErrInvalidLimiterLevel = errors.New("limiter level must be between 0 and 1 (exclusive of 0)")
var ErrInvalidCrusherBits  = errors.New("crusher bits must be between 1 and 16")

type LayerValidationError struct{ Layer int; Err error }
func (e *LayerValidationError) Error() string { return fmt.Sprintf("layer %d: %s", e.Layer, e.Err) }
func (e *LayerValidationError) Unwrap() error { return e.Err }
```

internal/encoding/encoder.go:
```
var ErrInvalidSampleRate = errors.New("encoding: sample rate must be positive")
var ErrInvalidChannels   = errors.New("encoding: channels must be 1 or 2")
var ErrOpusEncode        = errors.New("encoding: opus encoding failed")
var ErrWAVWrite          = errors.New("encoding: WAV write failed")
var ErrOGGWrite          = errors.New("encoding: OGG write failed")
```

internal/audio/ffmpeg/errors.go:
```
var ErrFFmpegNotFound = errors.New("ffmpeg: executable not found on PATH")
var ErrFFmpegFailed   = errors.New("ffmpeg: process failed")
```

internal/discord/errors.go:
```
var ErrVoiceJoinFailed    = errors.New("discord: failed to join voice channel")
var ErrSpeakingFailed     = errors.New("discord: failed to set speaking state")
var ErrNoPopulatedChannel = errors.New("discord: no populated voice channel found")
var ErrGuildStateFailed   = errors.New("discord: failed to retrieve guild state")
var ErrEmptyGuildID       = errors.New("discord: guild ID must not be empty")
var ErrEmptyChannelID     = errors.New("discord: channel ID must not be empty")
var ErrNilFrameChannel    = errors.New("discord: frame channel must not be nil")
```

Wrapping pattern:
```
return fmt.Errorf("%w: %w", ErrSomeSentinel, underlyingErr)
return fmt.Errorf("%w: creating encoder: %w", ErrOpusEncode, err)
```

Testing pattern:
```
if !errors.Is(err, ErrVoiceJoinFailed) { t.Errorf(...) }
```

Stage 5 should define package-prefixed errors in config and scream packages, e.g.:
```
var ErrMissingToken = errors.New("config: DISCORD_TOKEN is not set")
```

---

## 6. go.mod Dependencies

File: `/Users/jamesprial/code/go-scream/go.mod`

Current state - cobra and yaml.v3 are NOT present:
```
module github.com/JamesPrial/go-scream

go 1.25.7

require (
    github.com/bwmarrin/discordgo v0.29.0 // indirect
    github.com/gorilla/websocket v1.4.2 // indirect
    github.com/pion/randutil v0.1.0 // indirect
    github.com/pion/rtp v1.10.1 // indirect
    github.com/pion/webrtc/v3 v3.3.6 // indirect
    golang.org/x/crypto v0.21.0 // indirect
    golang.org/x/sys v0.18.0 // indirect
    layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32 // indirect
)
```

### cobra cache status:
- cobra v1.8.1 fully extracted: /Users/jamesprial/go/pkg/mod/github.com/spf13/cobra@v1.8.1/
- cobra v1.8.1 zip cached:      /Users/jamesprial/go/pkg/mod/cache/download/github.com/spf13/cobra/@v/v1.8.1.zip
- cobra v1.8.1 go.mod requires:
  - github.com/cpuguy83/go-md2man/v2 v2.0.4  (indirect, doc generation)
  - github.com/inconshreveable/mousetrap v1.1.0 (indirect, Windows)
  - github.com/spf13/pflag v1.0.5 (indirect, required at runtime)
  - gopkg.in/yaml.v3 v3.0.1 (indirect, cobra uses internally)

### yaml.v3 cache status:
- yaml.v3 v3.0.1 fully extracted: /Users/jamesprial/go/pkg/mod/gopkg.in/yaml.v3@v3.0.1/
- yaml.v3 v3.0.1 zip cached:      /Users/jamesprial/go/pkg/mod/cache/download/gopkg.in/yaml.v3/@v/v3.0.1.zip

### Required go.mod for Stage 5:
```
require (
    github.com/bwmarrin/discordgo v0.29.0
    github.com/spf13/cobra v1.8.1
    gopkg.in/yaml.v3 v3.0.1
    github.com/gorilla/websocket v1.4.2 // indirect
    github.com/pion/randutil v0.1.0 // indirect
    github.com/pion/rtp v1.10.1 // indirect
    github.com/pion/webrtc/v3 v3.3.6 // indirect
    golang.org/x/crypto v0.21.0 // indirect
    golang.org/x/sys v0.18.0 // indirect
    layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32 // indirect
)
```

Run `go mod tidy` after editing to populate go.sum with cobra transitive dep hashes.

---

## 7. cmd/ Directory Structure - Existing State

Both cmd/scream/ and cmd/skill/ are empty. Files to create:

```
cmd/scream/main.go       -- cobra root command, main()
cmd/scream/play.go       -- "scream play <guildId> [channelId]"
cmd/scream/generate.go   -- "scream generate -o output.ogg"
cmd/scream/presets.go    -- "scream presets"
```

---

## 8. Cobra v1.8.1 Command API

Source: /Users/jamesprial/go/pkg/mod/github.com/spf13/cobra@v1.8.1/command.go

Key Command struct fields:
```
Use     string         // "play <guildId> [channelId]"
Short   string         // one-line for help
Long    string         // detailed for help <cmd>
Example string         // usage examples
Args    PositionalArgs
RunE    func(cmd *Command, args []string) error
Version string         // enables --version/-v on root
```

Methods:
```
AddCommand(cmds ...*Command)
Execute() error
Flags() *pflag.FlagSet
PersistentFlags() *pflag.FlagSet
```

Positional arg validators:
```
cobra.ExactArgs(n int) PositionalArgs
cobra.RangeArgs(min, max int) PositionalArgs
cobra.MinimumNArgs(n int) PositionalArgs
cobra.NoArgs PositionalArgs
```

Flag methods (via Flags() or PersistentFlags()):
```
StringP(name, shorthand, value, usage string) *string
StringVarP(p *string, name, shorthand, value, usage string)
BoolP(name, shorthand, value, usage string) *bool
Int64P(name, shorthand, value, usage string) *int64
MarkRequired(name string) error
```

---

## 9. Data Flow Architecture for Stage 5

### scream play flow:
```
play.go
  1. config.Load()                                 -> Config
  2. discordgo.New("Bot " + cfg.DiscordToken)      -> *discordgo.Session
  3. dg.Identify.Intents = Guilds | GuildVoiceStates
  4. dg.Open()
  5. defer dg.Close()
  6. audio.GetPreset(name) OR audio.Randomize(seed) -> ScreamParams
  7. generator.Generate(params)                    -> io.Reader (s16le PCM)
  8. opus.EncodeFrames(reader, 48000, 2)           -> <-chan []byte, <-chan error
  9. if channelId == "":
         discord.FindPopulatedChannel(session, guildId, dg.State.User.ID)
 10. player.Play(ctx, guildId, channelId, frameCh)
 11. encErr := <-errCh
```

### scream generate flow:
```
generate.go
  1. Parse: -o/--output, --preset, --seed, --format (ogg|wav)
  2. audio.GetPreset() OR audio.Randomize()        -> ScreamParams
  3. native.NewNativeGenerator() OR ffmpeg.NewFFmpegGenerator()
  4. generator.Generate(params)                    -> io.Reader
  5. os.Create(outputPath)                         -> *os.File
  6. fileEncoder.Encode(file, reader, 48000, 2)
  7. file.Close()
```

### scream presets flow:
```
presets.go
  1. audio.AllPresets()  -> []PresetName
  2. Print each preset name (optionally with details from GetPreset)
```

---

## 10. internal/scream/scream.go - Orchestrator Design

Bridges the three component interfaces:

```
type Service struct {
    generator audio.AudioGenerator
    opus      encoding.OpusFrameEncoder
    file      encoding.FileEncoder
    player    discord.VoicePlayer
}

func NewService(
    generator audio.AudioGenerator,
    opus      encoding.OpusFrameEncoder,
    file      encoding.FileEncoder,
    player    discord.VoicePlayer,
) *Service

func (s *Service) Play(ctx context.Context, guildID, channelID string, params audio.ScreamParams) error
func (s *Service) Generate(ctx context.Context, dst io.Writer, params audio.ScreamParams) error
```

Play() wiring sequence:
```
reader, err := s.generator.Generate(params)
frameCh, errCh := s.opus.EncodeFrames(reader, params.SampleRate, params.Channels)
err = s.player.Play(ctx, guildID, channelID, frameCh)
encErr := <-errCh
// return first non-nil of err, encErr
```

---

## 11. internal/config/config.go - Design Guidance

```
type Config struct {
    DiscordToken string  // required for `play`
    Generator    string  // "native" or "ffmpeg", default "native"
    FFmpegPath   string  // explicit ffmpeg binary path
}

func Default() Config  // returns Config{Generator: "native"}
func Load() (Config, error)

var ErrMissingToken  = errors.New("config: DISCORD_TOKEN is not set")
var ErrInvalidConfig = errors.New("config: invalid configuration")
```

---

## 12. internal/config/env.go - Environment Variable Overrides

```
func applyEnv(cfg *Config) {
    if v := os.Getenv("DISCORD_TOKEN"); v != "" { cfg.DiscordToken = v }
    if v := os.Getenv("SCREAM_GENERATOR"); v != "" { cfg.Generator = v }
    if v := os.Getenv("FFMPEG_PATH"); v != "" { cfg.FFmpegPath = v }
}
```

Environment variables:
| Variable         | Field              | Notes                   |
|------------------|--------------------|-------------------------|
| DISCORD_TOKEN    | Config.DiscordToken | Required for `play`     |
| SCREAM_GENERATOR | Config.Generator    | "native" or "ffmpeg"   |
| FFMPEG_PATH      | Config.FFmpegPath   | Explicit path override  |

---

## 13. pkg/version/version.go - Build-time Version

```
package version

// Set at build time: go build -ldflags "-X .../version.Version=1.0.0"
var Version   = "dev"
var BuildTime = "unknown"
var Commit    = "unknown"
```

Usage in cobra root:
```
rootCmd.Version = version.Version
// cobra auto-adds --version/-v when Version is non-empty
```

---

## 14. cmd/scream/main.go - Cobra Root Pattern

```
package main

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

var rootCmd = &cobra.Command{
    Use:     "scream",
    Short:   "A Discord voice bot that screams",
    Version: version.Version,
}

func init() {
    rootCmd.AddCommand(playCmd)
    rootCmd.AddCommand(generateCmd)
    rootCmd.AddCommand(presetsCmd)
}
```

---

## 15. Critical: VoicePlayer Takes Frame Channel, Not ScreamParams

The EXISTING (already implemented and tested) discord.DiscordPlayer.Play signature:
```
Play(ctx context.Context, guildID, channelID string, frames <-chan []byte) error
```

NOT:
```
Play(ctx context.Context, guildID, channelID string, params audio.ScreamParams) error
```

The internal/scream/scream.go orchestrator is where ScreamParams gets converted
to frames. The cmd layer calls orchestrator.Play(), not player.Play() directly.

---

## 16. FindPopulatedChannel botUserID Parameter

```
FindPopulatedChannel(session Session, guildID, botUserID string) (string, error)
```

botUserID must be dg.State.User.ID (available after dg.Open() succeeds).
Purpose: prevents counting the bot's own voice state as a "populated" channel.
Without this, if the bot is already in a voice channel, it would select itself.

---

## 17. Existing Test Patterns to Follow

Table-driven tests (from discord/player_test.go):
```
tests := []struct {
    name    string
    input   SomeType
    wantErr error
}{
    {"empty guild ID", "", ErrEmptyGuildID},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        err := fn(tt.input)
        if !errors.Is(err, tt.wantErr) {
            t.Errorf("fn() error = %v, want %v", err, tt.wantErr)
        }
    })
}
```

Interface compliance (compile-time):
```
var _ VoicePlayer = (*DiscordPlayer)(nil)
```

Constructor nil check:
```
func TestNewXxx_NotNil(t *testing.T) {
    x := NewXxx(...)
    if x == nil { t.Fatal("NewXxx() returned nil") }
}
```

---

## 18. Summary: Files to Create in Stage 5

| File                             | Purpose                  | Key Dependencies                                    |
|----------------------------------|--------------------------|-----------------------------------------------------|
| pkg/version/version.go           | Build-time version vars  | none                                                |
| internal/config/config.go        | Config struct, Load, Default | yaml.v3 (optional), os                          |
| internal/config/env.go           | Env var overrides        | os stdlib                                           |
| internal/scream/scream.go        | Service orchestrator     | internal/audio, internal/discord, internal/encoding |
| cmd/scream/main.go               | cobra root + main()      | cobra, pkg/version                                  |
| cmd/scream/play.go               | scream play subcommand   | cobra, discordgo, config, scream, audio, discord    |
| cmd/scream/generate.go           | scream generate subcmd   | cobra, config, audio, encoding, native/ffmpeg       |
| cmd/scream/presets.go            | scream presets subcmd    | cobra, audio                                        |

---

## 19. go.mod Changes Required

Add as direct deps:
```
github.com/spf13/cobra v1.8.1   (fully cached at v1.8.1)
gopkg.in/yaml.v3 v3.0.1         (fully cached at v3.0.1)
```

cobra will bring these new indirect deps not yet in go.sum:
```
github.com/cpuguy83/go-md2man/v2 v2.0.4  // indirect
github.com/inconshreveable/mousetrap v1.1.0 // indirect (Windows)
github.com/spf13/pflag v1.0.5            // indirect (required)
```

Run `go mod tidy` after editing go.mod to populate all hashes in go.sum.

---

## 20. Package Import Paths

Module path: github.com/JamesPrial/go-scream

```
// Stage 5 new packages:
"github.com/JamesPrial/go-scream/internal/config"
"github.com/JamesPrial/go-scream/internal/scream"
"github.com/JamesPrial/go-scream/pkg/version"

// Existing packages consumed by Stage 5:
"github.com/JamesPrial/go-scream/internal/audio"
"github.com/JamesPrial/go-scream/internal/audio/native"
"github.com/JamesPrial/go-scream/internal/audio/ffmpeg"
"github.com/JamesPrial/go-scream/internal/discord"
"github.com/JamesPrial/go-scream/internal/encoding"

// External deps (new for Stage 5):
"github.com/spf13/cobra"
"gopkg.in/yaml.v3"

// External deps (already in go.mod):
"github.com/bwmarrin/discordgo"
"layeh.com/gopus"
```
