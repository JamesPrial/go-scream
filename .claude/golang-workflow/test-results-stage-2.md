# Test Execution Report — Stage 2: Rename Stuttered Types

**Date:** 2026-02-19
**Stage:** Stage 2 — Rename stuttered types
**Baseline ref:** `.claude/golang-workflow/refactor-baseline.md`

---

## Summary

| Check | Result |
|-------|--------|
| **Verdict** | **TESTS_PASS** |
| `go test -v ./...` | ALL PASS |
| `go test -race ./...` | NO RACES (3 macOS `ld` LC_DYSYMTAB linker warnings — pre-existing OS artifact, identical to baseline) |
| `go vet ./...` | NO WARNINGS |
| `go test -cover ./...` | Coverage identical to baseline (see table) |
| `golangci-lint run` | 0 issues |

---

## Regression Detection

### Classification of test-name changes (expected, not regressions)

| Baseline name | Stage 2 name | Classification |
|---|---|---|
| `TestNativeGenerator_CorrectByteCount` | `TestGenerator_CorrectByteCount` | RENAMED — same test, different function name |
| `TestNativeGenerator_NonSilent` | `TestGenerator_NonSilent` | RENAMED |
| `TestNativeGenerator_Deterministic` | `TestGenerator_Deterministic` | RENAMED |
| `TestNativeGenerator_DifferentSeeds` | `TestGenerator_DifferentSeeds` | RENAMED |
| `TestNativeGenerator_AllPresets` | `TestGenerator_AllPresets` | RENAMED |
| `TestNativeGenerator_InvalidParams` | `TestGenerator_InvalidParams` | RENAMED |
| `TestNativeGenerator_MonoOutput` | `TestGenerator_MonoOutput` | RENAMED |
| `TestNativeGenerator_S16LERange` | `TestGenerator_S16LERange` | RENAMED |
| `TestNativeGenerator_ImplementsInterface` | `TestGenerator_ImplementsInterface` | RENAMED |
| `TestNewFFmpegGenerator_Success` | `TestNewGenerator_Success` | RENAMED (still SKIP — no ffmpeg) |
| `TestNewFFmpegGeneratorWithPath_NotNil` | `TestNewGeneratorWithPath_NotNil` | RENAMED |
| `TestFFmpegGenerator_CorrectByteCount` | `TestGenerator_CorrectByteCount` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_NonSilent` | `TestGenerator_NonSilent` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_AllPresets` | `TestGenerator_AllPresets` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_AllNamedPresets` | `TestGenerator_AllNamedPresets` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_InvalidDuration` | `TestGenerator_InvalidDuration` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_NegativeDuration` | `TestGenerator_NegativeDuration` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_InvalidSampleRate` | `TestGenerator_InvalidSampleRate` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_NegativeSampleRate` | `TestGenerator_NegativeSampleRate` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_InvalidChannels` | `TestGenerator_InvalidChannels` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_ZeroChannels` | `TestGenerator_ZeroChannels` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_BadBinaryPath` | `TestGenerator_BadBinaryPath` (ffmpeg pkg) | RENAMED |
| `TestFFmpegGenerator_InvalidAmplitude` | `TestGenerator_InvalidAmplitude` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_InvalidCrusherBits` | `TestGenerator_InvalidCrusherBits` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_InvalidLimiterLevel` | `TestGenerator_InvalidLimiterLevel` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_EvenByteCount` | `TestGenerator_EvenByteCount` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_StereoAligned` | `TestGenerator_StereoAligned` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_MonoOutput` | `TestGenerator_MonoOutput` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestFFmpegGenerator_DeterministicOutput` | `TestGenerator_DeterministicOutput` (ffmpeg pkg) | RENAMED (still SKIP) |
| `TestNewDiscordPlayer_NotNil` | `TestNewPlayer_NotNil` | RENAMED |
| `TestDiscordPlayer_Play_10Frames` | `TestPlayer_Play_10Frames` | RENAMED |
| `TestDiscordPlayer_Play_EmptyChannel` | `TestPlayer_Play_EmptyChannel` | RENAMED |
| `TestDiscordPlayer_Play_1Frame` | `TestPlayer_Play_1Frame` | RENAMED |
| `TestDiscordPlayer_Play_SpeakingProtocol` | `TestPlayer_Play_SpeakingProtocol` | RENAMED |
| `TestDiscordPlayer_Play_SilenceFrames` | `TestPlayer_Play_SilenceFrames` | RENAMED |
| `TestDiscordPlayer_Play_DisconnectCalled` | `TestPlayer_Play_DisconnectCalled` | RENAMED |
| `TestDiscordPlayer_Play_DisconnectOnError` | `TestPlayer_Play_DisconnectOnError` | RENAMED |
| `TestDiscordPlayer_Play_JoinParams` | `TestPlayer_Play_JoinParams` | RENAMED |
| `TestDiscordPlayer_Play_EmptyGuildID` | `TestPlayer_Play_EmptyGuildID` | RENAMED |
| `TestDiscordPlayer_Play_EmptyChannelID` | `TestPlayer_Play_EmptyChannelID` | RENAMED |
| `TestDiscordPlayer_Play_NilFrames` | `TestPlayer_Play_NilFrames` | RENAMED |
| `TestDiscordPlayer_Play_JoinFails` | `TestPlayer_Play_JoinFails` | RENAMED |
| `TestDiscordPlayer_Play_SpeakingTrueFails` | `TestPlayer_Play_SpeakingTrueFails` | RENAMED |
| `TestDiscordPlayer_Play_CancelledContext` | `TestPlayer_Play_CancelledContext` | RENAMED |
| `TestDiscordPlayer_Play_CancelMidPlayback` | `TestPlayer_Play_CancelMidPlayback` | RENAMED |
| `TestDiscordPlayer_Play_ValidationErrors` | `TestPlayer_Play_ValidationErrors` | RENAMED |
| `T_Close_NilCloser` (baseline typo) | `Test_Close_NilCloser` | FIXED TYPO (was missing `est_` prefix in baseline listing; actual function was always `Test_Close_NilCloser`) |

**No regressions detected.** Every test that passed in the baseline passes now. Every test that was skipped in the baseline (18 ffmpeg-dependent tests) is still skipped now. Test COUNT is identical.

---

## Test Results

### Package: `github.com/JamesPrial/go-scream/cmd/scream`
No test files.

### Package: `github.com/JamesPrial/go-scream/cmd/skill`
All 13 top-level test functions PASS (cached).

### Package: `github.com/JamesPrial/go-scream/internal/audio`
All 14 top-level test functions PASS (cached).

### Package: `github.com/JamesPrial/go-scream/internal/audio/ffmpeg`
- PASS: 5 top-level tests (including all command/arg/filter tests)
- SKIP: 18 tests (ffmpeg not on PATH — pre-existing, identical to baseline)
- Run time: 0.322s

### Package: `github.com/JamesPrial/go-scream/internal/audio/native`
All tests PASS including:
- `TestGenerator_CorrectByteCount` (was `TestNativeGenerator_CorrectByteCount`)
- `TestGenerator_NonSilent`
- `TestGenerator_Deterministic`
- `TestGenerator_DifferentSeeds`
- `TestGenerator_AllPresets` (subtests: classic, whisper, death-metal, glitch, banshee, robot)
- `TestGenerator_InvalidParams`
- `TestGenerator_MonoOutput`
- `TestGenerator_S16LERange`
- `TestGenerator_ImplementsInterface`
- All filter, layer, mixer, and oscillator tests unchanged
- Run time: 1.450s

### Package: `github.com/JamesPrial/go-scream/internal/config`
All tests PASS (cached).

### Package: `github.com/JamesPrial/go-scream/internal/discord`
All tests PASS including:
- `TestNewPlayer_NotNil` (was `TestNewDiscordPlayer_NotNil`)
- `TestPlayer_Play_10Frames` (was `TestDiscordPlayer_Play_10Frames`)
- All remaining `TestPlayer_Play_*` tests (were `TestDiscordPlayer_Play_*`)
- `TestFindPopulatedChannel_*` tests unchanged
- Run time: 0.941s

### Package: `github.com/JamesPrial/go-scream/internal/encoding`
All tests PASS (cached).

### Package: `github.com/JamesPrial/go-scream/internal/scream`
All tests PASS including:
- `Test_Close_NilCloser` (baseline had `T_Close_NilCloser` — baseline was a transcription typo, actual function was always correct)
- All other `Test_*` service tests unchanged
- Run time: 1.353s

### Package: `github.com/JamesPrial/go-scream/pkg/version`
All tests PASS (cached).

---

## Race Detection

No races detected.

Three macOS linker (`ld`) warnings appeared during race-instrumented linking (same packages as baseline):
```
ld: warning: '...000012.o' has malformed LC_DYSYMTAB, expected 98 undefined symbols
    to start at index 1626, found 95 undefined symbols starting at index 1626
```
Affected packages: `internal/encoding`, `cmd/skill`, `internal/scream`. This is a pre-existing macOS SDK/toolchain artifact, not a Go race condition. Identical to baseline.

---

## Static Analysis

```
go vet ./...
```
No output. Zero warnings. Clean.

---

## Coverage Details

| Package | Coverage | vs Baseline |
|---------|----------|-------------|
| `cmd/scream` | 0.0% (no test files) | = |
| `cmd/skill` | 21.7% | = |
| `internal/audio` | 87.5% | = |
| `internal/audio/ffmpeg` | 90.6% | = |
| `internal/audio/native` | 100.0% | = |
| `internal/config` | 97.6% | = |
| `internal/discord` | 64.1% | = (pre-existing, not a failure) |
| `internal/encoding` | 85.7% | = |
| `internal/scream` | 95.0% | = |
| `pkg/version` | 100.0% | = |

---

## Linter Output

```
golangci-lint run
0 issues.
```

---

## Test Count Verification

| Metric | Baseline | Stage 2 | Delta |
|--------|----------|---------|-------|
| Tests passed | ~200+ | ~200+ | 0 |
| Tests failed | 0 | 0 | 0 |
| Tests skipped | 18 (ffmpeg) | 18 (ffmpeg) | 0 |
| Race conditions | 0 | 0 | 0 |
| Vet warnings | 0 | 0 | 0 |
| Lint issues | 0 | 0 | 0 |

Test function renames (expected by design): 46 functions renamed due to stutter removal. No tests were added or removed.

---

## VERDICT: TESTS_PASS

All checks pass. No regressions. Coverage identical to baseline. All 46 renamed test functions now correctly use the de-stuttered names (`Generator`, `Player` instead of `NativeGenerator`, `FFmpegGenerator`, `DiscordPlayer`). The rename in Stage 2 is a pure cosmetic change with zero behavioral impact.
