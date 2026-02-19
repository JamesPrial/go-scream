# Design Review: Stage 4 -- Discord Integration

**Package:** `internal/discord`
**Files reviewed:**
- `/Users/jamesprial/code/go-scream/internal/discord/errors.go`
- `/Users/jamesprial/code/go-scream/internal/discord/session.go`
- `/Users/jamesprial/code/go-scream/internal/discord/player.go`
- `/Users/jamesprial/code/go-scream/internal/discord/channel.go`
- `/Users/jamesprial/code/go-scream/internal/discord/player_test.go`
- `/Users/jamesprial/code/go-scream/internal/discord/channel_test.go`

**Compared against:**
- `/Users/jamesprial/code/go-scream/internal/encoding/` (encoder.go, opus.go, ogg.go, wav.go)
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/` (errors.go, command.go, generator.go)
- `/Users/jamesprial/code/go-scream/internal/audio/` (generator.go, errors.go)

---

## 1. Package Organization and Structure

The discord package is well-organized with a clean four-file layout:

| File | Responsibility |
|------|---------------|
| `errors.go` | Sentinel error definitions |
| `session.go` | Interface definitions and discordgo adapter types |
| `channel.go` | Channel discovery (standalone function) |
| `player.go` | Voice playback orchestration |

This mirrors the project convention seen in `internal/audio/ffmpeg/` (separate errors.go, command.go, generator.go) and `internal/encoding/` (separate encoder.go per concern). The separation of concerns is appropriate -- no file is overloaded.

**Verdict: Good.** File layout is consistent with existing packages.

---

## 2. Interface Design and Exported API Surface

### Session and VoiceConn interfaces (session.go)

```go
type Session interface {
    ChannelVoiceJoin(guildID, channelID string, mute, deaf bool) (VoiceConn, error)
    GuildVoiceStates(guildID string) ([]*VoiceState, error)
}

type VoiceConn interface {
    Speaking(speaking bool) error
    OpusSendChannel() chan<- []byte
    Disconnect() error
    IsReady() bool
}
```

The interface design follows the Go idiom of defining interfaces at the consumer side with minimal surface area. `Session` has exactly two methods (one for joining, one for state queries), and `VoiceConn` has exactly four methods covering the lifecycle needs. This is consistent with how `internal/encoding/` defines `OpusFrameEncoder` and `FileEncoder` -- small, focused interfaces at the abstraction boundary.

The `VoiceState` struct is a clean domain model that decouples from `discordgo.VoiceState`, which is the right call -- it prevents discordgo types from leaking across package boundaries.

**Observation:** `IsReady()` is defined on `VoiceConn` but is never called by any code in the discord package. It appears to exist for completeness of the adapter. This is acceptable for a production adapter that external consumers may use, but if it is purely internal it could be trimmed. Minor point -- not blocking.

### VoicePlayer interface (player.go)

```go
type VoicePlayer interface {
    Play(ctx context.Context, guildID, channelID string, frames <-chan []byte) error
}
```

Clean single-method interface. Consistent with the project pattern of `AudioGenerator` (single `Generate` method), `OpusFrameEncoder` (single `EncodeFrames` method), and `FileEncoder` (single `Encode` method). Context-first parameter ordering follows standard Go conventions.

### FindPopulatedChannel (channel.go)

```go
func FindPopulatedChannel(session Session, guildID, botUserID string) (string, error)
```

This is a package-level function rather than a method, which is the right design choice -- it does not need to hold state, and it operates on the `Session` interface. This is consistent with how `ffmpeg.BuildArgs` is a standalone function operating on params.

**Verdict: Good.** API surface is minimal, idiomatic, and consistent.

---

## 3. Adapter Pattern (discordgo Wrapping)

The `DiscordGoSession` and `DiscordGoVoiceConn` adapter types in `session.go` cleanly wrap the discordgo types:

```go
type DiscordGoSession struct {
    S *discordgo.Session
}

type DiscordGoVoiceConn struct {
    VC *discordgo.VoiceConnection
}
```

The adapters are thin wrappers with no business logic, which is correct. The conversion from `discordgo.VoiceState` to the package-local `VoiceState` type in `GuildVoiceStates` properly isolates the domain model.

**Note:** The exported field names `S` and `VC` are terse but acceptable for adapter types that are constructed at the application boundary and not widely used. This is a style preference, not a defect.

**Verdict: Good.** Adapter pattern is well-structured and testable.

---

## 4. Error Handling Patterns

### Sentinel errors (errors.go)

```go
var (
    ErrVoiceJoinFailed    = errors.New("discord: failed to join voice channel")
    ErrSpeakingFailed     = errors.New("discord: failed to set speaking state")
    ErrNoPopulatedChannel = errors.New("discord: no populated voice channel found")
    ErrGuildStateFailed   = errors.New("discord: failed to retrieve guild state")
    ErrEmptyGuildID       = errors.New("discord: guild ID must not be empty")
    ErrEmptyChannelID     = errors.New("discord: channel ID must not be empty")
    ErrNilFrameChannel    = errors.New("discord: frame channel must not be nil")
)
```

All sentinel errors use the `discord:` prefix, matching the convention in `internal/encoding/` (`encoding:` prefix) and `internal/audio/ffmpeg/` (`ffmpeg:` prefix). The grouped `var` block with doc comment matches the pattern in `internal/encoding/encoder.go`.

### Error wrapping

```go
return fmt.Errorf("%w: %w", ErrVoiceJoinFailed, err)
return fmt.Errorf("%w: %w", ErrSpeakingFailed, err)
return fmt.Errorf("%w: %w", ErrGuildStateFailed, err)
```

All error wrapping consistently uses the double `%w` pattern with sentinel first, underlying error second. This matches the existing codebase pattern seen throughout:
- `encoding/opus.go`: `fmt.Errorf("%w: creating encoder: %w", ErrOpusEncode, err)`
- `ffmpeg/generator.go`: `fmt.Errorf("%w: %w", ErrFFmpegNotFound, err)`

**Verdict: Good.** Error patterns are consistent with the rest of the project.

---

## 5. Documentation Completeness

Every exported item has a doc comment:
- Package-level doc comment on `errors.go` (line 1): "Package discord provides Discord voice channel integration for audio playback."
- All 7 sentinel errors are documented in the block comment
- `Session`, `VoiceConn`, `VoiceState` interfaces/struct: documented
- `DiscordGoSession`, `DiscordGoVoiceConn`: documented
- All adapter methods: documented
- `VoicePlayer` interface: documented
- `DiscordPlayer` struct and its constructor: documented
- `Play` method: thorough doc comment explaining join, stream, silence, and error behavior
- `FindPopulatedChannel`: thorough doc comment listing all error conditions
- `SilenceFrame`, `SilenceFrameCount`: documented
- `sendSilence`: documented (unexported, but still has comment -- consistent quality)

The `Play` method documentation at lines 32-37 of `player.go` is particularly strong -- it documents the full behavior contract and all error conditions, which matches the documentation style in `encoding/opus.go` for `EncodeFrames`.

**Verdict: Good.** Documentation is thorough and consistent.

---

## 6. Constructor Patterns

```go
func NewDiscordPlayer(session Session) *DiscordPlayer {
    return &DiscordPlayer{session: session}
}
```

This follows the codebase convention:
- `encoding.NewGopusFrameEncoder() *GopusFrameEncoder`
- `encoding.NewOGGEncoder() *OGGEncoder`
- `encoding.NewWAVEncoder() *WAVEncoder`
- `ffmpeg.NewFFmpegGeneratorWithPath(path string) *FFmpegGenerator`

Returns concrete type (not interface), consistent throughout the codebase.

**Verdict: Good.** Constructor pattern matches project conventions.

---

## 7. Core Design Decisions

### Double-select pattern (player.go, lines 70-89)

```go
for {
    select {
    case <-ctx.Done():
        sendSilence(opusSend)
        vc.Speaking(false)
        return ctx.Err()
    case frame, ok := <-frames:
        if !ok {
            break loop
        }
        select {
        case opusSend <- frame:
        case <-ctx.Done():
            sendSilence(opusSend)
            vc.Speaking(false)
            return ctx.Err()
        }
    }
}
```

This is a well-known Go pattern for ensuring context cancellation is respected even when a channel send could block. The outer select handles cancellation between frame reads, and the inner select handles cancellation during the send to `opusSend`. This is correct and necessary -- without the inner select, a blocked `opusSend <- frame` would prevent the goroutine from responding to cancellation.

The nolint comment `//nolint:errcheck // best-effort` for `vc.Speaking(false)` in cancellation paths is appropriate -- when shutting down due to cancellation, propagating a speaking-state error would obscure the actual cancellation error.

### sendSilence function (player.go, lines 102-106)

```go
func sendSilence(opusSend chan<- []byte) {
    for i := 0; i < SilenceFrameCount; i++ {
        opusSend <- SilenceFrame
    }
}
```

**Issue (minor):** `sendSilence` sends to `opusSend` without context awareness. If the `opusSend` channel is full or the receiving end is slow, this could block indefinitely. In the cancellation paths of `Play`, the context is already done, but `sendSilence` does not check it. In practice, discordgo's `OpusSend` channel is buffered and consumed by a separate goroutine, so this is unlikely to be a real problem, but it is worth noting for correctness.

Additionally, all cancellation paths send the same `SilenceFrame` slice reference. Since slices are reference types, this is safe as long as nothing mutates `SilenceFrame`. The `var SilenceFrame = []byte{...}` declaration is mutable in theory. A caller could do `discord.SilenceFrame[0] = 0x00`. This is a minor exposure but consistent with how Go standard library packages expose similar byte slices. Not blocking.

### Resource cleanup via defer (player.go, line 61)

```go
vc, err := p.session.ChannelVoiceJoin(...)
if err != nil { return ... }
defer vc.Disconnect()
```

Disconnect is deferred immediately after successful join, ensuring cleanup on all exit paths. This is correct and follows Go best practices. The test `TestDiscordPlayer_Play_DisconnectOnError` explicitly verifies this behavior.

---

## 8. Test Quality Assessment

### Mock design (player_test.go)

The `mockVoiceConn` with its background collector goroutine is well-designed. It solves the problem of collecting frames from a channel that `Play` sends to, without blocking `Play`. The `drainAndCollect` method with a 5-second timeout prevents tests from hanging.

The mock uses `sync.Mutex` for thread safety, which is correct since `Play` sends frames from its goroutine while the collector runs concurrently.

### Test coverage

**player_test.go** covers:
- Constructor validity
- Constant verification (SilenceFrame content, SilenceFrameCount value)
- Normal playback with 10 frames, 1 frame, 0 frames (empty channel)
- Speaking protocol verification (true at start, false at end)
- Silence frame verification (exactly 5 after data)
- Disconnect verification (called on success and on error)
- Join parameter verification (guildID, channelID, mute=false, deaf=true)
- Validation errors (empty guild ID, empty channel ID, nil frames)
- Join failure error wrapping
- Speaking failure error wrapping
- Pre-cancelled context
- Mid-playback context cancellation
- Table-driven validation error tests
- Benchmark

**channel_test.go** covers:
- Single user, multiple users, multiple channels
- Bot-only (returns ErrNoPopulatedChannel)
- Bot + user in same channel
- Bot in one channel, user in another
- Empty voice states
- Empty guild ID validation
- State retrieval error
- Table-driven combined tests
- Benchmark

### Table-driven tests

Both files include table-driven tests (`TestDiscordPlayer_Play_ValidationErrors` and `TestFindPopulatedChannel_Cases`) that follow the standard Go pattern with `t.Run` subtests. The test struct fields are well-named, and the table structure matches the convention seen elsewhere in the codebase.

**Observation:** There is some overlap between the individual test functions and the table-driven tests (e.g., `TestFindPopulatedChannel_EmptyGuildID` also appears as a case in `TestFindPopulatedChannel_Cases`). This is not harmful -- the individual tests serve as clear documentation of specific behaviors while the table tests provide a compact summary. This is a reasonable approach.

### Missing test coverage

- No test for `DiscordGoSession` or `DiscordGoVoiceConn` adapter types. These are thin wrappers around discordgo, so unit testing them requires a real discordgo session or additional mocking layers. Excluding them from unit tests is a pragmatic choice -- they would be covered by integration tests.
- No test for `Speaking(false)` returning an error on the normal (non-cancellation) completion path (line 94-96 of player.go). This would verify that `ErrSpeakingFailed` is returned when the post-playback `Speaking(false)` call fails. This is a minor gap.

---

## 9. Naming Conventions

All names follow Go conventions:
- Types: `DiscordPlayer`, `DiscordGoSession`, `VoiceConn` (PascalCase)
- Functions: `NewDiscordPlayer`, `FindPopulatedChannel` (PascalCase exported)
- Variables: `SilenceFrame`, `SilenceFrameCount` (PascalCase exported constants/vars)
- Private: `sendSilence`, `session` field (camelCase)
- Errors: `Err` prefix with descriptive suffix

The interface names `Session`, `VoiceConn`, `VoicePlayer` are noun-based, which is correct for Go interfaces that represent capabilities rather than single-method behaviors (-er convention).

---

## 10. Consistency Summary

| Pattern | Existing Codebase | Discord Package | Match? |
|---------|-------------------|-----------------|--------|
| Sentinel errors in `errors.go` | ffmpeg/errors.go, encoding/encoder.go | errors.go | Yes |
| Error prefix (`pkg:`) | `encoding:`, `ffmpeg:` | `discord:` | Yes |
| Error wrapping with `%w` | Throughout | Throughout | Yes |
| Interface at consumer | encoding.OpusFrameEncoder | discord.Session, VoiceConn | Yes |
| Constructor returns concrete | NewGopusFrameEncoder, NewWAVEncoder | NewDiscordPlayer | Yes |
| Compile-time interface check | ffmpeg/generator.go | player.go line 25 | Yes |
| Package doc comment | All packages | errors.go line 1 | Yes |
| Defer for cleanup | encoding/ogg.go (defer oggWriter.Close) | player.go (defer vc.Disconnect) | Yes |
| Table-driven tests with t.Run | Across test files | player_test.go, channel_test.go | Yes |

---

## 11. Issues Found

### Minor issues (not blocking)

1. **`sendSilence` could block indefinitely if `opusSend` is full.** In practice unlikely with discordgo's buffered channel, but adding a context-aware send or a select with a timeout would make it more robust. Low risk given the known consumer (discordgo).

2. **`SilenceFrame` is a mutable exported `var`.** A determined caller could mutate it. Consider making it a function that returns a copy, or documenting that it must not be modified. This is a common Go pattern tradeoff and not blocking.

3. **No test for `Speaking(false)` error on normal completion path.** The error path at player.go line 94-96 is untested. Adding a test where `speakingErr` is set after `Speaking(true)` succeeds (or using a mock that fails only on the second call) would close this gap.

4. **`IsReady()` on `VoiceConn` is unused within the package.** It exists on the interface but is never called by `Play` or `FindPopulatedChannel`. If it is part of the planned API for future stages, this is fine. Otherwise it could be trimmed.

None of these issues are architectural or correctness problems that would block approval.

---

## Verdict

**APPROVE**

The discord package demonstrates strong design quality across all review criteria. The adapter pattern cleanly isolates the discordgo dependency behind testable interfaces. Error handling, documentation, naming, constructors, and test structure are all consistent with the established patterns in the `internal/encoding/` and `internal/audio/ffmpeg/` packages. The double-select pattern for context cancellation is correct. The test suite is thorough with good edge case coverage, table-driven tests, and well-designed mocks. The minor issues noted above are suggestions for future hardening, not blockers.
