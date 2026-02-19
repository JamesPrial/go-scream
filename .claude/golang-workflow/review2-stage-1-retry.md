# Design Review: Stage 1 -- Core Types, Interfaces, Native Audio Synthesis (Retry after Fix Cycle)

**Reviewer:** Go Reviewer Agent
**Date:** 2026-02-18
**Scope:** All files in `internal/audio/` and `internal/audio/native/`
**Focus:** Verify previous MUST FIX issues are resolved; assess overall design quality

---

## Previous MUST FIX Issue Verification

### 1. seededRandom non-determinism -- RESOLVED

**Previous issue:** `seededRandom` consumed stateful RNG on every call, producing different values for the same `(step, coprime)` input depending on call order.

**Current implementation** (`/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 186-202):

```go
func splitmix64(seed int64) float64 {
    s := uint64(seed)
    s = (s ^ (s >> 30)) * 0xbf58476d1ce4e5b9
    s = (s ^ (s >> 27)) * 0x94d049bb133111eb
    s = s ^ (s >> 31)
    return float64(s>>11) / float64(1<<53)
}

func seededRandom(layerSeed, step, coprime int64) float64 {
    h := layerSeed ^ (step * coprime)
    return splitmix64(h)
}
```

**Verdict:** Properly fixed. `splitmix64` is a well-known stateless bijective hash. The function is now purely deterministic for any given `(layerSeed, step, coprime)` triple with no shared mutable state. The `float64(s>>11) / float64(1<<53)` conversion correctly produces uniform values in `[0, 1)` using the standard 53-bit mantissa technique.

### 2. Per-sample allocation -- RESOLVED

**Previous issue:** Noise layers allocated `rand.New(rand.NewSource(...))` on every sample call.

**Current implementation:** `NoiseBurstLayer` and `BackgroundNoiseLayer` now store a `*rand.Rand` field (`noiseRng`) initialized once at construction time (lines 116-127 and 153-158). Per-sample calls use `l.noiseRng.Float64()` which advances the existing RNG state without any allocation.

**Verdict:** Properly fixed. Zero per-sample allocations for noise generation.

### 3. NoiseBurstLayer/BackgroundNoiseLayer amplitude confusion -- RESOLVED

**Previous issue:** Noise layers used `LayerParams.Amplitude` instead of `NoiseParams.BurstAmp`/`FloorAmp`.

**Current implementation:**
- `NoiseBurstLayer` uses `noise.BurstAmp` (line 129)
- `BackgroundNoiseLayer` uses `noise.FloorAmp` (line 157)

**Verdict:** Properly fixed. Each noise layer uses its dedicated amplitude from `NoiseParams`.

### 4. rng_ field naming -- RESOLVED

**Previous issue:** Field named `rng_` violated Go naming conventions; `rng *rand.Rand` was ambiguous.

**Current implementation:** The field that was `rng_` is now `freqRange float64` (semantically correct). The RNG field for noise layers is now `noiseRng *rand.Rand` (descriptive and unambiguous).

**Verdict:** Properly fixed. All field names follow Go conventions.

---

## Design Review -- Current State

### Package Organization

The two-package layout is clean and appropriate:

- **`internal/audio`** -- Public types, interfaces, validation, presets. Zero dependencies on synthesis implementation.
- **`internal/audio/native`** -- Pure Go synthesis implementation. Depends on `internal/audio` for parameter types.

This separation enables future alternative backends (e.g., FFmpeg-based, WASM) to implement `audio.AudioGenerator` without touching the native package. The `internal/` prefix correctly prevents external consumers from depending on these packages directly.

### Interface Design

**`audio.AudioGenerator`** (`/Users/jamesprial/code/go-scream/internal/audio/generator.go`):

```go
type AudioGenerator interface {
    Generate(params ScreamParams) (io.Reader, error)
}
```

Minimal and appropriate. Returning `io.Reader` rather than `[]byte` allows for streaming in future implementations without breaking the interface. The single-method interface follows Go idioms.

**`native.Layer`** (`/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, line 11):

```go
type Layer interface {
    Sample(t float64) float64
}
```

Clean single-method interface. Package-internal (unexported from `native`), which is correct since callers interact through `AudioGenerator`.

**`native.Filter`** (`/Users/jamesprial/code/go-scream/internal/audio/native/filters.go`, line 11):

```go
type Filter interface {
    Process(sample float64) float64
}
```

Symmetric with `Layer`. Both `FilterChain` and `LayerMixer` compose their respective interface, which is idiomatic Go.

### Error Handling

**`/Users/jamesprial/code/go-scream/internal/audio/errors.go`:**

- Sentinel errors are package-level `var` declarations using `errors.New` -- correct pattern.
- `LayerValidationError` properly implements both `Error()` and `Unwrap()`, enabling `errors.Is` and `errors.As` chains.
- The generator wraps validation errors with `%w` format verb (line 29 of `generator.go`): `fmt.Errorf("invalid params: %w", err)` -- correct.

### Naming Conventions

All exported types, functions, and methods have doc comments. Naming follows Go conventions:

- Types: `PrimaryScreamLayer`, `FilterChain`, `ScreamParams` -- PascalCase, descriptive.
- Constructors: `NewPrimaryScreamLayer`, `NewFilterChain` -- standard `New*` prefix.
- Unexported helpers: `splitmix64`, `seededRandom`, `clamp`, `buildLayers` -- lowercase, concise.
- Constants: `PresetClassic`, `LayerPrimaryScream` -- PascalCase with type prefix.

### splitmix64 / seededRandom Design Assessment

The `splitmix64` implementation is a faithful port of the well-known splitmix64 hash by Sebastiano Vigna. The mixing constants (`0xbf58476d1ce4e5b9`, `0x94d049bb133111eb`) are correct.

The `seededRandom` combinator uses `layerSeed ^ (step * coprime)` to derive a unique hash input per (layer, time-step) pair. The coprime values (137, 251, 89, 173) are chosen to be distinct primes, which ensures the step multipliers create linearly independent sequences modulo 2^64. This is a reasonable approach for decorrelating layers.

**One minor observation:** When `step == 0`, all layers receive `splitmix64(layerSeed ^ 0) = splitmix64(layerSeed)`, meaning the first step's randomness depends only on the layer seed. This is acceptable because the layer seeds are themselves derived with different prime multipliers in `buildLayers` (line 91: `globalSeed * 1000003`, etc.), so the values will still differ across layers.

### Validation Completeness

`Validate()` in `/Users/jamesprial/code/go-scream/internal/audio/params.go` checks:
- Duration > 0
- SampleRate > 0
- Channels in {1, 2}
- All layer amplitudes in [0, 1]
- Filter cutoffs >= 0
- CrusherBits in [1, 16]
- LimiterLevel in (0, 1]

This covers the critical invariants. The `NoiseParams` fields (`BurstRate`, `Threshold`, `BurstAmp`, `FloorAmp`) are not validated, but since they have no constraints that would cause panics or division-by-zero (burst rate of 0 just means `step` is always 0, which is harmless), this is acceptable for Stage 1.

### Test Quality

Tests are well-structured and cover the key behavioral contracts:

- **Oscillator tests:** Frequency accuracy via zero-crossing counting, amplitude bounds, phase continuity, known values at cardinal phases, and reset behavior.
- **Layer tests:** Non-zero output, amplitude envelope bounds, envelope rising behavior, gated noise silence/active segments, background noise continuity, mixer summation and clamping.
- **Filter tests:** DC rejection (highpass), high-frequency passthrough (highpass), DC passthrough (lowpass), high-frequency attenuation (lowpass), bitcrusher quantization at multiple mix levels, compressor above/below threshold and sign preservation, volume boost at 0dB/+6dB/-6dB, limiter clipping bounds, filter chain ordering, interface compliance for all filter types.
- **Generator tests:** Correct byte count, non-silence, determinism, seed divergence, all presets generate successfully, invalid params rejection, mono output, s16le range verification, interface compliance.

**Minor note on test structure:** The `TestHighShriekLayer_EnvelopeRises` test (layers_test.go, line 145) creates a `layer` variable on line 149 that is immediately discarded in favor of fresh layers inside `calcAvgAbs`. The `_ = layer` suppression on line 173 acknowledges this. This is cosmetically untidy but functionally correct -- the fresh layers are necessary because the oscillator is stateful.

### Potential Concern: `for range channels` syntax

In `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go`, line 71:

```go
for range channels {
```

This uses the Go 1.22+ `for range <integer>` syntax. The `go.mod` specifies `go 1.25.7`, so this is valid. Worth noting for awareness if the minimum Go version ever needs to be lowered.

### Potential Concern: Preset `Seed` field is zero

All presets in `/Users/jamesprial/code/go-scream/internal/audio/presets.go` have `Seed: 0` (zero value, not explicitly set). In `buildLayers`, the global seed is XOR-mixed into per-layer seeds: `lp[0].Seed ^ (globalSeed * 1000003)`. With `globalSeed == 0`, the per-layer seeds remain unchanged. This means preset audio is fully determined by the layer-level seeds, which is likely intentional (presets should be reproducible). No bug here, but documenting this design choice in a comment on the `Seed` field or `buildLayers` would aid maintainability.

---

## Summary

All four previously identified MUST FIX issues have been properly resolved:

| # | Issue | Status |
|---|-------|--------|
| 1 | `seededRandom` non-determinism | Fixed via stateless `splitmix64` hash |
| 2 | Per-sample allocation in noise layers | Fixed via construction-time `*rand.Rand` |
| 3 | Noise layer amplitude confusion | Fixed: uses `NoiseParams.BurstAmp`/`FloorAmp` |
| 4 | `rng_` field naming | Fixed: renamed to `freqRange`; noise RNG is `noiseRng` |

The codebase demonstrates clean package organization, idiomatic Go interface design, thorough error handling with proper wrapping, consistent naming conventions, complete documentation on all exported items, and comprehensive test coverage across all components.

No new correctness issues, design flaws, or blocking concerns were found.

---

## VERDICT: APPROVE

The design meets quality standards. All previous MUST FIX issues are properly resolved. The code is ready to proceed to Stage 2.
