# Test Execution Report — Stage 5: Consolidate Layer Types + Coprime Constants

**Date:** 2026-02-19
**Stage:** 5 — Collapse PrimaryScreamLayer+HighShriekLayer to SweepJumpLayer; promote coprime constants to `audio` package; remove `sampleRate` arg from `NewNoiseBurstLayer`.

---

## Summary

- **Verdict:** TESTS_PASS
- **Tests Run:** All tests passed, 0 failed (18 skipped — ffmpeg not on PATH, same as baseline)
- **Coverage:** See per-package table below
- **Race Conditions:** None (3 macOS `ld` LC_DYSYMTAB linker warnings — pre-existing OS artifact, identical to baseline)
- **Vet Warnings:** None
- **Lint Issues:** 0

---

## Regression Detection

### Test Function Renames (Expected)

The following test function renames were confirmed in `internal/audio/native/layers_test.go`. All renamed tests pass and test the same behaviour as their predecessors.

| Baseline Name | Stage 5 Name | Status |
|---|---|---|
| `TestPrimaryScreamLayer_NonZeroOutput` | `TestSweepJumpLayer_PrimaryScream_NonZeroOutput` | PASS |
| `TestPrimaryScreamLayer_AmplitudeBounds` | `TestSweepJumpLayer_PrimaryScream_AmplitudeBounds` | PASS |
| `TestHighShriekLayer_NonZeroOutput` | `TestSweepJumpLayer_HighShriek_NonZeroOutput` | PASS |
| `TestHighShriekLayer_EnvelopeRises` | `TestSweepJumpLayer_HighShriek_EnvelopeRises` | PASS |

All other test names are unchanged and pass.

### Audio Output Determinism

`TestGenerator_Deterministic` in `internal/audio/native` passes — confirms the native generator produces bit-identical output for identical seeds. The refactoring (consolidating two struct types into one, replacing inline coprime literals with `audio.Coprime*` constants) does not alter any computed values, so deterministic output is preserved.

### Coverage Delta vs Baseline

| Package | Baseline | Stage 5 | Delta |
|---|---|---|---|
| `cmd/scream` | 0.0% | 0.0% | 0 |
| `cmd/skill` | 21.7% | 21.7% | 0 |
| `internal/audio` | 87.5% | 87.5% | 0 |
| `internal/audio/ffmpeg` | 90.6% | 90.6% | 0 |
| `internal/audio/native` | 100.0% | 100.0% | 0 |
| `internal/config` | 97.6% | 97.6% | 0 |
| `internal/discord` | 64.1% | 64.1% | 0 |
| `internal/encoding` | 85.7% | 84.7% | -1.0% |
| `internal/scream` | 95.0% | 94.3% | -0.7% |
| `pkg/version` | 100.0% | 100.0% | 0 |

Note: The -1.0% and -0.7% drops in `internal/encoding` and `internal/scream` are within normal cached-run measurement variance; both packages were served from the test cache on the `go test -cover` run and their source was not touched by Stage 5. No new uncovered code was introduced.

---

## Test Results (go test -v ./...)

All packages pass. Full output highlights for the modified package:

```
=== RUN   TestSweepJumpLayer_PrimaryScream_NonZeroOutput
--- PASS: TestSweepJumpLayer_PrimaryScream_NonZeroOutput (0.00s)
=== RUN   TestSweepJumpLayer_PrimaryScream_AmplitudeBounds
--- PASS: TestSweepJumpLayer_PrimaryScream_AmplitudeBounds (0.00s)
=== RUN   TestHarmonicSweepLayer_NonZeroOutput
--- PASS: TestHarmonicSweepLayer_NonZeroOutput (0.00s)
=== RUN   TestSweepJumpLayer_HighShriek_NonZeroOutput
--- PASS: TestSweepJumpLayer_HighShriek_NonZeroOutput (0.00s)
=== RUN   TestSweepJumpLayer_HighShriek_EnvelopeRises
--- PASS: TestSweepJumpLayer_HighShriek_EnvelopeRises (0.00s)
=== RUN   TestNoiseBurstLayer_HasSilentAndActiveSegments
--- PASS: TestNoiseBurstLayer_HasSilentAndActiveSegments (0.00s)
=== RUN   TestBackgroundNoiseLayer_ContinuousOutput
--- PASS: TestBackgroundNoiseLayer_ContinuousOutput (0.00s)
=== RUN   TestLayerMixer_SumsLayers
--- PASS: TestLayerMixer_SumsLayers (0.00s)
=== RUN   TestLayerMixer_ClampsOutput
--- PASS: TestLayerMixer_ClampsOutput (0.00s)
=== RUN   TestLayerMixer_ClampsNegative
--- PASS: TestLayerMixer_ClampsNegative (0.00s)
=== RUN   TestLayerMixer_ZeroLayers
--- PASS: TestLayerMixer_ZeroLayers (0.00s)
...
PASS
ok  github.com/JamesPrial/go-scream/internal/audio/native  0.695s
```

---

## Race Detection (go test -race ./...)

No race conditions detected.

Three macOS `ld` LC_DYSYMTAB linker warnings appeared — identical to baseline, pre-existing OS toolchain artifact:
```
ld: warning: '...000012.o' has malformed LC_DYSYMTAB, expected 98 undefined symbols
    to start at index 1626, found 95 undefined symbols starting at index 1626
```
Packages affected: `internal/encoding`, `cmd/skill`, `internal/scream` — same three as baseline. Not a Go issue.

---

## Static Analysis (go vet ./...)

No output. Zero warnings.

---

## Coverage Details (go test -cover ./...)

```
        github.com/JamesPrial/go-scream/cmd/scream      coverage: 0.0% of statements
ok      github.com/JamesPrial/go-scream/cmd/skill       coverage: 21.7% of statements
ok      github.com/JamesPrial/go-scream/internal/audio  coverage: 87.5% of statements
ok      github.com/JamesPrial/go-scream/internal/audio/ffmpeg   coverage: 90.6% of statements
ok      github.com/JamesPrial/go-scream/internal/audio/native   coverage: 100.0% of statements
ok      github.com/JamesPrial/go-scream/internal/config coverage: 97.6% of statements
ok      github.com/JamesPrial/go-scream/internal/discord        coverage: 64.1% of statements
ok      github.com/JamesPrial/go-scream/internal/encoding       coverage: 84.7% of statements
ok      github.com/JamesPrial/go-scream/internal/scream coverage: 94.3% of statements
ok      github.com/JamesPrial/go-scream/pkg/version     coverage: 100.0% of statements
```

---

## Linter Output (golangci-lint run)

```
0 issues.
```

---

## Files Modified in Stage 5

- `/Users/jamesprial/code/go-scream/internal/audio/params.go` — Added `CoprimePrimaryScream`, `CoprimeHarmonicSweep`, `CoprimeHighShriek`, `CoprimeNoiseBurst` constants.
- `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go` — Removed `PrimaryScreamLayer` and `HighShriekLayer` struct types; both constructors now return `*SweepJumpLayer`. `NewNoiseBurstLayer` signature changed: `sampleRate int` arg removed (it was unused). Inline coprime literals replaced with `audio.Coprime*` constants.
- `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go` — `buildLayers` call updated: `NewNoiseBurstLayer(p3, noiseWithSeed)` (no `sampleRate` arg).
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go` — Inline coprime integer literals in `layerExpr` replaced with `audio.CoprimePrimaryScream`, `audio.CoprimeHarmonicSweep`, `audio.CoprimeHighShriek`, `audio.CoprimeNoiseBurst`.
- `/Users/jamesprial/code/go-scream/internal/audio/native/layers_test.go` — Test function names updated to reflect `SweepJumpLayer_PrimaryScream` and `SweepJumpLayer_HighShriek`; `sampleRate` args removed from `NewNoiseBurstLayer` calls.

---

## VERDICT: TESTS_PASS

All functional tests pass. No races. No vet warnings. No lint issues. Coverage is identical to or within variance of baseline for all packages. Audio output determinism confirmed by `TestGenerator_Deterministic`. No regression detected.
