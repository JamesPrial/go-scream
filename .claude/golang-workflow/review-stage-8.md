# Review: Stage 8 -- Deduplicate Service Wiring

**Reviewer:** Go Reviewer (automated)
**Date:** 2026-02-19
**Files reviewed:**
- `/Users/jamesprial/code/go-scream/internal/app/wire.go` (NEW)
- `/Users/jamesprial/code/go-scream/internal/app/wire_test.go` (NEW)
- `/Users/jamesprial/code/go-scream/cmd/scream/service.go` (MODIFIED)
- `/Users/jamesprial/code/go-scream/cmd/skill/main.go` (MODIFIED)

---

## 1. Deduplication Quality

**Both cmd binaries now use `app.*` functions instead of inline wiring.** Verified:

- `cmd/scream/service.go` (line 17-23) calls `app.NewGenerator`, `app.NewFileEncoder`, and `app.NewDiscordDeps` instead of constructing these inline.
- `cmd/skill/main.go` (lines 106-115) uses the same three `app.*` functions.
- The only thing that remains "inline" in each binary is the `encoding.NewGopusFrameEncoder()` call, which is a single constructor with no branching logic. This is acceptable -- extracting a one-liner with no conditional logic would add indirection without deduplication value.

**No wiring logic duplicated between the two binaries.** The generator-selection switch (`native` vs `ffmpeg`), file-encoder selection (`ogg` vs `wav`), and Discord session setup (create, open, wrap, build player) all live exclusively in `wire.go`.

**Assessment:** Pass -- clean extraction.

## 2. Behavior Preservation

### Generator selection
`NewGenerator` in `wire.go` (lines 32-41) checks `backend == backendFFmpeg` and returns `ffmpeg.NewGenerator()` or `native.NewGenerator()`. This matches the prior inline logic exactly.

### File encoder selection
`NewFileEncoder` in `wire.go` (lines 46-51) checks `format == formatWAV` and returns `encoding.NewWAVEncoder()` or `encoding.NewOGGEncoder()`. This matches prior behavior and the doc comment correctly states it never returns nil.

### Discord session creation
`NewDiscordDeps` in `wire.go` (lines 57-68) creates the session with the `"Bot "` prefix, opens it, wraps in `GoSession`, and builds a `Player`. The returned `io.Closer` is the raw `*discordgo.Session`, which implements `Close()` to cleanly shut down the WebSocket. This matches the prior inline pattern.

**Assessment:** Pass -- identical behavior.

## 3. Import Cycle Analysis

Verified with grep: `internal/app` is only imported by `cmd/scream/service.go` and `cmd/skill/main.go`. None of the downstream packages (`internal/audio`, `internal/encoding`, `internal/discord`, `internal/scream`, `internal/config`) import `internal/app`.

The `internal/app` package imports:
- `internal/audio` (Generator interface)
- `internal/audio/ffmpeg` (ffmpeg.NewGenerator)
- `internal/audio/native` (native.NewGenerator)
- `internal/discord` (GoSession, NewPlayer, VoicePlayer)
- `internal/encoding` (NewOGGEncoder, NewWAVEncoder, FileEncoder)

No cycles exist.

**Assessment:** Pass.

## 4. Error Handling

### `NewGenerator`
Errors from `ffmpeg.NewGenerator()` are propagated directly. For the `native` path, `native.NewGenerator()` does not return an error. Clean.

### `NewFileEncoder`
No error return -- returns a valid encoder unconditionally. The doc comment "NewFileEncoder never returns nil" is correct and helpful.

### `NewDiscordDeps`
Errors are wrapped with `%w` format verb and descriptive context:
- `"failed to create discord session: %w"` (line 60)
- `"failed to open discord session: %w"` (line 63)

**Potential issue (minor):** If `session.Open()` succeeds but `discord.NewPlayer(sess)` were to fail (currently it cannot -- `NewPlayer` never returns an error), the opened session would leak. However, since `NewPlayer` is infallible (returns `*Player`, no error), this is not a real problem today. If `NewPlayer` ever gains error handling in the future, cleanup of the opened session would need to be added here. This is a defensive observation, not a blocking concern.

### Caller-side cleanup
- `cmd/scream/play.go` (lines 66-72): Defers `closer.Close()` with a nil guard, logs warning to stderr. Matches the user preference for CLI code.
- `cmd/scream/generate.go` (lines 50-55): Same pattern. Good.
- `cmd/skill/main.go` (lines 120-124): Defers `sessionCloser.Close()`, logs warning to stderr. Good.

**Assessment:** Pass.

## 5. Code Quality

### Documentation
All three exported functions in `wire.go` have complete doc comments describing parameters, return values, and error conditions. The package-level comment on line 1-3 clearly describes the purpose.

### Constants
The `backendFFmpeg` and `formatWAV` constants are unexported and include comments explaining they mirror `config.BackendFFmpeg` and `config.FormatWAV` without importing the config package. This avoids a dependency on `internal/config` from `internal/app`, which is a sound design choice -- `wire.go` depends on the audio/encoding/discord packages but does not need to know about config types. The callers perform the `string(cfg.Backend)` conversion.

### Naming
- `NewGenerator`, `NewFileEncoder`, `NewDiscordDeps` -- clear, idiomatic constructor names.
- `wire.go` -- conventional name for dependency wiring.

### No `//nolint` directives
None present. Compliant with user preferences.

**Assessment:** Pass.

## 6. Test Quality

### Coverage of `NewGenerator`
- `TestNewGenerator_NativeBackend`: Verifies native path returns non-nil generator, nil error.
- `TestNewGenerator_FFmpegBackend_Available`: Skips if ffmpeg not on PATH. Tests happy path.
- `TestNewGenerator_FFmpegBackend_NotAvailable`: Verifies the sentinel error exists (cannot force the error path when ffmpeg is present -- acceptable pragmatic choice).
- `TestNewGenerator_UnknownBackend_FallsBackToNative`: Table-driven with 5 edge cases including empty, typo, and case variations. Good coverage.
- `TestNewGenerator_TableDriven`: Consolidated table with `skipNoFFm` flag for conditional skipping. Well-structured.

### Coverage of `NewFileEncoder`
- `TestNewFileEncoder_OGG` and `TestNewFileEncoder_WAV`: Type-assert to verify correct encoder type.
- `TestNewFileEncoder_DefaultsToOGG`: Table-driven with empty, unknown, and case-sensitivity cases.
- `TestNewFileEncoder_NeverReturnsNil`: Exhaustive nil check across 6 inputs. Directly tests the doc contract.
- `TestNewFileEncoder_ImplementsFileEncoder`: Compile-time interface satisfaction check. Nice.
- `TestNewFileEncoder_TableDriven`: Consolidated table with wantType matching. Clean.

### Coverage of `NewDiscordDeps`
- `TestNewDiscordDeps_RequiresNetwork`: Properly skipped with clear reason. This is appropriate -- the function opens a real WebSocket connection to Discord and cannot be unit-tested without a live token.

### Constants consistency test
- `TestConstants_MatchConfig`: Verifies `backendFFmpeg == string(config.BackendFFmpeg)` and `formatWAV == string(config.FormatWAV)`. This is an excellent guard against the local constants drifting from the config package values.

### Test naming
All test names follow the `TestXxx` convention. Subtests use descriptive lowercase names.

**Minor observation:** There is some overlap between the individual tests (e.g., `TestNewGenerator_NativeBackend`) and the corresponding table-driven tests (`TestNewGenerator_TableDriven`). This is not harmful -- the individual tests serve as clear, readable smoke tests while the table-driven tests provide systematic coverage. Keeping both is fine.

**Assessment:** Pass.

## 7. Design Observations

### `cmd/skill/main.go` does not use `newServiceFromConfig`
Unlike `cmd/scream`, the skill binary wires dependencies inline in `main()` rather than using a shared `newServiceFromConfig`-like helper. This is because the skill binary always requires Discord (no optional closer nil check needed) and directly calls `svc.Play()` rather than going through a subcommand pattern. The inline wiring in skill's `main()` is only 7 lines and uses all three `app.*` functions, so the deduplication goal is still achieved at the right abstraction level.

### `newServiceFromConfig` creates Discord deps conditionally
In `cmd/scream/service.go` (lines 27-32), Discord deps are only created when `cfg.Token != ""`. This preserves the existing behavior where the `generate` subcommand works without a token. The skill binary always has a token (enforced at lines 84-87 of `main.go`), so it calls `app.NewDiscordDeps` unconditionally. Both paths are correct.

---

## Verdict: **APPROVE**

The deduplication is clean and well-executed. All three `app.*` functions correctly extract the wiring logic that was previously duplicated. Both binaries consume the shared functions identically. Error handling is correct with proper `%w` wrapping. The test suite is thorough with good edge case coverage, appropriate skips for network-dependent code, and an excellent constants-consistency guard. No import cycles, no `//nolint` directives, and documentation is complete on all exported items.
