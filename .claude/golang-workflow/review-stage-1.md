# Code Review: Stage 1 -- Core Types, Interfaces, Native Audio Synthesis

## Verdict: REQUEST_CHANGES

There are several issues that range from a significant correctness bug in the RNG/determinism design to minor naming and documentation gaps. The items marked **[MUST FIX]** are blocking; items marked **[SHOULD FIX]** are strongly recommended; items marked **[NIT]** are optional improvements.

---

## 1. [MUST FIX] `seededRandom` is non-deterministic for a given `(step, coprime)` pair

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 172-177

```go
func seededRandom(rng *rand.Rand, step int64, coprime int64) float64 {
	h := step*coprime + rng.Int63n(10000)
	r := rand.New(rand.NewSource(h))
	return r.Float64()
}
```

The function draws from `rng.Int63n(10000)` **every time it is called**, which means the RNG state advances on every call, not just at each new step. If `Sample(t)` is called multiple times for the same `step` value (which happens for every sample within the same frequency-jump interval), `seededRandom` returns a **different** value each time because `rng.Int63n` returns a different value each call.

This means frequency does **not** stay constant within a step interval -- it changes every sample. The comment says "deterministic frequency jumps at discrete time steps" but the implementation does not achieve this.

**Impact:** The frequency is effectively random on every sample rather than being piecewise-constant per step interval. This fundamentally changes the character of the synthesis. It also means the `TestNativeGenerator_Deterministic` test may pass (because the RNG seed path is deterministic) but the audio does not match the described algorithm.

**Suggested fix:** Use the step value itself to produce a deterministic hash without consuming from the stateful RNG, or cache the value per step:

```go
func seededRandom(baseSeed int64, step int64, coprime int64) float64 {
	h := baseSeed ^ (step * coprime)
	r := rand.New(rand.NewSource(h))
	return r.Float64()
}
```

Alternatively, store the current step and cached value on the layer struct, only recomputing when the step changes.

---

## 2. [MUST FIX] `seededRandomContinuous` collisions and poor noise quality

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 181-186

```go
func seededRandomContinuous(t float64, offset float64) float64 {
	h := int64(t*48000+offset) * 2654435761
	r := rand.New(rand.NewSource(h))
	return r.Float64()
}
```

Two problems:

**(a)** The hardcoded `48000` should be the actual sample rate. If the sample rate is ever changed (and `SampleRate` is configurable in `ScreamParams`), adjacent time values may map to the same integer and produce identical noise samples.

**(b)** Creating a new `rand.Rand` + `rand.NewSource` for **every single sample** is extremely expensive. At 48kHz stereo for 3 seconds that is 144,000 allocations just for noise layers alone (plus 144,000 more from `seededRandom` for each tonal layer). This function is called from `BackgroundNoiseLayer.Sample` and `NoiseBurstLayer.Sample` on every sample.

**Suggested fix:** Use a stateful `*rand.Rand` on the layer struct (seeded once at construction) instead of per-sample `NewSource` allocation. This also produces better-quality noise because consecutive samples are drawn from a well-distributed sequence rather than from correlated single-draw sources.

---

## 3. [MUST FIX] `NoiseBurstLayer` ignores `LayerParams.Amplitude` and `sampleRate`

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 117-124

```go
func NewNoiseBurstLayer(p audio.LayerParams, noise audio.NoiseParams, sampleRate int) *NoiseBurstLayer {
	return &NoiseBurstLayer{
		rng:       rand.New(rand.NewSource(noise.BurstSeed)),
		burstRate: noise.BurstRate,
		threshold: noise.Threshold,
		amp:       noise.BurstAmp,
	}
}
```

- The `sampleRate` parameter is accepted but never used (the hardcoded `48000` in `seededRandomContinuous` is used instead).
- `p.Amplitude` is never consulted; only `noise.BurstAmp` is used. The `LayerParams` argument `p` is effectively dead code. If these fields are intentionally redundant, the API surface should not accept a `LayerParams` that is ignored, or at minimum it should be documented.

---

## 4. [SHOULD FIX] Field name `rng_` is non-idiomatic

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`, lines 21, 53, 81

```go
rng_ float64 // freq range
```

The trailing underscore in `rng_` is used to avoid collision with the `rng *rand.Rand` field, but this is a Go anti-pattern. It also creates visual confusion since `rng_` looks like it refers to a random number generator. Use `freqRange` instead:

```go
type PrimaryScreamLayer struct {
	osc       *Oscillator
	rng       *rand.Rand
	baseFreq  float64
	freqRange float64
	jumpRate  float64
	amplitude float64
	rise      float64
}
```

This applies to `PrimaryScreamLayer`, `HarmonicSweepLayer`, and `HighShriekLayer`.

---

## 5. [SHOULD FIX] Missing doc comments on exported `Sample` methods

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`

The `Sample` methods on `PrimaryScreamLayer` (line 39), `HarmonicSweepLayer` (line 71), `HighShriekLayer` (line 101), `NoiseBurstLayer` (line 126), `BackgroundNoiseLayer` (line 147), and `LayerMixer` (line 162) are all missing doc comments. Since these are exported methods satisfying an exported interface, they should be documented.

---

## 6. [SHOULD FIX] Validate should check `CrusherBits` range

**File:** `/Users/jamesprial/code/go-scream/internal/audio/params.go`, lines 143-168

The `FilterParams.CrusherBits` field is documented as valid for range `6-12` (line 55), and `Randomize` generates values in `[6, 12]`, but `Validate()` never checks this. A `CrusherBits` of 0 would produce `levels = 1` (all samples quantized to 0), and negative values would produce `levels < 1`, both silently producing broken audio.

---

## 7. [SHOULD FIX] Validate should reject `HighpassCutoff >= LowpassCutoff`

**File:** `/Users/jamesprial/code/go-scream/internal/audio/params.go`

If someone sets `HighpassCutoff = 10000` and `LowpassCutoff = 100`, the filters will remove essentially all signal. While each cutoff is individually validated as non-negative, there is no cross-field validation. This is a straightforward footgun.

---

## 8. [SHOULD FIX] `int16` conversion may truncate toward zero instead of rounding

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go`, lines 62-64

```go
scaled := filtered * 32767.0
clamped := math.Max(-32768, math.Min(32767, scaled))
s16 := int16(clamped)
```

Go's `int16()` conversion truncates toward zero, not rounds. For audio, rounding to nearest is preferable to avoid a systematic DC bias. Consider:

```go
s16 := int16(math.Round(clamped))
```

This is a minor quality issue but matters for fidelity.

---

## 9. [SHOULD FIX] Per-sample allocation in `seededRandom` and `seededRandomContinuous`

Related to issue #2 above, but worth calling out for performance: both functions allocate a `rand.NewSource` and `rand.New` on **every call**. For a 3-second stereo clip at 48kHz, that is at minimum 144,000 samples, and each sample calls `seededRandom` 3 times (for the 3 tonal layers) plus `seededRandomContinuous` potentially 2 times (noise layers). That is approximately 720,000 heap allocations per generation. This will dominate the benchmark and create GC pressure.

---

## 10. [NIT] `Randomize(0)` uses `time.Now().UnixNano()` which is deprecated for seeding

**File:** `/Users/jamesprial/code/go-scream/internal/audio/params.go`, line 69

Since Go 1.20, `rand.NewSource` auto-seeds from a cryptographic source. However, here the seed is stored in `ScreamParams.Seed` for reproducibility, so explicit seeding is correct. The concern is that `time.Now().UnixNano()` has low entropy on some platforms (Windows pre-Go-1.22 had 100ns resolution). Consider using `crypto/rand` to generate the seed:

```go
import crand "crypto/rand"
import "encoding/binary"

var b [8]byte
crand.Read(b[:])
seed = int64(binary.LittleEndian.Uint64(b[:]))
```

This is a minor robustness improvement.

---

## 11. [NIT] `HighShriekLayer_EnvelopeRises` test creates an unused layer

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers_test.go`, lines 146, 173

```go
layer := NewHighShriekLayer(validHighShriekParams(), testSampleRate)
// ...
_ = layer
```

The layer is created and then immediately suppressed with `_ = layer`. This should be removed.

---

## 12. [NIT] `TestNativeGenerator_AllPresets` has an unused variable

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator_test.go`, line 169

```go
expectedBytes := int(params.Duration.Seconds()) * params.SampleRate * params.Channels * 2
```

This variable is computed but never used for assertion (the test uses `expectedBytesExact` instead). It should be removed or the logic consolidated.

---

## 13. [NIT] `LayerValidationError.Error()` uses `%s` instead of `%v` for wrapped error

**File:** `/Users/jamesprial/code/go-scream/internal/audio/errors.go`, line 24

```go
func (e *LayerValidationError) Error() string {
	return fmt.Sprintf("layer %d: %s", e.Layer, e.Err)
}
```

Using `%s` works but `%v` is more conventional for error formatting since it handles nil errors gracefully (prints `<nil>` instead of panicking on a nil error's `.Error()` call).

---

## Test Coverage Assessment (structural review only)

**Strengths:**
- Table-driven tests are used where appropriate (limiter levels, bitcrusher inputs).
- Interface compliance tests exist for all Filter implementations.
- Benchmarks are included for key hot paths.
- Determinism and different-seed tests are excellent.
- Edge cases like zero layers in the mixer, mono output, and unknown presets are covered.

**Gaps:**
- No test for `Validate` with negative amplitude (only tests > 1).
- No test for `Validate` with negative duration (only tests == 0).
- No test for `Validate` with `HighpassCutoff` or `LowpassCutoff` exactly at 0.
- No test for `seededRandom` or `seededRandomContinuous` directly (these are important enough to unit test, especially given the correctness concerns above).
- `HarmonicSweepLayer` has no amplitude-bounds or sweep-direction test (unlike PrimaryScream and HighShriek which test envelope behavior).
- No test verifies that stereo output contains duplicated L/R channels (the mono vs stereo byte count test is good but does not verify the actual sample duplication logic).

---

## Summary of Required Changes

| # | Severity | Issue |
|---|----------|-------|
| 1 | MUST FIX | `seededRandom` is not step-deterministic due to stateful RNG consumption |
| 2 | MUST FIX | `seededRandomContinuous` has hardcoded sample rate and per-sample allocation |
| 3 | MUST FIX | `NoiseBurstLayer` ignores `sampleRate` parameter and `LayerParams.Amplitude` |
| 4 | SHOULD FIX | `rng_` field naming is confusing and non-idiomatic |
| 5 | SHOULD FIX | Missing doc comments on exported `Sample` methods |
| 6 | SHOULD FIX | `Validate` does not check `CrusherBits` range |
| 7 | SHOULD FIX | No cross-field validation for highpass >= lowpass cutoff |
| 8 | SHOULD FIX | `int16` truncation instead of rounding in S16LE encoding |
| 9 | SHOULD FIX | Per-sample heap allocation in RNG functions |

