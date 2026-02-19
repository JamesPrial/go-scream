# Explorer Findings: Stage 4 — Discord Integration

**Date:** 2026-02-18
**Working Directory:** /Users/jamesprial/code/go-scream
**Go Version:** 1.25.7 (module: `github.com/JamesPrial/go-scream`)

---

## 1. OpusFrameEncoder Interface

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (lines 53–59)

```go
// OpusFrameEncoder encodes raw PCM audio into a stream of Opus frames.
type OpusFrameEncoder interface {
    // EncodeFrames reads s16le PCM data from src and sends encoded Opus frames
    // on the returned channel. Any error (including nil at completion) is sent
    // on the error channel. Both channels are closed after completion.
    EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error)
}
```

**Also defined in the same file:**

```go
// FileEncoder encodes raw PCM audio into a container format written to dst.
type FileEncoder interface {
    Encode(dst io.Writer, src io.Reader, sampleRate, channels int) error
}
```

**The Discord player uses `OpusFrameEncoder`** — it calls `EncodeFrames`, consumes the `<-chan []byte` frame channel, and sends each `[]byte` to `vc.OpusSend`. The `<-chan error` must be drained to completion after the frame channel closes.

---

## 2. AudioGenerator Interface

**File:** `/Users/jamesprial/code/go-scream/internal/audio/generator.go` (lines 1–8)

```go
package audio

import "io"

// AudioGenerator produces raw PCM audio data (s16le, 48kHz, stereo).
type AudioGenerator interface {
    Generate(params ScreamParams) (io.Reader, error)
}
```

The `DiscordPlayer` receives an `AudioGenerator` and calls `Generate(params)` to get the `io.Reader` of raw s16le PCM at 48kHz stereo, which is then fed to `OpusFrameEncoder.EncodeFrames`.

---

## 3. Encoding Constants

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (lines 11–21)

```go
const (
    // OpusFrameSamples is the number of samples per channel per Opus frame (20ms at 48kHz).
    OpusFrameSamples = 960

    // MaxOpusFrameBytes is the maximum size in bytes of an encoded Opus frame.
    MaxOpusFrameBytes = 3840

    // OpusBitrate is the default Opus encoding bitrate in bits per second.
    OpusBitrate = 64000
)
```

**Discord voice requirement**: Opus 48kHz, 960 samples/frame (20ms). This matches `OpusFrameSamples = 960` exactly. The discordgo internals confirm: `go v.opusSender(v.udpConn, v.close, v.OpusSend, 48000, 960)` (voice.go line 458). No mismatch.

---

## 4. Error Patterns

### internal/audio/errors.go

**File:** `/Users/jamesprial/code/go-scream/internal/audio/errors.go`

```go
var (
    ErrInvalidDuration     = errors.New("duration must be positive")
    ErrInvalidSampleRate   = errors.New("sample rate must be positive")
    ErrInvalidChannels     = errors.New("channels must be 1 or 2")
    ErrInvalidAmplitude    = errors.New("amplitude must be between 0 and 1")
    ErrInvalidFilterCutoff = errors.New("filter cutoff must be non-negative")
    ErrInvalidLimiterLevel = errors.New("limiter level must be between 0 and 1 (exclusive of 0)")
    ErrInvalidCrusherBits  = errors.New("crusher bits must be between 1 and 16")
)

type LayerValidationError struct {
    Layer int
    Err   error
}
func (e *LayerValidationError) Error() string { return fmt.Sprintf("layer %d: %s", e.Layer, e.Err) }
func (e *LayerValidationError) Unwrap() error { return e.Err }
```

### internal/encoding/encoder.go

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (lines 35–51)

```go
var (
    ErrInvalidSampleRate = errors.New("encoding: sample rate must be positive")
    ErrInvalidChannels   = errors.New("encoding: channels must be 1 or 2")
    ErrOpusEncode        = errors.New("encoding: opus encoding failed")
    ErrWAVWrite          = errors.New("encoding: WAV write failed")
    ErrOGGWrite          = errors.New("encoding: OGG write failed")
)
```

### Discord package error pattern to follow

The `internal/discord` package should define its own sentinel errors prefixed with the domain, and wrap with `fmt.Errorf("...: %w", err)`:

```go
var (
    ErrNoToken          = errors.New("discord: no bot token provided")
    ErrSessionOpen      = errors.New("discord: failed to open session")
    ErrVoiceJoin        = errors.New("discord: failed to join voice channel")
    ErrNoVoiceChannel   = errors.New("discord: no populated voice channel found")
    ErrGenerateAudio    = errors.New("discord: audio generation failed")
    ErrEncode           = errors.New("discord: opus encoding failed")
)
```

---

## 5. Node.js Reference Bot — Discord Voice Handling

**File:** `/tmp/scream-reference/scripts/scream.mjs`

### Token resolution pattern

```javascript
function getToken() {
  if (process.env.DISCORD_TOKEN) return process.env.DISCORD_TOKEN;
  // Fallback: read from ~/.openclaw/openclaw.json -> channels.discord.token
  const configPath = join(process.env.HOME, '.openclaw', 'openclaw.json');
  const config = JSON.parse(readFileSync(configPath, 'utf8'));
  return config?.channels?.discord?.token;
}
```

Go equivalent: read `DISCORD_TOKEN` env var, then fall back to parsing `~/.openclaw/openclaw.json`.

### Discord client setup

```javascript
const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildVoiceStates,
  ],
});
```

Go discordgo equivalent:
```go
dg, err := discordgo.New("Bot " + token)
dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates
err = dg.Open()
```

### Auto-detect first populated voice channel

```javascript
const voiceChannels = guild.channels.cache.filter(
  ch => ch.isVoiceBased() && ch.members.size > 0
);
const target = voiceChannels.first();
targetChannelId = target.id;
```

Go discordgo equivalent using `Guild.VoiceStates` (state-cached):
```go
// guild is *discordgo.Guild from dg.State.Guild(guildID)
// Build a set of ChannelIDs with members currently in them
occupied := map[string]bool{}
for _, vs := range guild.VoiceStates {
    if vs.ChannelID != "" {
        occupied[vs.ChannelID] = true
    }
}
// Find first voice channel that is occupied
channels, _ := dg.GuildChannels(guildID)
for _, ch := range channels {
    if ch.Type == discordgo.ChannelTypeGuildVoice && occupied[ch.ID] {
        return ch.ID, nil
    }
}
```

**Note:** `dg.State.Guild(guildID)` requires the bot to be connected and `StateEnabled = true` (default). For voice auto-detection, the State must be populated after the READY event. Alternatively, use `dg.GuildChannels(guildID)` (REST) and `dg.GuildMembers` to cross-check — but the State approach matching the reference is simpler.

### Join voice channel

```javascript
const connection = joinVoiceChannel({ channelId, guildId, adapterCreator, daveEncryption: false });
await entersState(connection, VoiceConnectionStatus.Ready, CONNECT_TIMEOUT_MS);
```

Go discordgo equivalent:
```go
vc, err := dg.ChannelVoiceJoin(guildID, channelID, false, true)
// false = not muted, true = deafened (bot does not need to receive audio)
// ChannelVoiceJoin blocks until connected (calls waitUntilConnected internally)
```

### Play audio (the critical discordgo pattern)

From the official airhorn example (`discordgo@v0.29.0/examples/airhorn/main.go`):

```go
// 1. Join the voice channel
vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)

// 2. Optional: brief sleep to ensure connection is stable
time.Sleep(250 * time.Millisecond)

// 3. Signal speaking = true BEFORE sending audio
vc.Speaking(true)

// 4. Send each Opus frame to the OpusSend channel
for _, buff := range buffer {
    vc.OpusSend <- buff
}

// 5. Signal speaking = false AFTER all frames are sent
vc.Speaking(false)

// 6. Brief sleep to ensure last packets are transmitted
time.Sleep(250 * time.Millisecond)

// 7. Disconnect
vc.Disconnect()
```

**Key detail from the plan**: Send **5 silence frames** before calling `vc.Speaking(false)` to prevent audio clipping. An Opus silence frame for discordgo is `[]byte{0xF8, 0xFF, 0xFE}` (3-byte Opus silence packet). This is standard practice.

Silence frame sequence:
```go
// After all real audio frames, send 5 silence frames
silenceFrame := []byte{0xF8, 0xFF, 0xFE}
for i := 0; i < 5; i++ {
    vc.OpusSend <- silenceFrame
}
vc.Speaking(false)
```

### Reference bot's connection teardown

```javascript
await sleep(500);          // 500ms pause before destroying
connection.destroy();      // disconnect
```

Go discordgo: `time.Sleep(250 * time.Millisecond)` then `vc.Disconnect()`.

---

## 6. go.mod — Available Dependencies

**File:** `/Users/jamesprial/code/go-scream/go.mod`

```
module github.com/JamesPrial/go-scream

go 1.25.7

require (
    github.com/pion/randutil v0.1.0 // indirect
    github.com/pion/rtp v1.10.1 // indirect
    github.com/pion/webrtc/v3 v3.3.6 // indirect
    layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32 // indirect
)
```

**discordgo is NOT in go.mod yet.** It must be added for Stage 4.

Version available in module cache: **`github.com/bwmarrin/discordgo v0.29.0`**
Path in mod cache: `/Users/jamesprial/go/pkg/mod/github.com/bwmarrin/discordgo@v0.29.0`

The `go.mod` change needed:
```
require (
    github.com/bwmarrin/discordgo v0.29.0
    // ... existing deps
)
```

discordgo v0.29.0's own dependencies (from its go.mod):
```
github.com/gorilla/websocket v1.4.2
golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
```

These will be added to go.sum automatically on `go mod tidy`.

---

## 7. internal/discord/ Directory — Status

**Path:** `/Users/jamesprial/code/go-scream/internal/discord/`

**Status: EMPTY.** The directory exists but contains zero files. This is the target package for Stage 4.

```
/Users/jamesprial/code/go-scream/internal/discord/
    (no files)
```

---

## 8. Pion RTP Usage in internal/encoding/ogg.go

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go`

The OGG encoder wraps each Opus frame in an RTP packet using `github.com/pion/rtp`:

```go
import (
    "github.com/pion/rtp"
    "github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

// Per frame:
seqNum++
timestamp += uint32(OpusFrameSamples)  // += 960

pkt := &rtp.Packet{
    Header: rtp.Header{
        Version:        2,
        PayloadType:    111,  // oggPayloadType — IANA dynamic Opus payload type
        SequenceNumber: seqNum,
        Timestamp:      timestamp,
        SSRC:           1,    // oggSSRC — fixed single-stream SSRC
    },
    Payload: frame,
}
oggWriter.WriteRTP(pkt)
```

**Relevance to Stage 4:** The Discord player does NOT use pion/rtp directly. The discordgo `vc.OpusSend` channel accepts raw Opus frame bytes (`[]byte`). The RTP framing is handled internally by discordgo's `opusSender` function in `voice.go`. The Discord player should feed raw Opus frames directly from `OpusFrameEncoder.EncodeFrames` to `vc.OpusSend`.

---

## 9. discordgo v0.29.0 Voice API — Complete Reference

**Source:** `/Users/jamesprial/go/pkg/mod/github.com/bwmarrin/discordgo@v0.29.0/voice.go`

### VoiceConnection struct (key fields)

```go
type VoiceConnection struct {
    sync.RWMutex

    Ready     bool    // true when voice is ready to send audio
    UserID    string
    GuildID   string
    ChannelID string

    OpusSend chan []byte   // Send opus audio frames here (buffered, size 2)
    OpusRecv chan *Packet  // Receive opus audio (not needed for playback)

    // ... internal fields
}
```

### Session methods

```go
// New creates a Session with "Bot " prefix required for bot tokens
func New(token string) (s *Session, err error)

// Open connects the WebSocket to Discord gateway
func (s *Session) Open() error

// Close cleanly closes the session
func (s *Session) Close() error

// ChannelVoiceJoin joins a voice channel and blocks until connected
// gID: Guild ID, cID: Channel ID, mute: false, deaf: true (bot doesn't listen)
func (s *Session) ChannelVoiceJoin(gID, cID string, mute, deaf bool) (voice *VoiceConnection, err error)

// GuildChannels returns all channels in a guild (REST API call)
func (s *Session) GuildChannels(guildID string, options ...RequestOption) (st []*Channel, err error)

// State.Guild returns a guild from the in-memory state cache
func (s *State) Guild(guildID string) (*Guild, error)
```

### VoiceConnection methods

```go
// Speaking sends the "speaking" flag to Discord over voice WebSocket
// Must be called with true BEFORE sending audio frames
// Must be called with false AFTER all frames (including silence frames) are sent
func (v *VoiceConnection) Speaking(b bool) (err error)

// Disconnect disconnects from the voice channel and closes all connections
func (v *VoiceConnection) Disconnect() (err error)
```

### Internal behavior details (from voice.go source)

1. `ChannelVoiceJoin` calls `waitUntilConnected()` which polls `v.Ready` up to 10 seconds.
2. When `Ready = true`, discordgo has already started the `opusSender` goroutine on a 20ms ticker (`960/(48000/1000)` = 20ms per frame).
3. The `opusSender` goroutine reads from `v.OpusSend` channel and sends RTP UDP packets. **It automatically calls `v.Speaking(true)` the first time it reads a frame** — but it is safer and more correct to call `vc.Speaking(true)` explicitly before sending, as the airhorn example does.
4. `v.OpusSend` is buffered with size 2.

### Gateway intents required

```go
// From structs.go lines 3026, 3033
discordgo.IntentsGuilds         Intent = 1 << 0  // needed for guild/channel info
discordgo.IntentsGuildVoiceStates Intent = 1 << 7  // needed for VoiceStates in Guild

// Combined:
dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates
```

### ChannelType for voice channels

```go
// From structs.go lines 302–319
ChannelTypeGuildVoice ChannelType = 2
```

### VoiceState struct (for finding occupied channels)

```go
type VoiceState struct {
    GuildID   string  `json:"guild_id"`
    ChannelID string  `json:"channel_id"`
    UserID    string  `json:"user_id"`
    Member    *Member `json:"member"`
    // ... other fields
}
```

Available on `guild.VoiceStates` from state cache. Each entry represents one user currently in a voice channel. To find occupied channels: collect all `ChannelID` values from `VoiceStates` where `ChannelID != ""`.

---

## 10. Architecture: Stage 4 Components

### Files to create

| File | Purpose |
|------|---------|
| `/Users/jamesprial/code/go-scream/internal/discord/player.go` | `VoicePlayer` interface + `DiscordPlayer` implementation |
| `/Users/jamesprial/code/go-scream/internal/discord/channel.go` | Voice channel auto-detection |
| `/Users/jamesprial/code/go-scream/internal/discord/player_test.go` | Tests for DiscordPlayer |
| `/Users/jamesprial/code/go-scream/internal/discord/channel_test.go` | Tests for channel auto-detection |

### Planned VoicePlayer interface

```go
// VoicePlayer joins a Discord voice channel, plays audio, and disconnects.
type VoicePlayer interface {
    // Play generates audio from params, joins the voice channel, streams audio,
    // and disconnects. guildID must be non-empty. channelID may be empty to
    // trigger auto-detection of the first populated voice channel.
    Play(ctx context.Context, guildID, channelID string, params audio.ScreamParams) error
}
```

### DiscordPlayer struct (planned)

```go
type DiscordPlayer struct {
    token     string
    generator audio.AudioGenerator
    encoder   encoding.OpusFrameEncoder
}

func NewDiscordPlayer(token string, generator audio.AudioGenerator, encoder encoding.OpusFrameEncoder) *DiscordPlayer

// Verify interface compliance
var _ VoicePlayer = (*DiscordPlayer)(nil)
```

### Data flow for Play()

```
params (ScreamParams)
    |
    v
AudioGenerator.Generate(params) --> io.Reader (s16le PCM, 48kHz, stereo)
    |
    v
OpusFrameEncoder.EncodeFrames(reader, 48000, 2)
    --> <-chan []byte (opus frames)
    --> <-chan error
    |
    v
vc.Speaking(true)
for frame := range frameCh {
    vc.OpusSend <- frame
}
// 5 silence frames
for i := 0; i < 5; i++ { vc.OpusSend <- []byte{0xF8, 0xFF, 0xFE} }
vc.Speaking(false)
err = <-errCh
time.Sleep(250 * time.Millisecond)
vc.Disconnect()
```

### Channel auto-detection flow (channel.go)

```go
// FindPopulatedVoiceChannel finds the first voice channel in guildID that has
// at least one member currently in it. Uses VoiceStates from the State cache.
func FindPopulatedVoiceChannel(s *discordgo.Session, guildID string) (channelID string, err error)
```

Internal algorithm:
1. `guild, err := s.State.Guild(guildID)` — get state-cached guild
2. Build `occupiedChannels map[string]bool` from `guild.VoiceStates`
3. Call `s.GuildChannels(guildID)` — REST call to get all channels
4. Iterate channels, return first where `ch.Type == discordgo.ChannelTypeGuildVoice && occupiedChannels[ch.ID]`
5. If none found, return `"", ErrNoVoiceChannel`

---

## 11. Full Playback Sequence for DiscordPlayer.Play()

```go
func (p *DiscordPlayer) Play(ctx context.Context, guildID, channelID string, params audio.ScreamParams) error {
    // 1. Create discordgo session
    dg, err := discordgo.New("Bot " + p.token)
    if err != nil {
        return fmt.Errorf("%w: %w", ErrSessionOpen, err)
    }
    dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates

    // 2. Open websocket connection
    if err := dg.Open(); err != nil {
        return fmt.Errorf("%w: %w", ErrSessionOpen, err)
    }
    defer dg.Close()

    // 3. Resolve channel (auto-detect if not provided)
    if channelID == "" {
        channelID, err = FindPopulatedVoiceChannel(dg, guildID)
        if err != nil {
            return err
        }
    }

    // 4. Generate audio
    reader, err := p.generator.Generate(params)
    if err != nil {
        return fmt.Errorf("%w: %w", ErrGenerateAudio, err)
    }

    // 5. Start Opus encoding (runs in goroutine)
    frameCh, errCh := p.encoder.EncodeFrames(reader, 48000, 2)

    // 6. Join voice channel
    vc, err := dg.ChannelVoiceJoin(guildID, channelID, false, true)
    if err != nil {
        // Drain encoder channels to prevent goroutine leak
        for range frameCh {}
        <-errCh
        return fmt.Errorf("%w: %w", ErrVoiceJoin, err)
    }

    // 7. Brief stabilization pause
    time.Sleep(250 * time.Millisecond)

    // 8. Start speaking
    vc.Speaking(true)

    // 9. Send all audio frames
    for frame := range frameCh {
        select {
        case vc.OpusSend <- frame:
        case <-ctx.Done():
            // Context cancelled — send silence, stop, disconnect
            goto stop
        }
    }

stop:
    // 10. Send 5 silence frames to prevent audio clipping
    silenceFrame := []byte{0xF8, 0xFF, 0xFE}
    for i := 0; i < 5; i++ {
        vc.OpusSend <- silenceFrame
    }

    // 11. Stop speaking
    vc.Speaking(false)

    // 12. Collect encoding error
    encErr := <-errCh

    // 13. Brief pause for last packets to transmit
    time.Sleep(250 * time.Millisecond)

    // 14. Disconnect
    vc.Disconnect()

    return encErr
}
```

---

## 12. go.mod Change Required

Add to `go.mod`:
```
require (
    github.com/bwmarrin/discordgo v0.29.0
    github.com/pion/randutil v0.1.0 // indirect
    github.com/pion/rtp v1.10.1 // indirect
    github.com/pion/webrtc/v3 v3.3.6 // indirect
    layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32 // indirect
)
```

discordgo v0.29.0 brings in two transitive deps:
- `github.com/gorilla/websocket v1.4.2`
- `golang.org/x/crypto` (version from discordgo's go.sum)

Run `go mod tidy` after adding discordgo to resolve the full dependency graph.

---

## 13. Token Configuration

The reference bot reads the token from:
1. `DISCORD_TOKEN` environment variable (priority)
2. `~/.openclaw/openclaw.json` → `channels.discord.token`

The Go implementation should follow the same resolution order. A `getToken()` helper that returns `(string, error)` would be the appropriate abstraction. The `DISCORD_TOKEN` value must be passed to `discordgo.New("Bot " + token)` — the `"Bot "` prefix is mandatory for bot tokens.

---

## 14. Required Discord Bot Permissions

From the reference SKILL.md:
- Bot needs **Guild Voice States** intent enabled in the Discord Developer Portal
- Bot needs **Connect** and **Speak** permissions in the target voice channel

The `IntentsGuildVoiceStates` intent is required to receive `VoiceStateUpdate` events (used by discordgo internally during `ChannelVoiceJoin`). Without it, `ChannelVoiceJoin` will time out.

---

## 15. Silence Frame Specification

Standard Opus silence frame: `[]byte{0xF8, 0xFF, 0xFE}` (3 bytes).

This is a valid Opus packet representing 20ms of silence. Sending 5 of these (100ms total) before calling `vc.Speaking(false)` prevents the Discord gateway from clipping the tail of the audio. This is a widely-documented pattern for discordgo voice bots.

The number 5 from the plan corresponds to 5 × 20ms = 100ms of silence padding.

---

## 16. Context Propagation

The `Play` method accepts `context.Context`. Context cancellation should:
1. Break out of the frame-sending loop
2. Still send the 5 silence frames (to clean up Discord's jitter buffer)
3. Still call `vc.Speaking(false)` and `vc.Disconnect()`

The `goto stop` pattern above handles this, or alternatively a `select` with `ctx.Done()`.

---

## 17. Testing Strategy for internal/discord/

Since discordgo requires a live Discord connection, unit tests must use interfaces and mock substitution. The `VoicePlayer` interface makes the player testable without a real Discord session.

For `channel.go`, tests can use a mock `*discordgo.Session` — but discordgo's Session is a concrete struct, not an interface, so channel detection tests will require either:
- Testing with a pre-populated `State` (inject a `*discordgo.Session` with `State.Guilds` populated)
- Extracting the detection logic to accept `*discordgo.Guild` directly (easier to test)

Recommended test approach for `FindPopulatedVoiceChannel`:
```go
// Extract inner logic to accept Guild directly — then test it without a Session
func findPopulatedVoiceChannelInGuild(guild *discordgo.Guild, channels []*discordgo.Channel) (string, error)
```

---

## 18. Summary Table

| Item | Location | Status |
|------|----------|--------|
| `OpusFrameEncoder` interface | `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go:54` | EXISTS |
| `AudioGenerator` interface | `/Users/jamesprial/code/go-scream/internal/audio/generator.go:6` | EXISTS |
| Encoding constants (960, 3840, 64000) | `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go:12-21` | EXISTS |
| `internal/audio/errors.go` | `/Users/jamesprial/code/go-scream/internal/audio/errors.go` | EXISTS |
| `internal/encoding/encoder.go` sentinel errors | `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go:35-51` | EXISTS |
| Node.js reference bot | `/tmp/scream-reference/scripts/scream.mjs` | EXISTS — fully analyzed |
| discordgo library | `/Users/jamesprial/go/pkg/mod/github.com/bwmarrin/discordgo@v0.29.0` | IN MODULE CACHE — v0.29.0 |
| `internal/discord/` directory | `/Users/jamesprial/code/go-scream/internal/discord/` | EXISTS but EMPTY |
| Pion RTP in ogg.go | `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go` | EXISTS — not needed for Discord player |
| go.mod | `/Users/jamesprial/code/go-scream/go.mod` | discordgo NOT yet listed — must add |
| discordgo airhorn example | `/Users/jamesprial/go/pkg/mod/github.com/bwmarrin/discordgo@v0.29.0/examples/airhorn/main.go` | EXISTS — key voice pattern reference |
