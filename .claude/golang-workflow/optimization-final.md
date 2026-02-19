# Go-Scream Performance Analysis — Final Report

**Date:** 2026-02-18  
**Platform:** darwin/arm64 (Apple M1)  
**Go version:** 1.25.7  
**Working directory:** `/Users/jamesprial/code/go-scream`

---

## 1. Summary

The native audio generator is well-structured and the hot path (sample loop) is impressively lean at 0 B/op on per-sample calls. All per-sample math functions inline correctly and no allocations occur inside the generation loop itself. The dominant cost is a single large upfront allocation of the output buffer (~576 KB for a 3-second stereo scream), which is expected and unavoidable given the design.

The encoding pipeline has several recurring per-frame allocations that can be eliminated. The `pcmBytesToInt16` helper allocates a fresh `[]int16` slice on every call — currently called once per Opus frame (up to 150 times per scream). The OGG encoder allocates an `*rtp.Packet` per frame but the escape analysis shows this stays on the stack, which is good. Channel buffer sizes are minimal (2 for frames, 1 for errors) and appropriate for the workload. No goroutine leaks or data races were detected.

---

## 2. Benchmark Results

### Native audio generator (`internal/audio/native`)

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| `BenchmarkNativeGenerator_Classic` | 22,450,000 | 593,154 | 23 |
| `BenchmarkLayerMixer` (per sample) | 71.46 | 0 | 0 |
| `BenchmarkPrimaryScreamLayer` (per sample) | 12.58 | 0 | 0 |
| `BenchmarkFilterChain_Classic` (per sample) | 66.03 | 0 | 0 |
| `BenchmarkOscillator_Sin` (per call) | 10.15 | 0 | 0 |
| `BenchmarkOscillator_Saw` (per call) | 8.557 | 0 | 0 |

**Throughput:** ~22.5 ms to generate 3 seconds of 48 kHz stereo audio = approximately **133x real-time** on Apple M1. This is excellent for a bot use case.

### Encoding (`internal/encoding`)

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| `BenchmarkGopusFrameEncoder_1s_Stereo48k` | 2,465,829 | 463,282 | 109 |
| `BenchmarkPcmBytesToInt16_1s_Stereo48k` | 88,071 | 196,610 | 1 |
| `BenchmarkOGGEncoder_150Frames` | 97,901 | 95,818 | 175 |
| `BenchmarkWAVEncoder_3s_Stereo48k` | 273,958 | 3,824,039 | 29 |

### Discord player (`internal/discord`)

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| `BenchmarkDiscordPlayer_Play_150Frames` | 9,095 | 3,744 | 77 |

### FFmpeg command builder (`internal/audio/ffmpeg`)

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| `BenchmarkBuildArgs` | 4,809 | 3,097 | 85 |
| `BenchmarkBuildAevalsrcExpr` | 3,081 | 1,552 | 54 |

---

## 3. Critical Issues

### Issue 1 — `pcmBytesToInt16` allocates a new `[]int16` slice every call

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go:74`  
**Severity:** Medium  

The function allocates a new `[]int16` slice on every invocation and is called twice per Opus frame (once for partial frames, once for full frames) in `opus.go`:

```go
// encoder.go:69-79 — current implementation
func pcmBytesToInt16(pcm []byte) []int16 {
    if len(pcm) < 2 {
        return nil
    }
    n := len(pcm) / 2
    out := make([]int16, n)   // <-- heap allocation every call
    for i := 0; i < n; i++ {
        out[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
    }
    return out
}
```

For a 3-second scream at 48 kHz stereo, the Opus encoder makes 150 frames × 1 allocation = 150 × 1,920 int16 values = 150 allocations of 3,840 bytes = **576 KB of redundant heap churn per playback**. The benchmark confirms: `463,282 B/op, 109 allocs/op` for 1 second (50 frames).

**Recommendation:** Pre-allocate one `[]int16` buffer in the goroutine and reuse it across all frames. The buffer size is fixed at `OpusFrameSamples * channels` for the duration of the encoding session:

```go
// In EncodeFrames goroutine, after encoder creation:
samples := make([]int16, OpusFrameSamples*channels)

// Then replace pcmBytesToInt16(pcmBuf) calls with an in-place decode:
for i := range samples {
    samples[i] = int16(binary.LittleEndian.Uint16(pcmBuf[i*2:]))
}
// Pass samples[:OpusFrameSamples*channels] to encoder.Encode(...)
```

This eliminates all per-frame `[]int16` allocations from the hot encoding loop.

---

### Issue 2 — `bytes.Buffer` as intermediate PCM store in the generator; double-copy path

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go:46-77`  
**Severity:** Low-Medium  

The generator uses a `bytes.Buffer` with a pre-sized backing slice, then extracts it via `w.Bytes()` and wraps it in a `bytes.Reader`:

```go
// generator.go:46-77 — current pattern
buf := make([]byte, 0, totalSamples*channels*2)
w := bytes.NewBuffer(buf)

for i := 0; i < totalSamples; i++ {
    // ...
    w.WriteByte(sampleBuf[0])
    w.WriteByte(sampleBuf[1])
}

return bytes.NewReader(w.Bytes()), nil  // wraps into Reader over the Buffer's slice
```

Two concerns:

1. `bytes.Buffer.WriteByte` is called **twice per channel per sample** = 4 calls per stereo sample = ~288,000 `WriteByte` calls for 3 s stereo. Each call does bounds-checking and increments an internal write cursor. A single `binary.LittleEndian.PutUint16` into a pre-allocated `[]byte` with manual index tracking avoids the `Buffer` overhead entirely.

2. The `sampleBuf := make([]byte, 2)` scratch buffer at line 50 is allocated once (good), but it could be eliminated entirely by writing directly into the output slice at a computed offset.

**Recommendation:** Allocate the output `[]byte` slice directly and write into it by index, then wrap in `bytes.NewReader`. This keeps the same single-allocation budget while eliminating all `WriteByte` dispatch overhead:

```go
out := make([]byte, totalSamples*channels*2)
pos := 0
for i := 0; i < totalSamples; i++ {
    t := float64(i) / float64(sampleRate)
    raw := mixer.Sample(t)
    filtered := filterChain.Process(raw)
    scaled := filtered * 32767.0
    clamped := math.Max(-32768, math.Min(32767, scaled))
    s16 := int16(math.Round(clamped))
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

Expected improvement: moderate (~5-10%) reduction in generator wall time; more meaningful reduction in instruction count due to eliminating method dispatch overhead in the inner loop.

---

### Issue 3 — `math.Floor` called on every sample in three layers

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go:42,75,107,136`  
**Severity:** Low  

`PrimaryScreamLayer.Sample`, `HarmonicSweepLayer.Sample`, and `HighShriekLayer.Sample` all call `math.Floor(t * l.jump)` to determine the current frequency step. At 48,000 samples/s for 3 s = 144,000 calls per layer, that is 432,000 `math.Floor` calls for the three tonal layers combined, plus another in `NoiseBurstLayer`.

For layers where `jump` is typically 5–25 Hz (meaning a step change every 40–200 ms), the step value changes very infrequently relative to the sample rate. Caching the current step and only recomputing `seededRandom` when the step changes would reduce the random-hash calls significantly.

**Recommendation:** Add a `curStep int64` and `curFreq float64` field to each stateful layer. On each `Sample` call, compute the step cheaply using integer arithmetic and only call `seededRandom` on a step change:

```go
func (l *PrimaryScreamLayer) Sample(t float64) float64 {
    step := int64(t * l.jump)   // avoids math.Floor — truncation toward zero is correct here
    if step != l.curStep {
        l.curStep = step
        l.curFreq = l.base + l.freqRange*seededRandom(l.seed, step, 137)
    }
    envelope := l.amp * (1 + l.rise*t)
    return envelope * l.osc.Sin(l.curFreq)
}
```

Note: `int64(t * l.jump)` truncates toward zero for positive `t`, which is equivalent to `math.Floor` for positive values. This eliminates `math.Floor` from the per-sample path.

---

### Issue 4 — `Compressor.Process` calls `math.Pow` on every sample above threshold

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/filters.go:135`  
**Severity:** Low  

```go
// filters.go:130-136
gain := 1.0
if f.envelope > f.threshold {
    excess := f.envelope / f.threshold
    gain = math.Pow(excess, 1.0/f.ratio-1.0)  // math.Pow per sample when compressing
}
```

`math.Pow` is implemented as `exp(y * log(x))` and is significantly more expensive than the multiplications in the filter chain. For a loud scream signal that spends most of its time above the compressor threshold, this executes at 48,000 Hz.

**Recommendation:** Pre-compute the ratio exponent `ratioExp := 1.0/ratio - 1.0` as a struct field (already partially done via `ratio` field). The main win would be to replace `math.Pow` with `math.Exp(ratioExp * math.Log(excess))` — but the real optimization is to consider whether the compressor algorithm can use a lookup table or a cheaper soft-knee approximation. For the current use case (perceptual compression of a scream for Discord), a simpler gain formula would be adequate:

```go
// Cheaper alternative: linear gain reduction above threshold
// gain = threshold / envelope * (ratio-1)/ratio + 1/ratio  (VCA-style)
```

This removes the transcendental function from the per-sample path entirely.

---

### Issue 5 — `EncodeFrames` frame channel buffer is only 2

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/opus.go:43`  
**Severity:** Low  

```go
frameCh := make(chan []byte, 2)
```

The encoding goroutine and the consumer (either `OGGEncoder.Encode` or `DiscordPlayer.Play`) are connected through this channel. With a buffer of 2, the encoding goroutine will block on `frameCh <- encoded` as soon as the consumer is slower than two frames ahead. For the `Play` path, the consumer is Discord's `OpusSendChannel` which is itself bounded by network I/O, creating back-pressure through the chain.

For the `DryRun` path and `Generate` (file encoding) path this is fine since the consumer is fast. For live Discord playback at 50 frames/second, a buffer of 2 means the encoder goroutine is nearly always blocked waiting for the player, adding latency and goroutine wake-up overhead.

**Recommendation:** Increase the frame channel buffer to match one Opus window (50 frames = 1 second of audio) or at minimum 10 frames. This allows the encoder to run ahead of the player and reduces scheduler pressure:

```go
frameCh := make(chan []byte, 50)  // 1 second of lookahead at 48kHz/960
```

---

### Issue 6 — WAV encoder double-allocates: `io.ReadAll` then `dst.Write`

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/wav.go:46`  
**Severity:** Low (WAV path only, not the Discord hot path)  

```go
pcmData, err := io.ReadAll(src)  // allocates entire PCM buffer: ~576 KB for 3s stereo
// ...
dst.Write(pcmData)               // copies again into dst
```

For the WAV encoder specifically, the PCM data is already held in a `bytes.Reader` (from the native generator). Reading it all into a new `[]byte` via `io.ReadAll` creates a second copy of the full PCM buffer. The benchmark shows `3,824,039 B/op` for a 3-second file, which is the expected PCM size plus the WAV header. The WAV path is not the Discord hot path, so the impact is limited to `generate` subcommand usage.

**Recommendation:** If the source is known to be a `*bytes.Reader`, use `src.(*bytes.Reader).Bytes()` with a type assertion; otherwise stream-copy via `io.Copy` to avoid the intermediate buffer. Alternatively, accept that WAV write is a one-shot operation and the current approach is fine for its non-realtime use.

---

### Issue 7 — `defer cancel()` in a loop inside `Play`

**File:** `/Users/jamesprial/code/go-scream/internal/discord/player.go:75-79, 87-91`  
**Severity:** Low (functional correctness concern, not a performance leak)  

```go
// player.go:75-79 — inside the for loop
case <-ctx.Done():
    silenceCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()   // <-- deferred inside a loop body
    sendSilence(silenceCtx, opusSend)
    vc.Speaking(false)
    return ctx.Err()
```

In this specific case, `defer cancel()` is immediately followed by `return`, so the deferred cancel fires correctly before the function returns. The loop body exits via `return` each time, so there is no accumulation of deferred calls. However, using `defer` inside a loop is a recognized Go anti-pattern that can confuse static analysis tools and future maintainers. Because `return` is always called immediately after `defer cancel()`, this is safe but misleading.

**Recommendation:** Replace `defer cancel()` with an explicit `cancel()` call after `sendSilence` completes, since the context is purely used by `sendSilence`:

```go
case <-ctx.Done():
    silenceCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    sendSilence(silenceCtx, opusSend)
    cancel()  // explicit, not deferred — safe since we return immediately after
    vc.Speaking(false)
    return ctx.Err()
```

---

## 4. Concurrency and Goroutine Analysis

### Goroutine lifecycle — PASS

`EncodeFrames` goroutine: Both `frameCh` and `errCh` are closed via `defer` in all code paths (normal completion, partial frame, read error, encode error). Consumers that range over `frameCh` will unblock and the goroutine will not leak.

`OGGEncoder.Encode`: Correctly drains `frameCh` even when `oggWriter` creation fails, preventing the encoder goroutine from blocking on `frameCh <- encoded`. Error priority logic (OGG write errors take priority over Opus errors) is sound.

`Service.Play`: Correctly waits for `<-errCh` after `player.Play` returns, ensuring the encoding goroutine has completed before the function returns. This prevents goroutine leaks on the happy path.

### Race detector — PASS

All packages passed `go test -race ./...` with no data races detected.

### Channel closing protocol — PASS

Channels are only closed by the sending side (`EncodeFrames` goroutine closes both `frameCh` and `errCh`). No double-close paths found. Receivers use range-based drain or select with closed-channel detection correctly.

### `defer cancel()` timing — see Issue 7 above

---

## 5. Escape Analysis Findings

### Per-sample loop — CLEAN (0 allocations)

The sample generation loop in `generator.go:52-75` allocates nothing per iteration. All filter and oscillator state is held in pre-allocated structs. The escape analysis confirms the hot path is allocation-free.

### Construction-time heap allocations — Expected

All layer and filter structs escape to the heap because they are stored as interface values (`Layer`, `Filter`) in slices. This is one-time allocation per `Generate` call, not per-sample. With 23 total allocs for the entire `Generate` call, the setup cost is negligible.

### `pcmBytesToInt16` — AVOIDABLE (see Issue 1)

```
internal/encoding/encoder.go:74:13: make([]int16, n) escapes to heap
```

This allocation occurs twice per Opus encode loop iteration (lines 94 and 109 in `opus.go`). It is the most impactful avoidable allocation in the encoding hot path.

### `binary.LittleEndian` in `wav.go` — Minor interface escape

```
internal/encoding/wav.go:69:36: binary.LittleEndian escapes to heap
```

`binary.Write(dst, binary.LittleEndian, &hdr)` causes `binary.LittleEndian` to escape because `binary.Write` accepts an `interface{}`. This is a 44-byte one-time write per WAV file and is not worth addressing.

---

## 6. Math Efficiency Review

### Oscillator phase management — GOOD

The phase accumulator pattern in `oscillator.go` is correct and efficient. `math.Floor(o.phase)` for phase wrapping is called once per `Sin` or `Saw` call. For phase values that are always in `[0, 2)` (which they will be since `freq/sampleRate << 1` for all audio frequencies), `math.Floor` can be replaced with a conditional subtraction:

```go
// Current (oscillator.go:20-22)
o.phase += freq / o.sampleRate
o.phase -= math.Floor(o.phase)

// Equivalent for phase in [0, 2):
o.phase += freq / o.sampleRate
if o.phase >= 1.0 {
    o.phase -= 1.0
}
```

This avoids the `math.Floor` transcendental call at 48,000 Hz per oscillator. Given there are 3 tonal oscillators active simultaneously, this saves ~144,000 `math.Floor` calls per second of audio.

### `splitmix64` hash — EXCELLENT

The stateless `splitmix64` implementation in `layers.go:188-194` is optimal for its purpose. It uses only integer bitwise operations and multiplications, with no heap allocation or branching. The `seededRandom` wrapper (`layers.go:199-202`) is correctly inlined.

### `math.Round` in generator — ACCEPTABLE

`math.Round(clamped)` at `generator.go:65` is called once per sample. Given the value is already clamped to `[-32767, 32767]`, `math.Round` is implemented as `math.Floor(x + 0.5)` internally. This is called at the same rate as `math.Sin` (once per sample) and is not a bottleneck.

### Bitcrusher `math.Floor` — ACCEPTABLE

`math.Floor(sample * f.levels) / f.levels` in `Bitcrusher.Process` is called once per sample but only in the filter chain, which runs after mixing. Not a bottleneck.

---

## 7. Recommendations Priority Order

| Priority | Issue | File | Expected Impact |
|---|---|---|---|
| 1 | Pre-allocate `[]int16` buffer in `EncodeFrames`, reuse per frame | `internal/encoding/opus.go` | Eliminate 100+ allocs/encode; reduce GC pressure |
| 2 | Replace `bytes.Buffer` + `WriteByte` with direct index writes in generator | `internal/audio/native/generator.go` | ~5-10% generator speedup; cleaner code |
| 3 | Cache frequency step in `PrimaryScreamLayer`, `HarmonicSweepLayer`, `HighShriekLayer` | `internal/audio/native/layers.go` | Eliminate `math.Floor` + hash compute per sample for most samples |
| 4 | Replace `math.Floor` phase wrap with conditional subtraction in `Oscillator` | `internal/audio/native/oscillator.go` | Eliminate `math.Floor` from 3 oscillators × 48,000 Hz |
| 5 | Increase `frameCh` buffer from 2 to 50 in `EncodeFrames` | `internal/encoding/opus.go` | Reduce encoder-player scheduling contention |
| 6 | Replace `math.Pow` in `Compressor.Process` with cheaper gain formula | `internal/audio/native/filters.go` | Eliminate transcendental function at 48,000 Hz |
| 7 | Replace `defer cancel()` with explicit `cancel()` in `Play` loop | `internal/discord/player.go` | Code clarity; no functional change |

---

## 8. Next Steps

1. **Apply Issue 1 fix** (pre-allocate `[]int16`) and re-run `BenchmarkGopusFrameEncoder_1s_Stereo48k` to verify alloc count drops from 109 to ~9 (one alloc per goroutine setup, not per frame).

2. **Apply Issue 2 fix** (direct-index generator writes) and re-run `BenchmarkNativeGenerator_Classic` to quantify wall-time improvement.

3. **Profile under real Discord conditions** using `go tool pprof` with a CPU profile captured during a live playback session to validate that the per-sample math (sin, floor, pow) dominates as expected, or whether the GC pause from the 576 KB buffer is the actual wall-time driver.

4. **Consider streaming generation**: The current pipeline generates the full PCM buffer before encoding begins. A streaming design where the generator produces samples into a fixed-size ring buffer, and the encoder reads from that buffer concurrently, would reduce peak memory from ~576 KB + ~192 KB (int16) to ~15 KB (one Opus frame worth of PCM). This would be a larger architectural change but would allow the bot to begin transmitting audio faster and reduce memory pressure.
