# Design Review -- Stage 1: Core Types, Interfaces, Native Audio Synthesis

**Date:** 2026-02-18
**Reviewer:** Go Reviewer (Opus 4.6)
**Stage:** 1 -- Core types, interfaces, native audio synthesis
**Verdict:** REQUEST_CHANGES

---

## Overall Assessment

The package structure is well organized with a clean separation between the
abstract `audio` package (types, interfaces, presets) and the concrete
`audio/native` implementation. The 5-layer synthesis model and 6-stage filter
chain correctly mirror the original bot's design. Documentation is thorough,
naming is idiomatic, and test coverage is broad. However, there are several
correctness bugs, a significant performance concern, and a few design issues
that must be addressed before this stage can be approved.

---

## MUST FIX (Blocking)

### 1. `seededRandom` is not deterministic -- breaks reproducibility contract

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 172-177

```go
func seededRandom(rng *rand.Rand, step int64, coprime int64) float64 {
    h := step*coprime + rng.Int63n(10000)
    r := rand.New(rand.NewSource(h))
    return r.Float64()
}
```

This function calls `rng.Int63n(10000)` which advances the shared `*rand.Rand`
state. Because `seededRandom` is called once per sample per layer, the RNG
state diverges on every call. The `step` parameter was clearly intended to make
frequency jumps deterministic at discrete time boundaries (all samples within
the same `step` should get the same frequency). But because `rng.Int63n` is
called every sample, two calls at the same `step` value will produce different
results depending on how many samples preceded them.

This means:
- The function is NOT deterministic for a given `(step, coprime)` pair -- it
  depends on the call count.
- The `step`-based "hash" design pattern is defeated.
- The `TestNativeGenerator_Deterministic` test likely passes only because the
  entire generation loop has a fixed sample count, so the RNG sequence is the
  same. But any change to duration or sample rate would break it.

**Fix:** Remove the `rng` parameter entirely. Use a pure hash function over
`(step, coprime, layerSeed)` to produce a deterministic value for any given
step:

```go
func seededRandom(layerSeed, step, coprime int64) float64 {
    h := layerSeed ^ (step * coprime)
    r := rand.New(rand.NewSource(h))
    return r.Float64()
}
```

Or better yet, use a proper integer hash (e.g., splitmix64) to avoid allocating
a `rand.Rand` on every sample.

### 2. Massive per-sample allocation in `seededRandom` and `seededRandomContinuous`

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 172-186

Both `seededRandom` and `seededRandomContinuous` allocate a new `rand.New(rand.NewSource(h))`
on every call. For a 3-second stereo clip at 48kHz, `seededRandom` is called
~144,000 times per tonal layer (3 layers), and `seededRandomContinuous` is
called ~144,000 times per noise layer (2 layers). That is approximately
**720,000 heap allocations** per generation.

This is a critical performance issue for a bot that may need to generate audio
with low latency.

**Fix:** Replace with a stateless hash function that returns a float64 in [0,1)
without allocating. A simple approach:

```go
func hashToFloat(seed int64) float64 {
    // splitmix64
    seed = (seed ^ (seed >> 30)) * 0xbf58476d1ce4e5b9
    seed = (seed ^ (seed >> 27)) * 0x94d049bb133111eb
    seed = seed ^ (seed >> 31)
    return float64(uint64(seed)>>11) / float64(1<<53)
}
```

### 3. `NoiseBurstLayer` ignores `LayerParams.Amplitude` in favor of `NoiseParams.BurstAmp`

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 117-124

```go
func NewNoiseBurstLayer(p audio.LayerParams, noise audio.NoiseParams, sampleRate int) *NoiseBurstLayer {
    return &NoiseBurstLayer{
        rng:       rand.New(rand.NewSource(noise.BurstSeed)),
        burstRate: noise.BurstRate,
        threshold: noise.Threshold,
        amp:       noise.BurstAmp,  // <-- uses NoiseParams, not LayerParams
    }
}
```

The `LayerParams.Amplitude` field for layer index 3 is set by `Randomize()`
(line 112: `Amplitude: rf(0.1, 0.25)`) and validated by `Validate()`, but is
then completely ignored by the constructor. The `sampleRate` parameter is also
accepted but unused.

Similarly, `BackgroundNoiseLayer` (line 143) ignores `LayerParams.Amplitude`
for layer index 4, using `NoiseParams.FloorAmp` instead. In `Randomize()`,
`Layers[4].Amplitude` is set to `rf(0.05, 0.15)` and `Noise.FloorAmp` is
independently set to `rf(0.05, 0.15)` -- these are different random draws with
the same range, creating a confusing duplication.

**Fix:** Either (a) use `LayerParams.Amplitude` in the layer constructors and
remove the duplicate amplitude fields from `NoiseParams`, or (b) remove the
amplitude field from noise `LayerParams` entries and document that noise layers
draw amplitude from `NoiseParams`. The current half-and-half approach is a
latent bug source where presets could set conflicting values.

### 4. Field naming: `rng_` is not idiomatic Go

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 21, 53, 81

```go
type PrimaryScreamLayer struct {
    ...
    rng_ float64 // freq range
    ...
}
```

The trailing underscore convention (`rng_`) is borrowed from other languages and
is not idiomatic Go. The name also collides conceptually with `rng` (the RNG
field) on the same struct, making it confusing to read. This field represents
"frequency range" and should be named accordingly.

**Fix:** Rename to `freqRange` across all three layer types.

---

## SHOULD FIX (Non-blocking but recommended)

### 5. `AudioGenerator` interface naming violates Go convention

**File:** `/Users/jamesprial/code/go-scream/internal/audio/generator.go`, line 6

```go
type AudioGenerator interface {
    Generate(params ScreamParams) (io.Reader, error)
}
```

Go convention (per Effective Go) is to avoid stuttering: since this is in
package `audio`, callers would write `audio.AudioGenerator`, which stutters.
The idiomatic name would be `Generator`.

### 6. `Validate()` does not check `CrusherBits` range

**File:** `/Users/jamesprial/code/go-scream/internal/audio/params.go`, lines 143-168

The `Randomize()` function generates `CrusherBits` in `[6, 12]` and the
`FilterParams` doc comment says "Bit depth for bitcrusher (6-12)". However,
`Validate()` does not enforce this range. A `CrusherBits` of 0 or negative
would cause `math.Pow(2, 0)` = 1 level in the `Bitcrusher`, and
`CrusherBits` of -1 would produce a `levels` value less than 1, leading to
division by a value less than 1 in quantization -- producing nonsensical but
not crashing behavior.

**Fix:** Add `CrusherBits` validation in `Validate()`.

### 7. `Validate()` does not check `CompRatio` or `CompAttack`/`CompRelease`

A `CompRatio` of 0 would cause `1.0/f.ratio - 1.0` to produce `-Inf` in the
compressor's `math.Pow` call. A `CompAttack` of 0 would cause division by zero
in `math.Exp(-1.0 / 0)`. These should be validated.

### 8. Stereo output duplicates mono signal

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go`, lines 66-71

```go
binary.LittleEndian.PutUint16(sampleBuf, uint16(s16))
for ch := 0; ch < channels; ch++ {
    w.WriteByte(sampleBuf[0])
    w.WriteByte(sampleBuf[1])
}
```

The stereo output writes the identical sample to both left and right channels.
This is functionally mono audio in a stereo container. This is fine if
intentional (and may match the original bot), but it should be documented
explicitly in the `Generate` method's doc comment so future maintainers
understand this is by design and not a bug.

### 9. Missing doc comments on `Sample` method implementations

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`

The `Sample` methods on `PrimaryScreamLayer` (line 39), `HarmonicSweepLayer`
(line 71), `HighShriekLayer` (line 101), `NoiseBurstLayer` (line 126),
`BackgroundNoiseLayer` (line 147), and `LayerMixer` (line 162) all lack doc
comments. While these are interface implementations, adding brief doc comments
would improve readability, especially since each layer has distinct synthesis
behavior.

### 10. `Randomize` uses deprecated `rand.NewSource` pattern

**File:** `/Users/jamesprial/code/go-scream/internal/audio/params.go`, line 71

Since Go 1.20+, the top-level `rand` functions are automatically seeded and
the `rand.New(rand.NewSource(seed))` pattern is discouraged for new code.
Consider using `rand.New(rand.NewPCG(uint64(seed), 0))` for the newer,
non-deprecated API, especially since the module targets Go 1.25.

---

## Design Observations (Positive)

### Package Organization

The two-level package structure (`internal/audio` for types/interfaces,
`internal/audio/native` for the pure-Go implementation) is clean and will
support the planned `internal/audio/ffmpeg` backend in Stage 3 without any
refactoring of the interface layer.

### Interface Design

- `AudioGenerator` is minimal (single method). Good.
- `Filter` interface in `native` is appropriately scoped to the implementation
  package rather than polluting the `audio` package with implementation details.
- `Layer` interface in `native` follows the same pattern.
- `FilterChain` implementing `Filter` enables composability.

### Filter Chain Order

The chain in `NewFilterChainFromParams` follows the correct order:
Highpass -> Lowpass -> Bitcrusher -> Compressor -> VolumeBoost -> Limiter.
This maps to the FFmpeg chain: `highpass -> lowpass -> acrusher -> acompressor
-> volume -> alimiter`.

### 5-Layer Synthesis Model

The five layer types map correctly to the original synthesis model:
1. Primary scream (base frequency + random jumps + rising envelope)
2. Harmonic sweep (linear frequency sweep + jumps)
3. High shriek (high frequency + fast jumps + rising envelope)
4. Noise burst (gated white noise)
5. Background noise floor (continuous white noise)

### Test Quality

- Tests are well-structured with clear names following `TestType_Behavior`.
- Table-driven tests used appropriately (e.g., `TestValidate_InvalidLimiterLevel`,
  `TestBitcrusher_FullMix`).
- Good use of behavioral assertions (zero crossings for frequency accuracy,
  envelope rise verification).
- Interface compliance tests are a nice touch.
- Benchmarks included for performance-critical paths.

### Error Handling

- Sentinel errors with `errors.New` for simple cases.
- `LayerValidationError` with proper `Unwrap()` for `errors.Is`/`errors.As`.
- Error wrapping with `%w` in `Generate`.

---

## Test Coverage Gaps (for awareness, not blocking)

- No test for `Validate()` with negative `Duration` (only tests `Duration = 0`).
- No test for negative `SampleRate`.
- No test for `CrusherBits` edge cases (since validation is missing).
- `TestHighShriekLayer_EnvelopeRises` creates a `layer` variable on line 146
  that is immediately discarded (line 173: `_ = layer`). This is dead code.
- No test for `Randomize` verifying that layer-specific parameter ranges are
  honored (e.g., `BaseFreq` ranges per layer type).

---

## Summary of Required Changes

| # | Issue | Severity | File |
|---|---|---|---|
| 1 | `seededRandom` is not deterministic per step | Bug | `native/layers.go:172` |
| 2 | Per-sample `rand.New` allocation (720K allocs/generation) | Performance | `native/layers.go:172,181` |
| 3 | `NoiseBurstLayer`/`BackgroundNoiseLayer` ignore `LayerParams.Amplitude` | Design Bug | `native/layers.go:117,143` |
| 4 | `rng_` field naming is non-idiomatic | Style | `native/layers.go:21,53,81` |

---

## Verdict

**REQUEST_CHANGES**

Issues 1-3 are correctness and design concerns that should be resolved before
proceeding to Stage 2. Issue 1 in particular (`seededRandom` non-determinism
per step) undermines the reproducibility contract that the seed-based design is
built upon. Issue 2 creates an unnecessary performance bottleneck. Issue 3 is a
parameter confusion bug waiting to surface.

Issues 5-10 are recommended improvements that can be addressed in this stage or
deferred to a cleanup pass, at the team's discretion.
