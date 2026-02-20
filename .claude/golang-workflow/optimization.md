# Performance Review: 9-Stage Refactoring

**Date:** 2026-02-19
**Branch:** main
**Tool:** go test -bench=. -benchmem -benchtime=3s ./...
**Platform:** darwin/arm64 (Apple M1)

---

## Summary

The 9-stage refactoring introduced **no measurable performance regressions**. The hot path (native audio synthesis) is allocation-free per sample. All refactoring-introduced costs are confined to cold startup/init paths. One pre-existing heap allocation pattern in the filter chain and layer construction is noted but is not a regression from this refactoring.

Race detector passes cleanly across all packages. No goroutine leaks detected.

---

## Benchmark Results (benchtime=3s)

### internal/audio/native (hot path)

```
BenchmarkGenerator_Classic-8              194   18,690,547 ns/op   593,220 B/op   23 allocs/op
BenchmarkSweepJumpLayer_PrimaryScream-8   422,664,709   8.523 ns/op   0 B/op   0 allocs/op
BenchmarkLayerMixer-8                      49,589,366  70.49 ns/op   0 B/op   0 allocs/op
BenchmarkFilterChain_Classic-8             66,332,750  55.36 ns/op   0 B/op   0 allocs/op
BenchmarkOscillator_Sin-8                 435,720,793   8.268 ns/op   0 B/op   0 allocs/op
BenchmarkOscillator_Saw-8                 501,691,466   7.150 ns/op   0 B/op   0 allocs/op
```

### internal/audio/ffmpeg (argument building, cold path)

```
BenchmarkBuildArgs-8            713,080   4,936 ns/op   3,097 B/op   85 allocs/op
BenchmarkBuildAevalsrcExpr-8  1,000,000   3,314 ns/op   1,552 B/op   54 allocs/op
BenchmarkBuildFilterChain-8   2,538,291   1,375 ns/op     752 B/op   25 allocs/op
```

### internal/scream (orchestration layer)

```
Benchmark_Play_HappyPath-8      2,641,386   1,354 ns/op   4,678 B/op    9 allocs/op
Benchmark_Generate_HappyPath-8  6,633,343     550 ns/op   4,192 B/op    3 allocs/op
Benchmark_ListPresets-8       123,641,322    29.5 ns/op      96 B/op    1 allocs/op
```

### internal/discord

```
BenchmarkPlayer_Play_150Frames-8   382,131   9,474 ns/op   3,566 B/op   74 allocs/op
```

### internal/encoding

```
BenchmarkGopusFrameEncoder_1s_Stereo48k-8   1,471   2,448,351 ns/op   263,809 B/op   60 allocs/op
BenchmarkWAVEncoder_1s_Stereo48k-8         44,026      82,157 ns/op 1,104,280 B/op   24 allocs/op
BenchmarkOGGEncoder_150Frames-8            36,685      97,654 ns/op    95,818 B/op  175 allocs/op
```

---

## Refactoring-Specific Analysis

### Stage 5 — SweepJumpLayer consolidation (two types collapsed into one with a coprime field)

**Verdict: Allocation-neutral. Slight layout improvement.**

Before stage 5 there were two separate layer struct types for the primary-scream and high-shriek paths. After stage 5 they share `SweepJumpLayer` with an additional `coprime int64` field.

Escape analysis confirms both `NewPrimaryScreamLayer` and `NewHighShriekLayer` still allocate one `SweepJumpLayer` and one `Oscillator` on the heap (as before). The extra `int64` field adds 8 bytes to each struct but this is a one-time allocation per `Generate` call, not a per-sample cost. `SweepJumpLayer.Sample` is allocation-free at 8.5 ns/op with 0 B/op.

There is a minor benefit: the `coprime` field is read from `l.coprime` on every step-change rather than being a compile-time constant branch. The field is hot in L1 cache during the step-change path. No measurable regression.

### Stage 6 — math.Log10 in resolveParams (volume conversion, hot-path claim)

**Verdict: Cold path only. No impact on audio generation.**

The call site is `/Users/jamesprial/code/go-scream/internal/scream/service.go:160`:

```go
params.Filter.VolumeBoostDB += 20 * math.Log10(cfg.Volume)
```

This executes **once per Play/Generate call** in `resolveParams`, before any PCM synthesis begins. It does not appear inside the `Generate` loop in `internal/audio/native/generator.go`. Escape analysis shows `math.Log10` is inlined at the call site. The `Benchmark_Generate_HappyPath` benchmark (which uses a mock generator) shows 550 ns/op; the cost of a single `math.Log10` call on M1 is under 5 ns. This is negligible at real audio generation timescales (18–21 ms for the Classic preset).

### Stage 8 — app.NewGenerator / NewFileEncoder / NewDiscordDeps wrappers

**Verdict: Cold path only. Function call indirection has zero hot-path cost.**

`NewGenerator` and `NewFileEncoder` in `/Users/jamesprial/code/go-scream/internal/app/wire.go` are called once during service construction (startup). `NewFileEncoder` is fully inlined at its call sites (cost 27, budget 80). `NewGenerator` is just over budget (cost 89) so it is not inlined, but it is called once per binary invocation. The interface conversion (`audio.Generator`) that results from returning `native.NewGenerator()` causes a one-time heap escape of `&native.Generator{}`, which was already happening at the call site before stage 8 centralised the wiring. No new allocations per audio operation.

`NewDiscordDeps` (cost 301) is not inlined, but it wraps network I/O and is unambiguously cold.

### Stage 9 — runWithService callback / closure allocation

**Verdict: One closure per command invocation. No per-audio-frame cost.**

`runWithService` in `/Users/jamesprial/code/go-scream/cmd/scream/service.go` accepts a `func(context.Context, *scream.Service) error`. The closures passed from `runPlay` (line 56) and `runGenerate` (line 42) are analysed by the compiler:

```
cmd/scream/service.go:55:9:  func literal does not escape   (defer closer.Close closure)
cmd/scream/generate.go:42:29: func literal does not escape   (runGenerate callback)
cmd/scream/generate.go:47:9:  func literal does not escape   (defer f.Close closure)
cmd/scream/play.go:56:29:     func literal does not escape   (runPlay callback)
```

All four closures are stack-allocated. The one heap escape in `play.go:16:24` is the `RunE` field assignment (`runPlay` function literal stored into the Cobra command struct at `init` time), which is a one-time program initialisation cost unrelated to the audio path.

The `fn` parameter itself is passed through as a plain function value; the compiler has confirmed neither the `fn` parameter nor its captured variables escape to the heap at the call site. **Zero closure allocation overhead on the audio path.**

---

## Pre-Existing Allocation Patterns (Not Regressions)

These patterns exist in the codebase prior to the refactoring and are noted for completeness.

### Filter and layer setup: 23 allocations per Generate call

`BenchmarkGenerator_Classic` shows **23 allocs/op** for a full generation run. These are all one-time setup allocations before the sample loop:

- 3x `SweepJumpLayer` + 3x `Oscillator` (primary, harmonic, high-shriek)
- 1x `HarmonicSweepLayer` (has its own type)
- 1x `NoiseBurstLayer` + 1x `rand.Rand` + 1x `rand.rngSource` (from `rand.New(rand.NewSource(...))`)
- 1x `BackgroundNoiseLayer` + 1x `rand.Rand` + 1x `rand.rngSource`
- 1x `[]Layer` slice (5 elements, escaped from `buildLayers`)
- 1x `LayerMixer` (stays on stack when created in `Generate`, but the Layer interface slice causes the mixer to escape — see note below)
- 6x filter objects (`HighpassFilter`, `LowpassFilter`, `Bitcrusher`, `Compressor`, `VolumeBoost`, `Limiter`) + 1x `FilterChain` + variadic `[]Filter` arg
- 1x `[]byte` output buffer (size-dependent, ~576 KB for 3 s stereo)
- 1x `bytes.Reader` wrapper

The inner sample loop at `internal/audio/native/generator.go:48` is entirely allocation-free. The 23 pre-loop allocs are amortised across 144,000 samples (3 s at 48 kHz) = ~0.16 ns/sample overhead from setup.

### Interface dispatch in LayerMixer.Sample (not inlinable)

`(*LayerMixer).Sample` iterates over `[]Layer` (an interface slice) and calls `l.Sample(t)` via dynamic dispatch. The compiler reports cost 94 (budget 80), so it cannot be inlined. However, since `LayerMixer.Sample` itself has 0 B/op at 70 ns/op, the virtual dispatch overhead is already absorbed and is not actionable without a more invasive refactoring (e.g., static dispatch via a concrete 5-layer struct).

### FilterChain.Process interface dispatch

The 6-element `[]Filter` slice similarly uses interface dispatch. `BenchmarkFilterChain_Classic` shows 55 ns/op with 0 B/op. Given the total `Generate` time is ~19 ms for 144,000 calls (about 132 ns per sample across all processing), and the filter chain is ~55 ns of that, there is no urgent case for eliminating the interface here.

---

## Concurrency and Goroutine Safety

**Race detector: PASS across all packages.**

- The encoding goroutine in `GopusFrameEncoder.EncodeFrames` is correctly terminated: `frameCh` and `errCh` are closed via `defer` before the goroutine exits. No leak path.
- `Service.Play` drains `frameCh` via `player.Play` and then synchronously reads `<-errCh`. The encoder goroutine is guaranteed to have exited by the time `Play` returns.
- `Service.Play` (DryRun path) explicitly drains `frameCh` with `for range frameCh {}` before reading `errCh`. No goroutine leak.
- `sendSilence` in `discord/player.go` uses a 500 ms timeout context, preventing it from blocking indefinitely if the Discord channel is full on cancellation.
- The `runWithService` pattern in `cmd/scream/service.go` installs a `signal.NotifyContext` and defers `stop()`, ensuring signal handling goroutines are released.

---

## Critical Issues

None. No performance regressions, no goroutine leaks, no race conditions.

---

## Recommendations

These are optional improvements, not bugs introduced by the refactoring.

### 1. Merge rand.New(rand.NewSource(...)) into a single allocation (minor)

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go:127` and `:160`

`rand.New(rand.NewSource(seed))` allocates two objects: the `rand.Rand` wrapper and the `rngSource`. Go 1.20+ `rand.New` still allocates both. Since these are created once per `Generate` call, the impact is ~4 allocs out of 23. Not worth changing unless you need to push below 20 allocs.

### 2. Preallocate the layers slice with a fixed-size array (removes 1 alloc)

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go:100`

```go
// Current: escapes to heap
layers := []Layer{...}
```

The 5-element `[]Layer` slice literal escapes because it is passed to `NewLayerMixer(layers...)` which stores it in a heap-allocated `LayerMixer`. Since `LayerMixer` already escapes (its `layers` field holds interface values), this cannot be avoided without restructuring `LayerMixer` to hold a `[5]Layer` array. Only worthwhile if the 23-alloc count is a target.

### 3. BenchmarkBuildArgs allocation count (ffmpeg cold path)

**File:** `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go`

85 allocs/op and ~5 µs for `BuildArgs` is acceptable since it runs once before spawning the ffmpeg subprocess (which takes hundreds of milliseconds). No action required.

---

## Next Steps

- No immediate profiling action required. All hot-path work (synthesis loop, filter chain) is allocation-free and running at expected ns/op ranges for M1.
- If `BenchmarkGenerator_Classic` ever needs to improve (e.g., to support very short latency re-generation), the main lever is reducing the 23 setup allocations by using `sync.Pool` for the layer and filter objects, or switching `LayerMixer` to a concrete `[5]Layer` array field. Current throughput of ~53 generated screams per second is more than sufficient for the use case.
- Re-run benchmarks after any future change to `buildLayers`, `NewFilterChainFromParams`, or the encoding pipeline.
