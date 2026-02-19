# Code Review: Stage 5b -- Service Orchestrator

**Files reviewed:**
- `/Users/jamesprial/code/go-scream/internal/scream/errors.go`
- `/Users/jamesprial/code/go-scream/internal/scream/service.go`
- `/Users/jamesprial/code/go-scream/internal/scream/resolve.go`
- `/Users/jamesprial/code/go-scream/internal/scream/service_test.go`

**Context files:**
- `/Users/jamesprial/code/go-scream/internal/discord/player.go`
- `/Users/jamesprial/code/go-scream/internal/config/errors.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go`

---

## Verdict: APPROVE

The service orchestrator is well-designed, with clean dependency injection, correct error handling, proper nil-safety via reflect normalization, and comprehensive test coverage. The code is consistent with established codebase patterns. Minor observations are noted below but none warrant blocking.

---

## Detailed Review

### 1. Error Handling

**Sentinel errors (`errors.go`):**
All five sentinel errors follow the established `"scream: ..."` prefix convention, matching the pattern used in `config/errors.go` (`"config: ..."`) and `encoding/encoder.go` (`"encoding: ..."`). Each is documented with a godoc comment. No issues.

**Error wrapping (`service.go`):**
All wrapping uses the `fmt.Errorf("%w: %w", SentinelErr, causeErr)` dual-%w pattern consistently:
- Line 75: `fmt.Errorf("%w: %w", ErrGenerateFailed, err)` in `Play()`
- Line 85: `fmt.Errorf("%w: %w", ErrEncodeFailed, encErr)` in DryRun path
- Line 94: `fmt.Errorf("%w: %w", ErrPlayFailed, playErr)` in `Play()`
- Line 97: `fmt.Errorf("%w: %w", ErrEncodeFailed, encErr)` in `Play()`
- Lines 117, 121: Same pattern in `Generate()`

This allows callers to use `errors.Is()` against both the sentinel and the underlying cause. Consistent with `discord/player.go` lines 59, 66, 100 which use the same dual-%w pattern.

**Error ordering in `Play()` (lines 93-98):**
After `player.Play()` completes, the code reads `encErr` from the error channel and then prioritizes `playErr` over `encErr`. This is the correct ordering -- a player failure is user-visible and should take precedence. The encoder error channel is still drained regardless, preventing goroutine leaks.

**`resolveParams` error passthrough:**
`ErrUnknownPreset` is returned directly (not wrapped), which is appropriate because it is already a sentinel error from this package with no additional context needed.

### 2. Nil Safety and Typed-Nil Normalization

**`NewServiceWithDeps` (lines 36-42):**
The reflect-based typed-nil normalization is correct and necessary. When a caller passes a concrete `*DiscordPlayer` pointer that happens to be nil, a naive `player == nil` comparison would return false due to Go's interface boxing behavior. The reflect guard:

```go
if player != nil {
    v := reflect.ValueOf(player)
    if v.Kind() == reflect.Ptr && v.IsNil() {
        player = nil
    }
}
```

This correctly handles the case, and the outer `player != nil` guard prevents a panic from `reflect.ValueOf(nil)`. The function comment correctly documents that it never returns nil.

**Observation:** The normalization only applies to the `player` parameter. The `gen`, `fileEnc`, and `frameEnc` parameters are not normalized. This is a deliberate design choice -- those dependencies are required (calling methods on nil would panic), while `player` is explicitly optional for DryRun/Generate-only modes. The asymmetry is appropriate.

### 3. DryRun Logic

**`Play()` DryRun path (lines 80-88):**
When `DryRun` is true, the code drains the frame channel (`for range frameCh`) and then reads the encoder error channel. This correctly:
- Consumes all frames so the encoder goroutine is not leaked
- Checks the encoder error so encoding failures are not silently swallowed
- Skips the player entirely

**DryRun nil player guard (line 60):**
The `!s.cfg.DryRun && s.player == nil` check allows `Play()` to proceed with a nil player when DryRun is set. Correct.

### 4. Context Propagation

**`Play()` (line 64):** Early `ctx.Err()` check prevents unnecessary work if context is already cancelled.

**`Play()` (line 90):** Context is passed to `s.player.Play()`, which handles cancellation during voice streaming (as seen in `discord/player.go`).

**`Generate()` (line 106):** Early `ctx.Err()` check present.

**Observation:** Neither `Play()` nor `Generate()` pass context to `s.generator.Generate()` or `s.fileEnc.Encode()`. The `audio.AudioGenerator` and `encoding.FileEncoder` interfaces do not accept context parameters, so this is an interface-level design decision, not a bug in this layer. The early `ctx.Err()` check provides a cancellation point before the potentially expensive generation step.

### 5. `resolveParams` Correctness

**`resolve.go` (lines 13-31):**
- When `cfg.Preset` is non-empty, it looks up the preset via `audio.GetPreset()` and returns `ErrUnknownPreset` on miss.
- When `cfg.Preset` is empty, it calls `audio.Randomize(0)` which seeds from `time.Now().UnixNano()`.
- A positive `cfg.Duration` overrides the duration from either preset or random params.
- Zero or negative `cfg.Duration` leaves the preset/random duration unchanged.

This is correct. The function is documented with a clear godoc comment explaining all three cases.

### 6. `Close()` and `closer` Field

The `closer` field is not set via `NewServiceWithDeps` -- tests set it directly (`svc.closer = mc`). This suggests the production wiring happens elsewhere (likely a `New()` or builder function not in scope). The `Close()` implementation correctly handles nil closer (returns nil) and delegates to the closer otherwise. No resource leak risk.

### 7. `ListPresets()`

Clean utility function. Converts `[]audio.PresetName` to `[]string` to avoid leaking the `audio.PresetName` type through the service API. Returns a new slice each time (no shared mutable state).

### 8. API Design

The service correctly separates `Play()` (voice playback) from `Generate()` (file output). Each method uses only the dependencies it needs -- `Play()` uses `frameEnc` + `player`, while `Generate()` uses `fileEnc`. The `Service` struct holds both sets of dependencies, which is reasonable for a single-service design.

**Observation on `channelID`:** `Play()` does not validate `channelID == ""`. This is intentional -- the downstream `discord.DiscordPlayer.Play()` (line 45 of `player.go`) validates `channelID` and returns `ErrEmptyChannelID`. The service layer validates only what it owns (guild ID, player presence) and delegates channel-level validation to the player. This is a clean separation of concerns.

### 9. Test Quality

**Coverage breadth:** 40+ test functions and 3 benchmarks covering:
- Constructor behavior (non-nil, nil player, config storage)
- `Play()` happy path, validation, all error paths (generator, player, encoder, context, unknown preset)
- DryRun mode (skips player, nil player allowed, encoder errors surfaced)
- `Generate()` happy path (OGG and WAV), all error paths
- `Close()` with closer, without closer, error propagation, double-close safety
- `ListPresets()` count, expected names, no duplicates, determinism
- `resolveParams` via behavioral tests (duration override, empty preset uses Randomize)
- Error wrapping verification (both sentinel and original cause are unwrappable)

**Mock design:** All mocks use `sync.Mutex` for thread safety. The `mockFrameEncoder` correctly runs in a goroutine matching the real `EncodeFrames` channel contract. The `mockPlayer` drains frames to prevent goroutine leaks. Well-structured.

**Table-driven tests:** Used appropriately for validation cases (`Test_Play_Validation`) and sentinel error existence checks (`Test_SentinelErrors_Exist`). Individual test functions are used where setup differs significantly, which is appropriate.

**Minor test gap:** There is no explicit test for the frame encoder returning an error during non-DryRun `Play()` (where `encErr != nil && playErr == nil`). The `mockFrameEncoder.encErr` field exists but is only exercised indirectly. However, the DryRun encoder error test (`Test_Play_DryRun_SkipsPlayer` does not test this -- it tests happy path) and the code path at line 97 is structurally identical to the DryRun path at line 85. This is a minor observation, not a blocking concern.

---

## Checklist

- [x] All exported items have documentation
- [x] Error handling follows dual-%w wrapping pattern
- [x] Nil safety guards present (reflect normalization for typed-nil, nil closer check)
- [x] Table tests structured correctly where used
- [x] Code is readable and well-organized (3-file split: errors, service, resolve)
- [x] Naming conventions followed (Go idioms, unexported helper `resolveParams`)
- [x] No logic errors or edge case gaps found
- [x] Consistent with codebase patterns (error format, interface usage, context checks)
