# Code Review: Stage 4 -- Discord Integration

**Verdict: REQUEST_CHANGES**

## Files Reviewed

- `/Users/jamesprial/code/go-scream/internal/discord/errors.go`
- `/Users/jamesprial/code/go-scream/internal/discord/session.go`
- `/Users/jamesprial/code/go-scream/internal/discord/player.go`
- `/Users/jamesprial/code/go-scream/internal/discord/channel.go`
- `/Users/jamesprial/code/go-scream/internal/discord/player_test.go`
- `/Users/jamesprial/code/go-scream/internal/discord/channel_test.go`

## Summary

The Discord integration layer is well-architected. The interface abstractions (`Session`, `VoiceConn`, `VoicePlayer`) cleanly separate the discordgo dependency from business logic, enabling thorough unit testing without a live Discord connection. Error handling follows the project's sentinel-error-with-%w-wrapping pattern established in `internal/encoding` and `internal/audio`. The mock infrastructure is carefully designed with proper synchronization and a collector goroutine pattern for capturing sent frames. The test suite is comprehensive, covering normal playback, edge cases, error paths, context cancellation, and speaking/disconnect protocol verification.

One issue must be addressed before approval.

---

## Required Changes

### 1. `sendSilence` can block forever if the opus channel is full or closed

**File:** `/Users/jamesprial/code/go-scream/internal/discord/player.go`, lines 102-106

```go
func sendSilence(opusSend chan<- []byte) {
	for i := 0; i < SilenceFrameCount; i++ {
		opusSend <- SilenceFrame
	}
}
```

This function performs an unconditional blocking send to `opusSend`. There are two concerns:

**(a) Blocking indefinitely when Discord's send buffer is full.** The real `discordgo.VoiceConnection.OpusSend` channel has a finite buffer. If the Discord gateway is slow, disconnected, or the connection is in a degraded state, these sends could block indefinitely, preventing `Play` from returning and preventing `Disconnect` (via defer) from firing. This creates a goroutine leak where the Play caller hangs forever.

**(b) Context is not threaded through.** When `sendSilence` is called from the context-cancellation path (lines 74 and 84), the context is already cancelled, yet the function does not check it. If the opus channel is full, the caller's goroutine will hang even though cancellation was requested.

**Recommended fix:** Accept a context parameter and use a select to respect cancellation or a timeout:

```go
func sendSilence(ctx context.Context, opusSend chan<- []byte) {
	for i := 0; i < SilenceFrameCount; i++ {
		select {
		case opusSend <- SilenceFrame:
		case <-ctx.Done():
			return
		}
	}
}
```

For the normal-completion path (line 92), `context.Background()` or the original context can be passed. For the cancellation paths (lines 74 and 84), pass a short timeout context derived from `context.Background()` (not the already-cancelled `ctx`) so silence frames are sent on a best-effort basis:

```go
// In cancellation paths:
silenceCtx, silenceCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
defer silenceCancel()
sendSilence(silenceCtx, opusSend)
```

This prevents the goroutine from hanging indefinitely while still making a best-effort attempt to send silence frames for a clean audio cutoff.

---

## Observations (non-blocking, for awareness)

### `SilenceFrame` is a mutable package-level `var`

**File:** `/Users/jamesprial/code/go-scream/internal/discord/player.go`, line 9

```go
var SilenceFrame = []byte{0xF8, 0xFF, 0xFE}
```

Since this is a `var` holding a `[]byte` slice, any caller could mutate it (e.g., `discord.SilenceFrame[0] = 0x00`), which would corrupt silence frames for all subsequent playback. Go does not support `const` slices, so this is a known Go limitation. Two options to harden this:

- Make it unexported (`silenceFrame`) since it is only used internally by `sendSilence`, and expose the value through a function if external consumers need it.
- Document that it must not be modified.

This is not blocking because the package is internal and the current usage is safe, but it is worth noting for future maintainability.

### `DiscordGoSession.GuildVoiceStates` accesses `d.S.State.Guild` without nil-checking `d.S.State`

**File:** `/Users/jamesprial/code/go-scream/internal/discord/session.go`, line 42

```go
guild, err := d.S.State.Guild(guildID)
```

If `d.S.State` is nil (e.g., if state tracking is disabled on the discordgo session), this will panic with a nil pointer dereference. In practice, discordgo initializes `State` by default in `discordgo.New()`, so this is unlikely. However, adding a nil guard would make the adapter more defensive:

```go
if d.S.State == nil {
    return nil, errors.New("discord: session state tracking is disabled")
}
```

Not blocking because the adapter wraps a standard discordgo session which always has `State` initialized, but worth considering for robustness.

### `FindPopulatedChannel` returns the first non-bot voice state, not the most populated channel

**File:** `/Users/jamesprial/code/go-scream/internal/discord/channel.go`, lines 20-23

```go
for _, vs := range voiceStates {
    if vs.ChannelID != "" && vs.UserID != botUserID {
        return vs.ChannelID, nil
    }
}
```

The function name says "populated" but the implementation returns the first channel with any non-bot user, not the channel with the most users. This is fine for a scream bot (where any channel with a human is a valid target), and the doc comment accurately describes the behavior ("first voice channel containing at least one non-bot user"). The name is slightly misleading but acceptable. Just noting for awareness.

### `VoiceState` struct duplicates discordgo fields

**File:** `/Users/jamesprial/code/go-scream/internal/discord/session.go`, lines 19-24

Defining a custom `VoiceState` struct decouples the package from discordgo's type, which is good for testability. This is the correct pattern.

### Mock infrastructure is well-designed

The `mockVoiceConn` with its collector goroutine pattern, `drainAndCollect` helper, and proper mutex synchronization is thoughtfully built. The `newMockVoiceConn()` constructor correctly initializes the background collector. The `drainAndCollect` method with a 5-second timeout prevents tests from hanging on bugs. This is high-quality test infrastructure.

### Test coverage is thorough

The test suite covers:
- Normal playback with varying frame counts (0, 1, 10)
- Speaking protocol verification (true at start, false at end)
- Silence frame count and content verification
- Disconnect called on success and on error
- Join parameter forwarding (guild ID, channel ID, mute=false, deaf=true)
- All three validation errors (empty guild ID, empty channel ID, nil frames)
- Join failure error wrapping
- Speaking failure error wrapping
- Pre-cancelled context before join
- Mid-playback context cancellation
- Table-driven validation error tests
- `FindPopulatedChannel` with single user, multiple users, multiple channels, bot-only, bot+user, bot-in-different-channel, empty states, empty guild ID, state retrieval error
- Table-driven `FindPopulatedChannel` cases
- Benchmarks for both `Play` and `FindPopulatedChannel`

### Benchmark in `channel_test.go` has a minor string construction concern

**File:** `/Users/jamesprial/code/go-scream/internal/discord/channel_test.go`, lines 228-230

```go
uid := "user-" + string(rune('A'+i%26)) + string(rune('0'+i/26))
```

For `i >= 260`, `i/26 >= 10`, and `string(rune('0'+10))` produces `':'` (the character after `'9'` in ASCII). This is harmless for a benchmark (the IDs just need to be distinct, and they are since `i` varies), but it is slightly surprising if someone reads the output. Using `fmt.Sprintf("user-%d", i)` would be clearer. Not blocking.

---

## Checklist

**Code Quality:**
- [x] All exported items have documentation (`Session`, `VoiceConn`, `VoiceState`, `DiscordGoSession`, `DiscordGoVoiceConn`, `VoicePlayer`, `DiscordPlayer`, `NewDiscordPlayer`, `Play`, `FindPopulatedChannel`, `SilenceFrame`, `SilenceFrameCount`, all sentinel errors)
- [x] Error handling follows patterns -- sentinel errors in `errors.go`, `%w` wrapping with dual-sentinel pattern (`fmt.Errorf("%w: %w", ...)`)
- [x] Nil safety guards present -- empty string checks for guild/channel IDs, nil check for frames channel, pre-cancelled context check
- [x] Table tests structured correctly -- `TestDiscordPlayer_Play_ValidationErrors` and `TestFindPopulatedChannel_Cases` use proper table-driven patterns with `t.Run`
- [x] Code is readable and well-organized -- clean separation across four files (errors, session/interfaces, player, channel)
- [x] Naming conventions followed -- `DiscordPlayer`, `VoicePlayer`, `FindPopulatedChannel`, `ErrVoiceJoinFailed` all follow Go conventions
- [ ] No obvious logic errors or edge case gaps -- `sendSilence` can block indefinitely (see Required Changes)

**Interface Design:**
- [x] `Session` and `VoiceConn` interfaces are minimal (only methods actually used)
- [x] `VoicePlayer` interface has a single method (`Play`) with clean signature
- [x] Compile-time interface check: `var _ VoicePlayer = (*DiscordPlayer)(nil)`
- [x] Adapter types (`DiscordGoSession`, `DiscordGoVoiceConn`) cleanly bridge to discordgo

**Pattern Consistency with Existing Packages:**
- [x] Sentinel errors follow `errors.New("package: description")` pattern from `internal/encoding`
- [x] Error wrapping uses `fmt.Errorf("%w: %w", sentinel, underlying)` consistently
- [x] Interface-based design matches `AudioGenerator` and `OpusFrameEncoder` patterns
- [x] Constructor follows `NewXxx` naming convention
- [x] Resource cleanup uses `defer` (`defer vc.Disconnect()`)

**Test Coverage Assessment:**
- [x] All exported functions have tests (`NewDiscordPlayer`, `Play`, `FindPopulatedChannel`)
- [x] All sentinel errors tested for reachability
- [x] Error wrapping verified with `errors.Is`
- [x] Edge cases covered (0 frames, 1 frame, pre-cancelled context, mid-playback cancellation)
- [x] Protocol verification (speaking true/false, disconnect always called)
- [x] Mock infrastructure has proper synchronization (mutexes, collector goroutine)
- [x] Both individual and table-driven tests present
- [x] Benchmarks included for both `Play` and `FindPopulatedChannel`
