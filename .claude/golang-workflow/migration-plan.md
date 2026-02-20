# Migration Plan -- go-scream Refactor

**Date:** 2026-02-19
**Baseline Commit:** 008595f Initial commit: go-scream audio generation tool
**Mode:** Aggressive (public API changes allowed)

---

## Current State Summary

The codebase is a Go audio generation tool with two binaries (`cmd/scream` CLI, `cmd/skill` OpenClaw skill wrapper) and five internal packages (`audio`, `encoding`, `discord`, `config`, `scream`). All tests pass, no race conditions, no vet warnings, zero lint issues per the refactor baseline.

### Issues Identified

1. **Stuttered naming** -- Multiple exported types stutter when qualified with their package name (e.g., `audio.AudioGenerator`, `native.NativeGenerator`, `discord.DiscordPlayer`). This violates Go naming conventions per Effective Go.

2. **`//nolint:errcheck` directives** -- Two occurrences in `/Users/jamesprial/code/go-scream/internal/discord/player.go` (lines 82, 94) suppress error checking on `vc.Speaking(false)` calls. Per user preferences, `//nolint` directives must never be used; errors must be handled properly.

3. **`reflect` import for typed-nil detection** -- `/Users/jamesprial/code/go-scream/internal/scream/service.go` imports `reflect` solely to detect typed-nil interface values in `NewServiceWithDeps`. This is a code smell; the constructor should not need reflection.

4. **Dead `closer` field and `Close()` method** -- `Service.closer` is never set through the constructor. Only tests set it via direct field access (`svc.closer = mc`). The `Close()` method and `closer` field are dead production code.

5. **Dead `pcmBytesToInt16` function** -- Defined in `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (line 69) and extensively tested, but never called in any production code path.

6. **Single-function file `resolve.go`** -- `/Users/jamesprial/code/go-scream/internal/scream/resolve.go` contains only `resolveParams`, which is a small unexported helper. It can be merged into `service.go` for cohesion.

7. **Duplicated wiring logic** -- `/Users/jamesprial/code/go-scream/cmd/scream/service.go` and `/Users/jamesprial/code/go-scream/cmd/skill/main.go` duplicate the logic for selecting audio backend, creating encoders, and wiring the Discord session.

---

## Target State

After all migration stages are complete:

- All exported types use idiomatic Go names without stutter.
- No `//nolint` directives exist in the codebase.
- No `reflect` import exists in the codebase.
- No dead code exists (no `pcmBytesToInt16`, no `closer` field, no `Close()` method on `Service`).
- `resolve.go` is merged into `service.go`.
- `PrimaryScreamLayer` and `HighShriekLayer` collapsed into single `SweepJumpLayer`.
- Coprime constants shared between native and ffmpeg backends.
- `Config.Volume` actually applied during audio generation (bug fixed).
- `internal/config` no longer imports `internal/audio` (decoupled).
- Service wiring is deduplicated into a shared `internal/app` package.
- Signal-context + closer defer pattern extracted in `cmd/scream`.
- All existing test behaviors are preserved.
- Zero test failures, zero race conditions, zero vet warnings, zero lint issues.

---

## Migration Stages

### Stage 1: Remove `//nolint` directives

**Rationale:** Per user preferences, `//nolint` directives must never be used. The two occurrences in `player.go` suppress error checking on best-effort `vc.Speaking(false)` calls during context cancellation. Since these are in cancellation paths where the error genuinely cannot be propagated (we are already returning `ctx.Err()`), the correct fix is to assign the return value to the blank identifier.

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/internal/discord/player.go` | Line 82: Replace `vc.Speaking(false) //nolint:errcheck // best-effort` with `_ = vc.Speaking(false)`. Line 94: Same replacement. |

**Dependencies:** None. This is a standalone change.

**Risks:** None. The behavior is identical -- the error is still discarded. The `_ =` assignment satisfies both the linter and the user's preference.

**Test impact:** No test changes required. All existing tests continue to pass unchanged.

**Verification:** `golangci-lint run` reports zero issues. `go vet ./...` clean. All tests pass.

---

### Stage 2: Rename stuttered types

**Rationale:** Go naming convention states that a type exported from a package should not repeat the package name. `audio.AudioGenerator` should be `audio.Generator`, `native.NativeGenerator` should be `native.Generator`, etc.

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/internal/audio/generator.go` | Rename `AudioGenerator` to `Generator`. Update doc comment. |
| `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go` | Rename `NativeGenerator` to `Generator`. Rename `NewNativeGenerator` to `NewGenerator`. Update doc comments. Update compile-time interface reference. |
| `/Users/jamesprial/code/go-scream/internal/audio/native/generator_test.go` | Update all references: `NewNativeGenerator` to `NewGenerator`, `NativeGenerator` to `Generator`. Update `TestNativeGenerator_ImplementsInterface` to use `audio.Generator`. Rename test functions: `TestNativeGenerator_*` to `TestGenerator_*`, `BenchmarkNativeGenerator_*` to `BenchmarkGenerator_*`. |
| `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go` | Rename `FFmpegGenerator` to `Generator`. Rename `NewFFmpegGenerator` to `NewGenerator`. Rename `NewFFmpegGeneratorWithPath` to `NewGeneratorWithPath`. Update compile-time interface check. Update doc comments. |
| `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator_test.go` | Update all references. Rename test functions: `TestNewFFmpegGenerator_*` to `TestNewGenerator_*`, `TestFFmpegGenerator_*` to `TestGenerator_*`, `BenchmarkFFmpegGenerator_*` to `BenchmarkGenerator_*`. Update compile-time interface check. |
| `/Users/jamesprial/code/go-scream/internal/discord/session.go` | Rename `DiscordGoSession` to `GoSession`. Rename `DiscordGoVoiceConn` to `GoVoiceConn`. Update doc comments. |
| `/Users/jamesprial/code/go-scream/internal/discord/player.go` | Rename `DiscordPlayer` to `Player`. Rename `NewDiscordPlayer` to `NewPlayer`. Update compile-time interface check. Update doc comments. |
| `/Users/jamesprial/code/go-scream/internal/discord/player_test.go` | Update all references: `NewDiscordPlayer` to `NewPlayer`, `DiscordPlayer` to `Player`. Rename test functions: `TestNewDiscordPlayer_*` to `TestNewPlayer_*`, `TestDiscordPlayer_*` to `TestPlayer_*`, `BenchmarkDiscordPlayer_*` to `BenchmarkPlayer_*`. Update compile-time interface check. |
| `/Users/jamesprial/code/go-scream/internal/scream/service.go` | Update `generator audio.AudioGenerator` to `generator audio.Generator` in struct field and constructor parameter. |
| `/Users/jamesprial/code/go-scream/internal/scream/service_test.go` | Update doc comment on `mockGenerator` from "implements audio.AudioGenerator" to "implements audio.Generator". |
| `/Users/jamesprial/code/go-scream/cmd/scream/service.go` | Update `var gen audio.AudioGenerator` to `var gen audio.Generator`. Update `ffmpeg.NewFFmpegGenerator()` to `ffmpeg.NewGenerator()`. Update `native.NewNativeGenerator()` to `native.NewGenerator()`. Update `discord.DiscordGoSession{S: session}` to `discord.GoSession{S: session}`. Update `discord.NewDiscordPlayer(sess)` to `discord.NewPlayer(sess)`. |
| `/Users/jamesprial/code/go-scream/cmd/skill/main.go` | Same changes as `cmd/scream/service.go`: Update `audio.AudioGenerator` to `audio.Generator`, `ffmpeg.NewFFmpegGenerator()` to `ffmpeg.NewGenerator()`, `native.NewNativeGenerator()` to `native.NewGenerator()`, `discord.DiscordGoSession` to `discord.GoSession`, `discord.NewDiscordPlayer` to `discord.NewPlayer`. |

**Dependencies:** Stage 1 must be complete first (to maintain a clean diff history).

**Risks:**
- **Medium:** This is a broad rename touching many files. Perform a global search for each old name after the rename to ensure no references are missed.
- **Low:** The `DiscordGoSession` and `DiscordGoVoiceConn` types have an exported field (`S` and `VC` respectively) that is accessed directly by callers. The field names do not change -- only the struct type names change.

**Test impact:** Test function names change. All test logic remains identical. No assertions change.

**Verification:** `go build ./...` compiles. All tests pass. `golangci-lint run` clean.

---

### Stage 3: Remove `reflect` dependency and dead `closer` field

**Rationale:** The `reflect` import in `service.go` exists solely to detect typed-nil interface values. Both callers already pass untyped nil. The `closer` field and `Close()` method are dead production code.

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/internal/scream/service.go` | (a) Remove `"reflect"` from import block. (b) Remove the typed-nil normalization block. (c) Add doc comment clarifying callers must pass untyped nil. (d) Remove `closer io.Closer` field. (e) Remove `Close()` method. |
| `/Users/jamesprial/code/go-scream/internal/scream/service_test.go` | (a) Remove `mockCloser` type. (b) Remove all `Close()` tests: `Test_Close_WithCloser`, `Test_Close_WithCloserError`, `Test_Close_NilCloser`, `Test_Close_CalledTwice_NoPanic`. (c) Remove any `svc.closer = mc` lines. |

**Dependencies:** Stage 2 must be complete.

**Risks:**
- **Low:** `Close()` lives in `internal/scream`, so only `cmd/` binaries can call it. Neither does.
- **Low:** Removing typed-nil guard means future callers passing typed-nil would hit a nil-pointer. Doc comment mitigates.

**Test impact:** Four `Close()` tests and `mockCloser` type removed. All remaining tests unaffected.

**Verification:** `go build ./...` compiles without `reflect`. All remaining tests pass. `golangci-lint run` clean.

---

### Stage 4: Remove dead `pcmBytesToInt16` and merge `resolve.go`

**Rationale:** `pcmBytesToInt16` is never called in production. `resolve.go` contains a single unexported function that belongs in `service.go`.

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` | Remove `pcmBytesToInt16` function. Remove `"encoding/binary"` from imports. |
| `/Users/jamesprial/code/go-scream/internal/encoding/encoder_test.go` | Remove all `pcmBytesToInt16` tests. Remove `"encoding/binary"` and `"math"` from imports. |
| `/Users/jamesprial/code/go-scream/internal/scream/resolve.go` | Delete this file entirely. |
| `/Users/jamesprial/code/go-scream/internal/scream/service.go` | Add the `resolveParams` function to the bottom. |

**Dependencies:** Stage 3 must be complete.

**Risks:** None. Both are confirmed dead/internal code.

**Test impact:** `pcmBytesToInt16` tests removed. `resolveParams` tested indirectly through `Play()` and `Generate()` tests.

**Verification:** `go build ./...` compiles. All tests pass. `golangci-lint run` clean.

---

### Stage 5: Consolidate native layer types and extract coprime constants

**Rationale:** `PrimaryScreamLayer` and `HighShriekLayer` in `internal/audio/native/layers.go` are structurally identical (~80 lines of near-duplicate code). They share the same 9 fields, same constructor shape, and identical `Sample()` method — differing only by one coprime constant (137 vs 89). Collapse into a single `SweepJumpLayer` parameterized by coprime value. Also extract coprime constants (137, 251, 89, 173) to `internal/audio` so both backends share a single source of truth, and remove the unused `sampleRate` parameter from `NewNoiseBurstLayer`.

**New constants in `internal/audio`:**

| Constant | Value | Usage |
|----------|-------|-------|
| `CoprimePrimaryScream` | 137 | PrimaryScream/SweepJump layer |
| `CoprimeHarmonicSweep` | 251 | HarmonicSweep layer |
| `CoprimeHighShriek` | 89 | HighShriek/SweepJump layer |
| `CoprimeNoiseBurst` | 173 | NoiseBurst layer |

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/internal/audio/params.go` (or new `constants.go`) | Add coprime constants. |
| `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go` | (a) Collapse `PrimaryScreamLayer` and `HighShriekLayer` into `SweepJumpLayer` with a `coprime int64` field. (b) Update `NewPrimaryScreamLayer` and `NewHighShriekLayer` to return `*SweepJumpLayer` with appropriate coprime. (c) Update `HarmonicSweepLayer` and `NoiseBurstLayer` to use `audio.Coprime*` constants. (d) Remove unused `sampleRate` param from `NewNoiseBurstLayer`. |
| `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go` | Update `buildLayers` to remove `sampleRate` arg from `NewNoiseBurstLayer` call. |
| `/Users/jamesprial/code/go-scream/internal/audio/native/layers_test.go` | Update test references if type names change. Existing behavior tests preserved. |
| `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go` | Replace hardcoded coprime values (137, 251, 89, 173) with `audio.Coprime*` constants. |

**Dependencies:** Stage 4 must be complete (avoid conflicts with earlier stages touching the same packages).

**Risks:**
- **Medium:** Collapsing two types into one changes the type names in the native package. The old types are unexported through the interface (`audio.Generator`), so consumers don't reference them directly. But tests reference them by name.
- **Low:** Removing `sampleRate` param is a signature change on an unexported constructor.

**Test impact:** Layer tests need type name updates. Behavior assertions unchanged.

**Verification:** `go build ./...` compiles. All tests pass. `golangci-lint run` clean.

---

### Stage 6: Fix Config.Volume silent no-op

**Rationale:** `Config.Volume` is accepted via CLI flag (`--volume`), env var (`SCREAM_VOLUME`), YAML config (`volume:`), validated, and stored — but never applied during audio generation. `resolveParams` (now in `service.go`) never reads `cfg.Volume`. This means the volume setting is silently ignored — a bug.

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/internal/scream/service.go` | In `resolveParams`: apply `cfg.Volume` as a multiplier to `params.Filter.VolumeBoostDB` (or equivalent appropriate scaling). |

**Dependencies:** Stage 4 must be complete (resolve.go merged into service.go).

**Risks:**
- **Low:** This is technically a behavior change (fixing a bug), but the current behavior is clearly unintended. The config field exists and is validated but never used.

**Test impact:** Add test(s) in `service_test.go` to verify volume config is applied. Existing tests that don't set volume should be unaffected (default volume = 1.0 or 0dB should be a no-op).

**Verification:** `go build ./...` compiles. All tests pass. Manual verification that `--volume` flag now affects output.

---

### Stage 7: Break config->audio coupling

**Rationale:** `internal/config/validate.go` imports `internal/audio` solely to call `audio.AllPresets()` in `isValidPreset()`. This couples the config layer to the audio domain. Config should be a pure data/validation layer.

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/internal/config/validate.go` | Change `isValidPreset` to accept a list of valid preset names (or use a package-level variable set at init time) instead of importing `audio.AllPresets()`. |
| `/Users/jamesprial/code/go-scream/internal/config/config.go` | Add a `ValidPresets []string` field or a `RegisterPresets` function that callers use to inject valid preset names. |
| `/Users/jamesprial/code/go-scream/cmd/scream/service.go` | Pass `audio.AllPresets()` result when configuring validation. |
| `/Users/jamesprial/code/go-scream/cmd/skill/main.go` | Same as above. |
| `/Users/jamesprial/code/go-scream/internal/config/validate_test.go` | Update tests to provide preset names explicitly. |

**Dependencies:** Stage 2 must be complete (renames done).

**Risks:**
- **Medium:** Changes the config validation API. All callers must now provide preset names.

**Test impact:** Config validation tests updated to inject preset names. Behavior unchanged.

**Verification:** `go build ./...` compiles. All tests pass. Confirm `internal/config` no longer imports `internal/audio`.

---

### Stage 8: Deduplicate service wiring

**Rationale:** `cmd/scream/service.go` and `cmd/skill/main.go` duplicate generator selection, encoder creation, and Discord session wiring logic.

**New file:**

| File | Purpose |
|------|---------|
| `/Users/jamesprial/code/go-scream/internal/app/wire.go` | Shared wiring: `NewGenerator(backend string) (audio.Generator, error)`, `NewFileEncoder(format string) encoding.FileEncoder`, `NewDiscordDeps(token string) (discord.VoicePlayer, io.Closer, error)`. |

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/cmd/scream/service.go` | Replace inline wiring with calls to `app.*` functions. |
| `/Users/jamesprial/code/go-scream/cmd/skill/main.go` | Replace inline wiring with calls to `app.*` functions. |

**Dependencies:** Stages 1-7 must be complete.

**Risks:**
- **Medium:** New package `internal/app`. No import cycles (verified: it imports from `internal/audio`, `internal/encoding`, `internal/discord` — none import back).

**Test impact:** New test file `internal/app/wire_test.go`. Existing tests unchanged.

**Verification:** `go build ./...` compiles. All tests pass. No import cycles.

---

### Stage 9: Extract signal-context + closer defer pattern

**Rationale:** `cmd/scream/play.go` and `cmd/scream/generate.go` both contain identical 10-line blocks for signal context setup and deferred closer cleanup. Extract to a shared helper.

**Files to modify:**

| File | Change |
|------|--------|
| `/Users/jamesprial/code/go-scream/cmd/scream/service.go` (or new `cmd/scream/run.go`) | Add `runWithService(cfg config.Config, fn func(ctx context.Context, svc *scream.Service) error) error` helper that handles signal context, service creation, closer defer, and delegates to the callback. |
| `/Users/jamesprial/code/go-scream/cmd/scream/play.go` | Replace inline signal+closer+service setup with `runWithService` call. |
| `/Users/jamesprial/code/go-scream/cmd/scream/generate.go` | Same replacement. |

**Dependencies:** Stage 8 must be complete (wiring already extracted to `internal/app`).

**Risks:**
- **Low:** `runWithService` is an unexported helper in the `cmd/scream` package. No public API change.
- **Low:** The two commands have slightly different post-setup logic (play calls `svc.Play`, generate calls `svc.Generate` with a file writer). The callback pattern handles this cleanly.

**Test impact:** No existing tests for `cmd/scream` — no test changes needed. Optionally add tests for `runWithService`.

**Verification:** `go build ./...` compiles. All tests pass.

---

## Implementation Guide

### Execution Order

Stages MUST be executed in order (1 through 9). Each stage depends on prior stages being complete and verified.

### Per-Stage Workflow

For each stage:
1. Make the changes described in the stage.
2. Run `go build ./...` to confirm compilation.
3. Run `go vet ./...` to confirm no vet warnings.
4. Run `go test ./...` to confirm all tests pass.
5. Run `go test -race ./...` to confirm no race conditions.
6. Run `golangci-lint run` to confirm zero lint issues.

---

## Test Adaptation Guide

### Stage 1
No test changes.

### Stage 2
- Rename test functions to match new type names (mechanical mapping).
- Update compile-time interface checks: `var _ audio.AudioGenerator` becomes `var _ audio.Generator`.
- Update doc comments on mock types.
- All test assertions remain identical.

### Stage 3
- Remove `mockCloser` type and all four `Test_Close_*` tests.
- Remove any `svc.closer = mc` lines.
- All remaining test logic unchanged.

### Stage 4
- Remove all `TestPcmBytesToInt16_*` tests and `BenchmarkPcmBytesToInt16_*`.
- Remove `"encoding/binary"` and `"math"` imports from `encoder_test.go`.
- All `resolveParams` behavior tested indirectly. No new tests needed.

### Stage 5
- Update layer test type references (`PrimaryScreamLayer`/`HighShriekLayer` → `SweepJumpLayer`).
- Update `NewNoiseBurstLayer` calls (remove `sampleRate` arg).
- Behavior assertions unchanged.

### Stage 6
- Add test(s) verifying `cfg.Volume` is applied in `resolveParams`.
- Existing tests unaffected (default volume should be no-op).

### Stage 7
- Update config validation tests to inject preset names.
- Remove implicit `audio.AllPresets()` dependency in test setup.

### Stage 8
- Write new tests in `internal/app/wire_test.go`.
- Existing tests unchanged.

### Stage 9
- No existing tests for `cmd/scream` commands. No test changes needed.

---

## Behavior Preservation Constraints

1. Audio generation produces identical output for same params.
2. Opus encoding produces same frame count and sizes.
3. OGG and WAV file encoding produces valid output.
4. Discord playback protocol unchanged.
5. Error sentinel chains unchanged.
6. Configuration loading and validation unchanged.
7. Preset definitions unchanged.
8. CLI behavior unchanged.
9. `internal/` package boundary maintained.
10. CGO dependency boundaries unchanged.
