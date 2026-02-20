# Stage 5 Review: Consolidate native layer types and extract coprime constants

**Reviewer:** Go Reviewer Agent  
**Date:** 2026-02-19  
**Files Reviewed:**
- `/Users/jamesprial/code/go-scream/internal/audio/params.go`
- `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`
- `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go`
- `/Users/jamesprial/code/go-scream/internal/audio/native/layers_test.go`

---

## 1. Behavior Preservation (CRITICAL)

### 1a. Coprime constants match original hardcoded values

**Status: PASS**

In `/Users/jamesprial/code/go-scream/internal/audio/params.go` lines 32-37:

```go
const (
    CoprimePrimaryScream int64 = 137
    CoprimeHarmonicSweep int64 = 251
    CoprimeHighShriek    int64 = 89
    CoprimeNoiseBurst    int64 = 173
)
```

All four values (137, 251, 89, 173) are present and correctly typed as `int64`, matching the `coprime` field type in `SweepJumpLayer` and the `coprime` parameter type of `seededRandom`.

### 1b. No hardcoded coprime literals remain in source code

**Status: PASS**

A grep for the literal values `137`, `251`, `89`, and `173` across all `.go` files under `internal/` returned matches only in the constant definitions in `params.go`. The ffmpeg backend (`command.go`) uses `audio.CoprimePrimaryScream`, `audio.CoprimeHarmonicSweep`, `audio.CoprimeHighShriek`, and `audio.CoprimeNoiseBurst` exclusively. The native backend (`layers.go`) similarly references the constants through `audio.Coprime*`. No magic numbers remain.

### 1c. SweepJumpLayer.Sample() uses the coprime field correctly

**Status: PASS**

In `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go` lines 47-55:

```go
func (l *SweepJumpLayer) Sample(t float64) float64 {
    step := int64(t * l.jump)
    if step != l.curStep {
        l.curStep = step
        l.curFreq = l.base + l.freqRange*seededRandom(l.seed, step, l.coprime)
    }
    envelope := l.amp * (1 + l.rise*t)
    return envelope * l.osc.Sin(l.curFreq)
}
```

The `l.coprime` field is passed to `seededRandom`, which uses it as the third argument in the hash computation `layerSeed ^ (step * coprime)`. This is the same pattern that was previously hardcoded per-type. The `NewPrimaryScreamLayer` constructor sets `coprime: audio.CoprimePrimaryScream` (137) and `NewHighShriekLayer` sets `coprime: audio.CoprimeHighShriek` (89), preserving the original distinct stepping behavior.

### 1d. HarmonicSweepLayer and NoiseBurstLayer use constants correctly

**Status: PASS**

- `HarmonicSweepLayer.Sample()` (line 90): calls `seededRandom(l.seed, step, audio.CoprimeHarmonicSweep)` -- correctly uses the constant directly since this layer type was not consolidated.
- `NoiseBurstLayer.Sample()` (line 141): calls `seededRandom(l.burstSeed, step, audio.CoprimeNoiseBurst)` -- correctly uses the constant.

### 1e. Bit-identical output

**Status: PASS (by inspection)**

The `seededRandom` function signature and implementation are unchanged. The same (seed, step, coprime) triples are produced for any given `ScreamParams`, so audio output is bit-identical. The `SweepJumpLayer.Sample` method performs the same arithmetic as the original `PrimaryScreamLayer.Sample` and `HighShriekLayer.Sample` -- the only difference is the coprime value comes from a struct field instead of a literal, which produces identical machine code.

---

## 2. Structural Quality

### 2a. SweepJumpLayer properly replaces both old types

**Status: PASS**

A grep for `type PrimaryScreamLayer` and `type HighShriekLayer` across all `.go` files found no matches. The old struct types are fully removed. The single `SweepJumpLayer` struct (lines 17-28) serves both purposes, differentiated by the `coprime` field value set by the respective constructors.

### 2b. Constructor names preserved

**Status: PASS**

- `NewPrimaryScreamLayer` (line 31): returns `*SweepJumpLayer` with `coprime: audio.CoprimePrimaryScream`
- `NewHighShriekLayer` (line 98): returns `*SweepJumpLayer` with `coprime: audio.CoprimeHighShriek`

Both constructors maintain their original names and signatures (accepting `audio.LayerParams` and `sampleRate int`), providing backward compatibility for callers in `generator.go` (lines 101, 103).

### 2c. Coprime constants used in both backends

**Status: PASS**

- **Native backend** (`layers.go`): Uses `audio.CoprimePrimaryScream` (line 40), `audio.CoprimeHarmonicSweep` (line 90), `audio.CoprimeHighShriek` (line 107), `audio.CoprimeNoiseBurst` (line 141).
- **FFmpeg backend** (`command.go`): Uses `audio.CoprimePrimaryScream` (line 68), `audio.CoprimeHarmonicSweep` (line 82), `audio.CoprimeHighShriek` (line 96), `audio.CoprimeNoiseBurst` (line 111).

Both backends reference the same constants from the shared `audio` package, eliminating the risk of drift between backends.

### 2d. NewNoiseBurstLayer signature change

**Status: PASS**

In `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go` line 124:

```go
func NewNoiseBurstLayer(p audio.LayerParams, noise audio.NoiseParams) *NoiseBurstLayer {
```

The `sampleRate` parameter has been removed as approved. The `NoiseBurstLayer` struct does not contain an `Oscillator` (it uses an `*rand.Rand` directly), so `sampleRate` was indeed unused and its removal is correct. The caller in `generator.go` line 104 passes only `(p3, noiseWithSeed)`, matching the new signature.

---

## 3. API Break Detection

### 3a. Approved changes only

**Status: PASS**

All changes match the approved API changes table exactly:

| Approved Change | Verified |
|---|---|
| RENAMED `PrimaryScreamLayer` to `SweepJumpLayer` | Yes -- old type removed, new type present |
| RENAMED `HighShriekLayer` to `SweepJumpLayer` | Yes -- old type removed, same struct used |
| NEW `CoprimePrimaryScream` (const) | Yes -- `int64 = 137` |
| NEW `CoprimeHarmonicSweep` (const) | Yes -- `int64 = 251` |
| NEW `CoprimeHighShriek` (const) | Yes -- `int64 = 89` |
| NEW `CoprimeNoiseBurst` (const) | Yes -- `int64 = 173` |
| SIGNATURE `NewNoiseBurstLayer` drops `sampleRate` | Yes -- now `(p, noise)` only |

### 3b. No unplanned public API changes

**Status: PASS**

No other exported symbols were added, removed, or modified. The `Layer` interface, `LayerMixer`, `HarmonicSweepLayer`, `BackgroundNoiseLayer`, `NoiseBurstLayer`, `Oscillator`, and `Generator` types are unchanged. The `audio.ScreamParams`, `audio.LayerParams`, and related types in `params.go` are unchanged beyond the new `const` block.

---

## 4. Code Quality

### 4a. Documentation

**Status: PASS**

All exported items have documentation:
- `SweepJumpLayer` struct (line 14-16): documents the coprime-parameterised design and its dual use.
- `NewPrimaryScreamLayer` (line 30): documents purpose.
- `NewHighShriekLayer` (line 96-97): documents that it returns a `*SweepJumpLayer` configured with `CoprimeHighShriek`.
- `Sample` method (line 45-46): documents the frequency-jump behavior including the coprime dependency.
- Coprime constants (lines 30-31): block comment explains they are used by both backends for deterministic frequency stepping.

### 4b. Error handling

**Status: N/A**

No error paths exist in the layer constructors or `Sample` methods. This is appropriate -- these are pure computation functions with no I/O or fallible operations.

### 4c. Naming conventions

**Status: PASS**

- `SweepJumpLayer` accurately describes the synthesis method (frequency sweeps with discrete jumps).
- `Coprime*` constant names clearly identify both the purpose (coprime stepping) and the layer they belong to.
- Test names follow `TestSweepJumpLayer_PrimaryScream_*` and `TestSweepJumpLayer_HighShriek_*` conventions, clearly identifying which constructor path is being tested.

### 4d. Test coverage assessment

**Status: PASS with one observation**

The test file properly covers:
- Non-zero output for both constructors (`TestSweepJumpLayer_PrimaryScream_NonZeroOutput`, `TestSweepJumpLayer_HighShriek_NonZeroOutput`)
- Amplitude bounds for primary scream (`TestSweepJumpLayer_PrimaryScream_AmplitudeBounds`)
- Envelope rise behavior for high shriek (`TestSweepJumpLayer_HighShriek_EnvelopeRises`)
- NoiseBurstLayer gating behavior (`TestNoiseBurstLayer_HasSilentAndActiveSegments`)
- BackgroundNoiseLayer continuous output (`TestBackgroundNoiseLayer_ContinuousOutput`)
- LayerMixer sum, clamping, and zero-layer edge cases (four tests)
- Benchmarks updated to use new constructor names

**Observation (non-blocking):** There is no test that explicitly verifies the two `SweepJumpLayer` constructors produce _different_ output for identical `LayerParams` (due to different coprime values). While the existing tests implicitly validate each path works, a test like `TestSweepJumpLayer_DifferentCoprimesProduceDifferentOutput` would strengthen the guarantee that the coprime parameterization is actually effective. This is a minor test improvement suggestion, not a blocking issue.

### 4e. Minor style note in test file

In `TestSweepJumpLayer_HighShriek_EnvelopeRises` (lines 145-181), a `layer` variable is created on line 146 but immediately discarded with `_ = layer` on line 173, because the `calcAvgAbs` closure creates fresh layers internally. This is harmless but slightly confusing. The initial `layer` creation could simply be removed.

---

## 5. Summary

The refactoring is clean and correct:

1. Two structurally identical types (`PrimaryScreamLayer`, `HighShriekLayer`) are consolidated into one (`SweepJumpLayer`) parameterized by a `coprime` field, with no behavioral change.
2. Four coprime magic numbers are extracted to named constants in the shared `audio` package, used consistently by both native and ffmpeg backends.
3. The unused `sampleRate` parameter is removed from `NewNoiseBurstLayer`.
4. All changes match the approved API changes exactly. No unplanned changes detected.
5. Documentation, naming, and test coverage are all adequate.

---

## Verdict: **APPROVE**

All review criteria are satisfied. The two non-blocking observations (coprime differentiation test, unused variable in test) are minor improvements that can be addressed in a follow-up if desired.
