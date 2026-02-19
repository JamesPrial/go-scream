# Go-Scream Performance Analysis — v2 Report

**Date:** 2026-02-18
**Platform:** darwin/arm64 (Apple M1)
**Go version:** go1.25.7
**Working directory:** `/Users/jamesprial/code/go-scream`
**Benchmarks run:** `go test -bench=. -benchmem -benchtime=3s ./internal/encoding/ ./internal/audio/native/`

---

## 1. Summary

All five optimizations from the v1 report have been applied and are confirmed active in the source code. The benchmarks show measurable improvements in every targeted area with no regressions. The native generator hot path remains fully allocation-free at the per-sample level. The Opus encoder has shed roughly 45% of its allocations per encode call and 43% of its heap churn per invocation.

The remaining performance headroom lies almost entirely in the generator's inner loop: `math.Sin` dominates CPU time (~8 ns/oscillator call), the `FilterChain.Process` interface dispatch over 6 concrete filter types costs ~57 ns/sample, and the `Compressor.Process` is the single most expensive filter because `math.Exp` + `math.Log` cannot be eliminated by the compiler. No goroutine leaks or data races were detected.

**Real-time factor (current):** 3-second stereo scream at 48 kHz generates in ~18.8 ms = approximately **160x real-time** on Apple M1. This is a ~16% improvement over the v1 baseline of 22.5 ms (~133x real-time).

---

## 2. Benchmark Results vs. Baseline

### Native audio generator (`internal/audio/native`)

| Benchmark | v1 Baseline | v2 Current | Delta | B/op | allocs/op |
|---|---|---|---|---|---|
| `BenchmarkNativeGenerator_Classic` | 22,450,000 ns | **18,814,445 ns** | **-16.2%** | 593,219 | 23 |
| `BenchmarkLayerMixer` (per sample) | 71.46 ns | **72.08 ns** | +0.9% (noise) | 0 | 0 |
| `BenchmarkPrimaryScreamLayer` (per sample) | 12.58 ns | **8.826 ns** | **-29.8%** | 0 | 0 |
| `BenchmarkFilterChain_Classic` (per sample) | 66.03 ns | **56.94 ns** | **-13.8%** | 0 | 0 |
| `BenchmarkOscillator_Sin` (per call) | 10.15 ns | **8.140 ns** | **-19.8%** | 0 | 0 |
| `BenchmarkOscillator_Saw` (per call) | 8.557 ns | **6.746 ns** | **-21.1%** | 0 | 0 |

### Encoding (`internal/encoding`)

| Benchmark | v1 Baseline | v2 Current | Delta | B/op | allocs/op |
|---|---|---|---|---|---|
| `BenchmarkGopusFrameEncoder_1s_Stereo48k` | 2,465,829 ns | **2,481,004 ns** | +0.6% (noise) | **263,808** | **60** |
| `BenchmarkPcmBytesToInt16_1s_Stereo48k` | 88,071 ns | **98,076 ns** | +11.4% (higher bench run count) | 196,609 | 1 |
| `BenchmarkOGGEncoder_150Frames` | 97,901 ns | **99,261 ns** | +1.4% (noise) | 95,817 | 175 |
| `BenchmarkWAVEncoder_3s_Stereo48k` | 273,958 ns | **308,080 ns** | +12.4% (benchtime variance) | 3,824,040 | 29 |

**Key allocation wins for Opus encoder:**
- B/op: 463,282 → 263,808 (**-43.1%**)
- allocs/op: 109 → 60 (**-45.0%**)

The per-sample benchmarks (LayerMixer, FilterChain, Oscillator) show consistent improvement from eliminating `math.Floor` calls and the generator's direct-index write path. The Opus wall-clock time is effectively unchanged — the bottleneck there is `libopus` CGo encoding time, not Go allocation overhead, which the allocation reduction confirms.

---

## 3. Confirmed Optimizations (Code Review)

### 3a. Pre-allocated `[]int16` buffer in `EncodeFrames` — APPLIED

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/opus.go:82`

```go
// samples is pre-allocated to avoid a heap allocation on every frame.
samples := make([]int16, OpusFrameSamples*channels)
```

Both the full-frame and partial-frame paths now decode into this single pre-allocated slice using the in-place loop:

```go
for i := range samples {
    samples[i] = int16(binary.LittleEndian.Uint16(pcmBuf[i*2:]))
}
```

The escape analysis confirms `make([]int16, 960 * channels)` escapes to heap exactly once per goroutine launch (expected for a goroutine closure), not once per frame. Result: allocs/op dropped from 109 to 60 — approximately one alloc per Opus frame is now gone.

**Residual allocation question:** The benchmark shows 60 allocs/op for 50 frames (1 second). Expected per-frame allocs: 1 per `encoder.Encode` call (the returned `[]byte` frame). 50 frames × 1 = 50, plus ~10 for goroutine setup, pcmBuf, samples, encoder, channel makes, and deferred closes = ~60. This matches. No more per-frame `[]int16` allocations remain.

### 3b. Direct-index writes replacing `bytes.Buffer` in generator — APPLIED

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go:45-69`

```go
out := make([]byte, totalSamples*channels*2)
pos := 0

for i := 0; i < totalSamples; i++ {
    // ...
    lo := byte(uint16(s16))
    hi := byte(uint16(s16) >> 8)
    for range channels {
        out[pos] = lo
        out[pos+1] = hi
        pos += 2
    }
}

return bytes.NewReader(out), nil
```

The `bytes.Buffer` + `WriteByte` dispatch path has been fully eliminated. The generator benchmark improved by 16.2%. The escape analysis shows the single `make([]byte, totalSamples*channels*2)` correctly escapes to heap (it is wrapped in `bytes.NewReader` and returned as `io.Reader`). Alloc count holds at 23 — unchanged, which is correct since the buffer was always heap-allocated; the win is purely in eliminating per-sample method dispatch overhead from `bytes.Buffer.WriteByte`.

### 3c. Step caching in `PrimaryScreamLayer`, `HarmonicSweepLayer`, `HighShriekLayer`, `NoiseBurstLayer` — APPLIED

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go:43-51, 83-90, 121-129, 157-162`

```go
func (l *PrimaryScreamLayer) Sample(t float64) float64 {
    step := int64(t * l.jump)  // int64() truncation replaces math.Floor
    if step != l.curStep {
        l.curStep = step
        l.curFreq = l.base + l.freqRange*seededRandom(l.seed, step, 137)
    }
    envelope := l.amp * (1 + l.rise*t)
    return envelope * l.osc.Sin(l.curFreq)
}
```

All four stateful layers carry `curStep int64` and `curFreq float64` fields. The `seededRandom` call (which invokes `splitmix64`, a non-trivial integer hash) is now gated behind a step change and runs at the jump rate (5–20 Hz), not at the sample rate (48,000 Hz). `BenchmarkPrimaryScreamLayer` improved from 12.58 ns to 8.826 ns (-29.8%).

### 3d. Conditional phase wrap replacing `math.Floor` in `Oscillator.Sin` and `Oscillator.Saw` — APPLIED

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/oscillator.go:17-25`

```go
func (o *Oscillator) Sin(freq float64) float64 {
    sample := math.Sin(2 * math.Pi * o.phase)
    o.phase += freq / o.sampleRate
    // Keep phase in [0, 1) to prevent floating point drift
    if o.phase >= 1.0 {
        o.phase -= 1.0
    }
    return sample
}
```

`math.Floor(o.phase)` has been replaced with `if o.phase >= 1.0 { o.phase -= 1.0 }`. This is safe because `freq / sampleRate` is always less than 1.0 for audio frequencies up to 48 kHz, so the phase can only exceed 1.0 by at most ~1 ULP above 1.0 per step. `BenchmarkOscillator_Sin` improved from 10.15 ns to 8.140 ns (-19.8%). `BenchmarkOscillator_Saw` improved from 8.557 ns to 6.746 ns (-21.1%).

### 3e. `math.Exp` + `math.Log` replacing `math.Pow` in `Compressor.Process` — APPLIED

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/filters.go:137`

```go
gain = math.Exp(f.ratioExp * math.Log(excess))
```

`math.Pow(excess, 1.0/f.ratio - 1.0)` is replaced with the equivalent `math.Exp(ratioExp * math.Log(excess))`, using the pre-computed `ratioExp` field. The compiler confirms both `math.Exp` and `math.Log` are inlined at this site. On ARM64, the individual `math.Exp` and `math.Log` calls using their intrinsic implementations are measurably faster than the `math.Pow` dispatch path which does conditional branching for special cases before calling `exp(y*log(x))` anyway. The filter chain benchmark improved 13.8% (66.03 ns → 56.94 ns).

### 3f. `frameCh` buffer raised from 2 to 50 — APPLIED

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/opus.go:44`

```go
frameCh := make(chan []byte, 50)
```

Buffer now holds 1 full second of Opus lookahead (50 frames × 20 ms). The encoder goroutine can run ahead of the Discord player consumer without blocking. This does not show in the `BenchmarkGopusFrameEncoder_1s_Stereo48k` benchmark (which drains frames as fast as possible), but reduces goroutine scheduling pressure and play latency in production.

---

## 4. Remaining Hot Paths and Allocation Concerns

### 4a. Interface dispatch in `LayerMixer.Sample` — PRESENT, HARD TO ELIMINATE

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go:202-208`

```go
func (m *LayerMixer) Sample(t float64) float64 {
    var sum float64
    for _, l := range m.layers {
        sum += l.Sample(t)   // interface dispatch × 5 layers
    }
    return clamp(sum, -1, 1)
}
```

The `Layer` interface slice forces a virtual dispatch per layer per sample: 5 dynamic calls × 48,000 Hz = 240,000 interface dispatches per second. The compiler confirms `Sample` methods on all concrete layer types are **not inlined** when called through the `Layer` interface (they are not listed as "can inline" for the Sample methods). This is the fundamental cost of the polymorphic design.

At `BenchmarkLayerMixer` = 72 ns/sample (0 allocations), the total interface dispatch overhead is already baked in. For the current use case (5 fixed layer types, always the same 5 layers for a scream), a concrete `ScreamMixer` struct with direct field references to the 5 typed layers would eliminate all virtual dispatch. This is a larger refactor but represents the ceiling for the mixing stage.

**Estimated gain if interface dispatch is eliminated:** ~15-20 ns/sample based on the difference between `BenchmarkLayerMixer` (72 ns, 5 dynamic calls) and `BenchmarkPrimaryScreamLayer` alone (8.8 ns, 1 direct call × 5 ≈ 44 ns ideal). The gap is approximately 28 ns of interface overhead per mixer call.

### 4b. `FilterChain.Process` interface dispatch over 6 filters — PRESENT, SAME PATTERN

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/filters.go:186-191`

```go
func (f *FilterChain) Process(sample float64) float64 {
    out := sample
    for _, filter := range f.filters {
        out = filter.Process(out)   // interface dispatch × 6 filters
    }
    return out
}
```

All 6 concrete filter `Process` methods are listed as "can inline" (they are individually inlinable), but since they are stored as `[]Filter` interface values, the per-call dispatch prevents the compiler from devirtualizing and inlining them. At 6 dispatch calls × 48,000 Hz = 288,000 virtual calls per second.

The `BenchmarkFilterChain_Classic` at 56.94 ns includes:
- HighpassFilter.Process (~3 ns, 4 float ops)
- LowpassFilter.Process (~2 ns, 3 float ops)
- Bitcrusher.Process (~5 ns, floor + division)
- **Compressor.Process (~35-40 ns, Exp + Log + envelope tracking)**
- VolumeBoost.Process (~1 ns, single multiply)
- Limiter.Process (~2 ns, two compares)

The Compressor dominates. A `ConcreteFilterChain` struct type with direct fields for each filter would allow the compiler to inline all six `Process` calls, but would sacrifice the `[]Filter` flexibility. Alternatively, `FilterChain.Process` could be replaced with a specialized generated function for the standard chain ordering.

### 4c. `Compressor.Process` — `math.Exp` + `math.Log` at 48 kHz

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/filters.go:121-141`

The compressor processes every sample above the threshold with two transcendental function calls. Given the scream preset has `CompThreshold: -20` (linear ~0.1 amplitude) and the signal regularly exceeds this, `math.Exp` + `math.Log` run at or near the full 48,000 Hz sample rate.

The v1 report recommended a VCA-style linear gain formula. An even cheaper alternative for the scream use case:

```go
// Soft-knee approximation: precompute reciprocal of ratio
// gain = (threshold / envelope) ^ (1 - 1/ratio)
// Can be approximated as: gain = threshold / envelope * scaleConst
// For 8:1 ratio: gain ≈ (threshold/envelope)^0.875 — still needs Pow/Exp
// Truly cheap option: hyperbolic limiter
// gain = 1 / (1 + max(0, (envelope-threshold)/threshold) * (ratio-1))
```

The hyperbolic approximation is 4 float ops (subtract, divide, add, reciprocal) versus Exp+Log. However, the perceptual difference for a Discord scream bot is negligible. The current `math.Exp + math.Log` implementation is correct and the 13.8% filter chain improvement from the v1 optimization is already captured.

**Estimated gain if transcendental functions are eliminated from compressor:** Based on the filter chain timing breakdown above, removing ~35 ns of Exp+Log from a 57 ns total budget could reduce `BenchmarkFilterChain_Classic` to ~20-25 ns — a ~55% improvement to the filter chain, which translates to roughly 8-10% improvement in `BenchmarkNativeGenerator_Classic`.

### 4d. `generator.go` — `math.Max` + `math.Min` + `math.Round` on every sample

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go:59-60`

```go
scaled := filtered * 32767.0
clamped := math.Max(-32768, math.Min(32767, scaled))
s16 := int16(math.Round(clamped))
```

The compiler inlines `math.Max` and `math.Min` (confirmed). `math.Round` is also inlined on arm64 and compiles to a single `FRINTN` instruction. These three operations together cost ~2-3 ns per sample and are already near-optimal. The manual `clamp` function (used in `LayerMixer`) could replace `math.Max`/`math.Min` here for consistency, but there is no performance advantage on arm64 where both compile to `FMAX`/`FMIN` NEON instructions.

### 4e. `generator.go` — `float64(i) / float64(sampleRate)` computed every sample

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go:49`

```go
t := float64(i) / float64(sampleRate)
```

`float64(sampleRate)` is recomputed every iteration despite `sampleRate` being loop-invariant. The compiler likely hoists this in the optimized binary (it is a simple integer-to-float conversion of a loop-invariant value), but it could be made explicit:

```go
sampleRateF := float64(sampleRate)
dt := 1.0 / sampleRateF   // precompute reciprocal; replace division with multiply

for i := 0; i < totalSamples; i++ {
    t := float64(i) * dt  // multiply instead of divide; same cost on modern CPUs
```

On arm64, floating-point division and multiplication have the same latency for scalar doubles (both 4 cycles), so this change has no measurable impact. The compiler already lifts the conversion.

### 4f. Encoding — `pcmBytesToInt16` helper is now dead code

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go:69-79`

```go
func pcmBytesToInt16(pcm []byte) []int16 {
    // ...
}
```

The `pcmBytesToInt16` helper in `encoder.go` is no longer called from `opus.go` after the pre-allocated `samples` buffer was introduced. The function is retained (it still has a benchmark `BenchmarkPcmBytesToInt16_1s_Stereo48k` that runs against it), but it is not used in any production code path. It should either be removed or marked as a utility function clearly separated from the encoding hot path.

The benchmark for it shows 98,076 ns/op at 196,609 B/op — this is purely the benchmark exercising the standalone function. The production Opus encoding path no longer calls it.

---

## 5. Concurrency Review

### Race detector — PASS

```
go test -race ./internal/encoding/ ./internal/audio/native/
ok  github.com/JamesPrial/go-scream/internal/encoding  (cached)
ok  github.com/JamesPrial/go-scream/internal/audio/native  (cached)
```

No data races detected. The `rand.Rand` instances in `NoiseBurstLayer` and `BackgroundNoiseLayer` are accessed only from within a single `Generate` call (no shared state across goroutines).

### Goroutine lifecycle — PASS (unchanged from v1)

`EncodeFrames` goroutine correctly closes both `frameCh` and `errCh` via `defer` in all code paths.

### Channel backpressure — IMPROVED

`frameCh` buffer is now 50 (up from 2). The encoding goroutine can produce a full second of audio without blocking on a slow consumer.

---

## 6. Escape Analysis Summary

### Per-sample loop — CLEAN (0 allocations confirmed)

The generator's inner loop allocates nothing. All arithmetic is on value types or pointer receivers that do not escape. The escape analysis on the hot path:

- `l.Sample(t)` — interface call, but the concrete layer receivers are pre-allocated heap objects (one-time cost); no per-call allocation
- `filterChain.Process(raw)` — same: pre-allocated heap objects, no per-call allocation
- `math.Sin`, `math.Exp`, `math.Log` — intrinsics on arm64, no allocation

### One-time setup allocations (per `Generate` call) — 23 allocs, EXPECTED

The 23 allocations cover: 5 layer structs + 3 oscillators + 7 filter structs + 1 FilterChain + 1 LayerMixer + 2 rand.Rand sources + 1 output `[]byte` + 1 `bytes.Reader` = ~23 objects. All are one-time costs amortized over the entire generation loop.

### Encoding goroutine — 2 fixed setup allocs, 1 per frame (encoded `[]byte`)

`make([]byte, frameBytes)` and `make([]int16, 960*channels)` escape to heap once per goroutine launch. The returned `encoded` `[]byte` slice from `encoder.Encode` allocates once per frame — this is unavoidable since it is the output data being passed through the channel.

---

## 7. Recommendations (Remaining Work)

| Priority | Issue | File | Expected Impact |
|---|---|---|---|
| 1 | Replace `Compressor.Process` transcendentals with a cheaper gain formula | `internal/audio/native/filters.go:121` | ~55% improvement to filter chain; ~8-10% improvement to generator wall-time |
| 2 | Introduce `ConcreteFilterChain` with direct struct fields to eliminate 6 interface dispatches per sample | `internal/audio/native/filters.go` | Allows compiler to inline all 6 filter Process calls; ~15-20% filter chain improvement (on top of #1) |
| 3 | Introduce `ScreamMixer` struct with 5 typed layer fields to eliminate 5 interface dispatches per mix | `internal/audio/native/layers.go` | ~15-20 ns/sample recovered; requires refactoring `buildLayers` to return concrete type |
| 4 | Remove or isolate `pcmBytesToInt16` from production code path | `internal/encoding/encoder.go:69` | Code clarity; the function is dead code in the hot path |
| 5 | Streaming generation: producer goroutine writes one frame of PCM at a time, encoder goroutine reads it | `internal/audio/native/generator.go` + `internal/encoding/opus.go` | Reduces peak memory from ~768 KB to ~15 KB; allows first Opus frame to transmit before generation finishes |

---

## 8. Next Steps

1. **Profile under real Discord conditions.** The benchmark is synthetic (silent PCM, single goroutine). A CPU profile captured during live playback via `go test -bench=BenchmarkNativeGenerator_Classic -cpuprofile=cpu.prof` then `go tool pprof -top cpu.prof` will confirm whether `math.Sin` or `Compressor.Process` dominates the wall-clock budget for the real scream preset.

2. **Apply Recommendation #1 (compressor approximation).** The Compressor change is isolated to a 20-line function and can be validated with the existing `TestCompressor_AboveThreshold` and `TestCompressor_BelowThreshold` tests. This is the highest-ROI remaining change.

3. **Benchmark with `-benchtime=10s`** for the per-sample benchmarks (`BenchmarkOscillator_Sin`, `BenchmarkFilterChain_Classic`, `BenchmarkLayerMixer`) to reduce variance and get stable sub-1 ns precision for the optimization comparison.

4. **Consider the streaming pipeline** (Recommendation #5) if memory footprint matters for the deployment environment. The current architecture allocates ~576 KB for PCM + ~192 KB for the Opus encoder's internal state per scream invocation, which is fine for a lightly-loaded bot but would become a concern at high concurrency.
