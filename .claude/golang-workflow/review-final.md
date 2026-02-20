# Final Refactoring Review -- All 9 Stages

**Date:** 2026-02-19
**Reviewer:** Code Review Agent (Claude Opus 4.6)
**Baseline Commit:** 008595f Initial commit: go-scream audio generation tool
**Scope:** Full codebase refactoring across Stages 1-9

---

## 1. Stage-by-Stage Verification

### Stage 1: Remove `//nolint` directives

**Status: VERIFIED**

No `//nolint` directives remain in any `.go` source file. The two occurrences in `/Users/jamesprial/code/go-scream/internal/discord/player.go` (lines 82 and 94 in the cancellation paths) have been replaced with `_ = vc.Speaking(false)` blank-identifier assignments. This satisfies the project's rule against `//nolint` directives while preserving identical behavior -- the error is still discarded in the cancellation path where `ctx.Err()` is the authoritative return value.

### Stage 2: Rename stuttered types

**Status: VERIFIED**

All old symbol names are confirmed absent from Go source files:

| Old Symbol | New Symbol | Confirmed Absent |
|-----------|------------|-----------------|
| `AudioGenerator` | `Generator` (audio pkg) | Yes |
| `NativeGenerator` | `Generator` (native pkg) | Yes |
| `NewNativeGenerator` | `NewGenerator` (native pkg) | Yes |
| `FFmpegGenerator` | `Generator` (ffmpeg pkg) | Yes |
| `NewFFmpegGenerator` | `NewGenerator` (ffmpeg pkg) | Yes |
| `NewFFmpegGeneratorWithPath` | `NewGeneratorWithPath` (ffmpeg pkg) | Yes |
| `DiscordGoSession` | `GoSession` (discord pkg) | Yes |
| `DiscordGoVoiceConn` | `GoVoiceConn` (discord pkg) | Yes |
| `DiscordPlayer` | `Player` (discord pkg) | Yes |
| `NewDiscordPlayer` | `NewPlayer` (discord pkg) | Yes |

All test function names, compile-time interface checks, mock type doc comments, and consumer references have been updated consistently across all files listed in the migration plan.

### Stage 3: Remove `reflect` dependency and dead `closer` field

**Status: VERIFIED**

- No `"reflect"` import exists in any Go source file.
- The `closer io.Closer` field is absent from the `Service` struct at `/Users/jamesprial/code/go-scream/internal/scream/service.go`.
- No `Close()` method exists on `Service`.
- The constructor doc comment at line 27 correctly states: "Callers must pass an untyped nil (not a typed-nil interface value) when no player is needed."
- No `mockCloser` or `Test_Close_*` tests remain in `/Users/jamesprial/code/go-scream/internal/scream/service_test.go`.

### Stage 4: Remove dead `pcmBytesToInt16` and merge `resolve.go`

**Status: VERIFIED**

- No `pcmBytesToInt16` function exists in any Go source file.
- `/Users/jamesprial/code/go-scream/internal/scream/resolve.go` does not exist (confirmed via glob).
- The `resolveParams` function is present at the bottom of `/Users/jamesprial/code/go-scream/internal/scream/service.go` (lines 138-164).
- `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` imports only `errors` and `io` -- no `encoding/binary`.
- `/Users/jamesprial/code/go-scream/internal/encoding/encoder_test.go` imports only `testing` -- no `encoding/binary` or `math`.

### Stage 5: Consolidate native layer types and extract coprime constants

**Status: VERIFIED**

- `PrimaryScreamLayer` and `HighShriekLayer` are consolidated into `SweepJumpLayer` at `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go` line 17.
- `NewPrimaryScreamLayer` (line 31) and `NewHighShriekLayer` (line 98) both return `*SweepJumpLayer` parameterized by their respective coprime constants.
- The `coprime int64` field at line 25 differentiates primary scream (`audio.CoprimePrimaryScream = 137`) from high shriek (`audio.CoprimeHighShriek = 89`).
- Coprime constants at `/Users/jamesprial/code/go-scream/internal/audio/params.go` lines 33-37:
  - `CoprimePrimaryScream int64 = 137`
  - `CoprimeHarmonicSweep int64 = 251`
  - `CoprimeHighShriek int64 = 89`
  - `CoprimeNoiseBurst int64 = 173`
- The ffmpeg backend at `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go` uses `audio.CoprimePrimaryScream`, `audio.CoprimeHarmonicSweep`, `audio.CoprimeHighShriek`, and `audio.CoprimeNoiseBurst` -- no hardcoded values.
- `NewNoiseBurstLayer` signature (line 124) takes `(p audio.LayerParams, noise audio.NoiseParams)` -- the `sampleRate` parameter has been removed.
- `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go` line 104 calls `NewNoiseBurstLayer(p3, noiseWithSeed)` without `sampleRate`.
- Layer tests at `/Users/jamesprial/code/go-scream/internal/audio/native/layers_test.go` reference `SweepJumpLayer` correctly.

### Stage 6: Fix Config.Volume silent no-op (bug fix)

**Status: VERIFIED**

- `resolveParams` at `/Users/jamesprial/code/go-scream/internal/scream/service.go` lines 159-161 applies volume:
  ```go
  if cfg.Volume > 0 {
      params.Filter.VolumeBoostDB += 20 * math.Log10(cfg.Volume)
  }
  ```
- The `math` import is present (line 8).
- The `cfg.Volume > 0` guard protects against `log10(0) = -Inf`.
- When `cfg.Volume == 1.0`, `log10(1.0) == 0`, making it a no-op -- backward-compatible.
- Seven dedicated test cases in `/Users/jamesprial/code/go-scream/internal/scream/service_test.go` verify:
  - Volume 1.0 leaves VolumeBoostDB unchanged (line 798)
  - Volume 0.5 reduces by ~6 dB (line 806)
  - Volume 2.0 increases by ~6 dB (line 810) -- NOTE: see finding below
  - Volume 0.1 reduces by 20 dB (line 816)
  - Volume applied via Generate path (line 855)
  - Volume 0.0 is a no-op, no -Inf/NaN (line 887)

### Stage 7: Decouple config from audio

**Status: VERIFIED**

- No import of `"github.com/JamesPrial/go-scream/internal/audio"` exists in the `/Users/jamesprial/code/go-scream/internal/config/` directory.
- `/Users/jamesprial/code/go-scream/internal/config/validate.go` defines `knownPresets` as a package-level `[]string` variable (line 6) with hardcoded preset names.
- The `isValidPreset` function (line 51) iterates over `knownPresets` without any audio package dependency.
- The preset list matches `audio.AllPresets()` exactly: `classic`, `whisper`, `death-metal`, `glitch`, `banshee`, `robot`.

**Note on synchronization risk:** The `knownPresets` list in `/Users/jamesprial/code/go-scream/internal/config/validate.go` must be manually kept in sync with preset definitions in `/Users/jamesprial/code/go-scream/internal/audio/presets.go`. The doc comment at line 5 notes this requirement: "This list must be kept in sync with the preset constants defined in internal/audio/presets.go (audio.AllPresets)." This is acceptable for a codebase of this size, though a compile-time or test-time sync check would be more robust.

### Stage 8: Deduplicate service wiring into internal/app/wire.go

**Status: VERIFIED**

- `/Users/jamesprial/code/go-scream/internal/app/wire.go` is a new file providing three exported functions:
  - `NewGenerator(backend string) (audio.Generator, error)` -- selects native or ffmpeg backend.
  - `NewFileEncoder(format string) encoding.FileEncoder` -- selects WAV or OGG encoder.
  - `NewDiscordDeps(token string) (discord.VoicePlayer, io.Closer, error)` -- creates discordgo session.
- `/Users/jamesprial/code/go-scream/cmd/scream/service.go` uses `app.NewGenerator`, `app.NewFileEncoder`, `app.NewDiscordDeps`.
- `/Users/jamesprial/code/go-scream/cmd/skill/main.go` uses `app.NewGenerator`, `app.NewFileEncoder`, `app.NewDiscordDeps`.
- Package imports are unidirectional: `internal/app` imports from `internal/audio`, `internal/encoding`, `internal/discord` -- none import back. No import cycles (confirmed via `go build ./...`).
- Package-local constants `backendFFmpeg` and `formatWAV` at lines 21 and 25 avoid importing the `config` package, preventing a cycle. Tests at `/Users/jamesprial/code/go-scream/internal/app/wire_test.go` line 284-292 verify these constants match `config.BackendFFmpeg` and `config.FormatWAV`.

### Stage 9: Extract signal-context + closer defer into runWithService

**Status: VERIFIED**

- `runWithService` helper at `/Users/jamesprial/code/go-scream/cmd/scream/service.go` lines 46-62 handles:
  - Signal-notifying context creation (`SIGINT`, `SIGTERM`)
  - Service construction via `newServiceFromConfig`
  - Deferred closer cleanup with stderr warning on error
  - Delegation to callback `fn func(ctx context.Context, svc *scream.Service) error`
- `/Users/jamesprial/code/go-scream/cmd/scream/play.go` line 56 calls `runWithService(cfg, func(ctx context.Context, svc *scream.Service) error { ... })`.
- `/Users/jamesprial/code/go-scream/cmd/scream/generate.go` line 42 calls `runWithService(cfg, func(ctx context.Context, svc *scream.Service) error { ... })`.
- The deferred closer in `runWithService` follows the CLI code pattern: logs warning to stderr in the defer (matching MEMORY.md: "For deferred Close() in CLI code: log warnings to stderr in the defer").

---

## 2. API Changes Compliance

All API changes match `/Users/jamesprial/code/go-scream/.claude/golang-workflow/api-changes.md` exactly:

- **Stage 2 renames**: All 10 renamed symbols confirmed present with new names, absent with old names.
- **Stage 3 deletions**: `Service.Close()` and `closer` field removed.
- **Stage 4 deletions**: `pcmBytesToInt16` removed, `resolve.go` deleted, `resolveParams` moved.
- **Stage 5 changes**: `PrimaryScreamLayer` and `HighShriekLayer` collapsed into `SweepJumpLayer`. Four coprime constants added. `NewNoiseBurstLayer` signature changed.
- **Stage 6 behavior**: `resolveParams` now applies `cfg.Volume`.
- **Stage 7 signature**: `Validate(cfg)` no longer imports audio (uses `knownPresets`).
- **Stage 8 new symbols**: `app.NewGenerator`, `app.NewFileEncoder`, `app.NewDiscordDeps`.
- **Stage 9 new symbol**: `runWithService` (unexported).

No unplanned API changes detected.

---

## 3. Cross-Cutting Concerns

### Error Handling Consistency

All error wrapping uses `%w` format verb consistently. Verified across:
- `/Users/jamesprial/code/go-scream/internal/scream/service.go`: `fmt.Errorf("%w: %w", ...)` pattern for `ErrGenerateFailed`, `ErrEncodeFailed`, `ErrPlayFailed`.
- `/Users/jamesprial/code/go-scream/internal/discord/player.go`: `fmt.Errorf("%w: %w", ...)` for `ErrVoiceJoinFailed`, `ErrSpeakingFailed`.
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go`: `fmt.Errorf("%w: %w", ...)` for `ErrFFmpegNotFound`, `ErrFFmpegFailed`.
- `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go`: `fmt.Errorf("invalid params: %w", err)`.
- `/Users/jamesprial/code/go-scream/internal/app/wire.go`: `fmt.Errorf("failed to create discord session: %w", err)`.

### Nil Safety

- `Player.Play` at `/Users/jamesprial/code/go-scream/internal/discord/player.go` line 48 checks `frames == nil` before use.
- `Service.Play` at `/Users/jamesprial/code/go-scream/internal/scream/service.go` line 52 checks `s.player == nil` before use.
- `runWithService` at `/Users/jamesprial/code/go-scream/cmd/scream/service.go` line 54 checks `closer != nil` before deferring close.
- `newServiceFromConfig` properly returns nil closer when no token is set.

### Resource Cleanup

- `/Users/jamesprial/code/go-scream/internal/discord/player.go` line 61: `defer func() { if derr := vc.Disconnect(); ... }()` with named return `retErr`.
- `/Users/jamesprial/code/go-scream/cmd/scream/service.go` line 55: Deferred `closer.Close()` with stderr warning.
- `/Users/jamesprial/code/go-scream/cmd/skill/main.go` line 120: Deferred `sessionCloser.Close()` with stderr warning.
- `/Users/jamesprial/code/go-scream/cmd/scream/generate.go` line 47: Deferred `f.Close()` with stderr warning.

All cleanup patterns are consistent with MEMORY.md preferences (named returns for library code, stderr warnings for CLI code).

### Interface Design Consistency

- `audio.Generator` is the single interface for audio generation (used by both native and ffmpeg backends).
- `discord.VoicePlayer` is the single interface for Discord playback.
- `discord.Session` and `discord.VoiceConn` abstract discordgo types.
- `encoding.FileEncoder` and `encoding.OpusFrameEncoder` are the encoding interfaces.
- All interface implementations have compile-time checks (`var _ Interface = (*Impl)(nil)`).

---

## 4. Dead Code Analysis

No leftover dead code detected:

- `pcmBytesToInt16`: absent from all Go source files.
- `Service.Close()` and `closer` field: absent from `service.go`.
- `reflect` import: absent from all Go source files.
- `resolve.go`: file does not exist.
- Old stuttered type names: absent from all Go source files.
- Hardcoded coprime values (137, 251, 89, 173) in ffmpeg backend: replaced with `audio.Coprime*` constants.
- Duplicate wiring logic in `cmd/scream/service.go` and `cmd/skill/main.go`: replaced with `app.*` calls.
- Duplicate signal-context setup: replaced with `runWithService`.

---

## 5. Test Coverage Assessment

### Test Quality

All test files follow Go conventions:
- Test function names follow `TestXxx` / `TestXxx_Yyy` pattern.
- Table-driven tests with descriptive `name` fields are used throughout.
- Error paths have dedicated test cases.
- Mock types are well-structured with mutex protection for concurrent access.
- Compile-time interface checks present in both implementation and test files.
- Benchmarks present for key hot paths.

### Tests Added/Removed Per Stage

| Stage | Tests Added | Tests Removed |
|-------|-----------|--------------|
| 1 | 0 | 0 |
| 2 | 0 (renamed only) | 0 (renamed only) |
| 3 | 0 | 4 (`Test_Close_*`) + `mockCloser` |
| 4 | 0 | `TestPcmBytesToInt16_*` tests |
| 5 | 0 (type refs updated) | 0 |
| 6 | 7 (`Test_ResolveParams_Volume*`) | 0 |
| 7 | 0 | 0 |
| 8 | 15+ (`wire_test.go`) | 0 |
| 9 | 0 | 0 |

### Documentation Quality

All exported items have doc comments:
- All exported types, functions, constants, and variables in `internal/audio`, `internal/audio/native`, `internal/audio/ffmpeg`, `internal/discord`, `internal/encoding`, `internal/scream`, `internal/config`, `internal/app`, and `pkg/version` have doc comments.
- Unexported helpers that warrant explanation have comments (e.g., `splitmix64`, `seededRandom`, `deriveSeed`).

---

## 6. Findings

### Finding 1 (Informational): Volume test allows values above 1.0

At `/Users/jamesprial/code/go-scream/internal/scream/service_test.go` line 808, the test case "volume 2.0 increases VolumeBoostDB by ~6 dB" tests with `cfg.Volume = 2.0`. However, `config.Validate` at `/Users/jamesprial/code/go-scream/internal/config/validate.go` line 39 rejects volume values above 1.0 (`ErrInvalidVolume`). This means `cfg.Volume = 2.0` would fail validation if the full config pipeline were used.

This is not a bug -- the `resolveParams` function receives an already-validated config in production, and the test bypasses validation to verify the volume math directly. The test is documenting correct behavior of `resolveParams` in isolation. However, the test name could be misleading since 2.0 is not a valid production volume value.

**Severity:** Informational only. No action required.

### Finding 2 (Informational): knownPresets synchronization risk

At `/Users/jamesprial/code/go-scream/internal/config/validate.go` line 6, the `knownPresets` variable is a hardcoded `[]string` that must be manually kept in sync with `audio.AllPresets()`. A compile-time or test-time synchronization check would prevent drift. For example, a test in `internal/config` (or a cross-package integration test) could verify the lists match.

However, the current approach is acceptable for a codebase of this size, and the doc comment at line 5 clearly documents the synchronization requirement.

**Severity:** Informational only. No action required for this refactoring wave.

### Finding 3 (Informational): `cmd/skill/main.go` does not use `runWithService`

The Stage 9 extraction of `runWithService` only applies to `cmd/scream`. The `cmd/skill/main.go` binary at lines 102-131 still has inline signal-context setup, service construction, and deferred closer cleanup. This is because `cmd/skill` uses a different wiring flow (direct `main()` function, not cobra commands), and the `runWithService` helper is specific to the `cmd/scream` package.

This is not an issue -- the binaries have different structures. The duplication between them was addressed in Stage 8 (shared `app.*` functions). The remaining inline code in `cmd/skill/main.go` is structurally different enough that extracting a shared helper would require cross-package code or a different abstraction.

**Severity:** Informational only. No action required.

---

## 7. Behavior Preservation Verification

All ten behavior preservation constraints from the migration plan are satisfied:

1. **Audio generation produces identical output for same params.** The native generator math is unchanged. The coprime constants are extracted but values are identical.
2. **Opus encoding produces same frame count and sizes.** Encoding code is unchanged (only dead `pcmBytesToInt16` removed).
3. **OGG and WAV file encoding produces valid output.** Encoder implementations untouched.
4. **Discord playback protocol unchanged.** Player logic unchanged (only `//nolint` to `_ =` and type renames).
5. **Error sentinel chains unchanged.** All sentinel errors and wrapping patterns preserved.
6. **Configuration loading and validation unchanged.** Validation logic preserved; `knownPresets` matches `AllPresets()`.
7. **Preset definitions unchanged.** Preset values in `presets.go` untouched.
8. **CLI behavior unchanged.** Command structure, flags, and output preserved.
9. **`internal/` package boundary maintained.** All refactored packages remain in `internal/`.
10. **CGO dependency boundaries unchanged.** Only `layeh.com/gopus` requires CGO; this is untouched.

The only behavior change is the Stage 6 bug fix: `cfg.Volume` is now applied to `params.Filter.VolumeBoostDB`. This was an intentional fix for a silent no-op where the config field was validated but never used.

---

## 8. Compilation and Import Cycle Check

`go build ./...` succeeds with zero errors and zero warnings. No import cycles exist. The import graph is:

```
cmd/scream -> internal/app, internal/config, internal/encoding, internal/scream
cmd/skill  -> internal/app, internal/config, internal/encoding, internal/scream
internal/app -> internal/audio, internal/audio/native, internal/audio/ffmpeg,
               internal/discord, internal/encoding
internal/scream -> internal/audio, internal/config, internal/discord, internal/encoding
internal/config -> (no internal imports)
internal/audio/native -> internal/audio
internal/audio/ffmpeg -> internal/audio
internal/discord -> (external only: discordgo)
internal/encoding -> (external only: gopus, pion/rtp, pion/webrtc)
```

No cycles. `internal/config` no longer imports `internal/audio` (Stage 7 verified).

---

## Verdict

**APPROVE**

The 9-stage refactoring is complete, correct, and consistent. All planned API changes match the approved api-changes.md specification. No unplanned API breaks were introduced. Dead code has been thoroughly removed. Error handling, nil safety, resource cleanup, and naming conventions are consistent throughout. The three informational findings are minor observations that do not require changes for this refactoring wave.

The codebase is ready for Wave 4 verification (test execution by the Test Runner agent).
