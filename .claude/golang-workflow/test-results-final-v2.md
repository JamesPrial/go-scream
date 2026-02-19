# Test Execution Report — go-scream Final v2

**Date:** 2026-02-18  
**Platform:** darwin/arm64 (Apple M1)  
**Go Version:** 1.24  
**Module:** github.com/JamesPrial/go-scream

---

## Summary

- **Verdict:** TESTS_PASS
- **Tests Run:** 209 passed, 0 failed (18 skipped — ffmpeg not installed on host)
- **Coverage (total):** 71.4% across all statements
- **Race Conditions:** None detected
- **Vet Warnings:** None (go vet clean)
- **Linter (golangci-lint):** 16 non-critical style warnings (14 errcheck, 2 staticcheck) — no blocking errors

---

## Test Results (`go test -v ./...`)

### Package Results

| Package | Status | Notes |
|---|---|---|
| cmd/scream | no test files | CLI wiring — covered via integration |
| cmd/skill | PASS | 12 tests |
| internal/audio | PASS | 13 tests |
| internal/audio/ffmpeg | PASS (18 SKIP) | 4 pass, 18 skipped (ffmpeg binary absent) |
| internal/audio/native | PASS | 37 tests |
| internal/config | PASS | 44 tests |
| internal/discord | PASS | 22 tests |
| internal/encoding | PASS | 52 tests |
| internal/scream | PASS | 29 tests |
| pkg/version | PASS | 6 tests |

All 209 runnable tests passed. The 18 skips in `internal/audio/ffmpeg` are expected — each test guards itself with `t.Skip("ffmpeg not available")` when the binary is absent from PATH. This is correct behavior.

### Skipped Tests (expected — ffmpeg not on host)
- TestNewFFmpegGenerator_Success
- TestFFmpegGenerator_CorrectByteCount
- TestFFmpegGenerator_NonSilent
- TestFFmpegGenerator_AllPresets
- TestFFmpegGenerator_AllNamedPresets
- TestFFmpegGenerator_InvalidDuration
- TestFFmpegGenerator_NegativeDuration
- TestFFmpegGenerator_InvalidSampleRate
- TestFFmpegGenerator_NegativeSampleRate
- TestFFmpegGenerator_InvalidChannels
- TestFFmpegGenerator_ZeroChannels
- TestFFmpegGenerator_InvalidAmplitude
- TestFFmpegGenerator_InvalidCrusherBits
- TestFFmpegGenerator_InvalidLimiterLevel
- TestFFmpegGenerator_EvenByteCount
- TestFFmpegGenerator_StereoAligned
- TestFFmpegGenerator_MonoOutput
- TestFFmpegGenerator_DeterministicOutput

---

## Race Detection (`go test -race ./...`)

```
ok  github.com/JamesPrial/go-scream/cmd/skill
ok  github.com/JamesPrial/go-scream/internal/audio
ok  github.com/JamesPrial/go-scream/internal/audio/ffmpeg
ok  github.com/JamesPrial/go-scream/internal/audio/native
ok  github.com/JamesPrial/go-scream/internal/config
ok  github.com/JamesPrial/go-scream/internal/discord
ok  github.com/JamesPrial/go-scream/internal/encoding
ok  github.com/JamesPrial/go-scream/internal/scream
ok  github.com/JamesPrial/go-scream/pkg/version
```

**No race conditions detected.**

Note: Three `ld: warning: malformed LC_DYSYMTAB` linker warnings appeared for cmd/skill, internal/encoding, and internal/scream test binaries. These are macOS system linker warnings caused by the gopus C library's object format; they are non-fatal, do not affect test correctness, and are not Go race conditions.

---

## Static Analysis (`go vet ./...`)

```
(no output)
```

**No vet warnings.** Exit status 0.

---

## Coverage Details (`go test -cover ./...`)

| Package | Coverage |
|---|---|
| cmd/scream | 0.0% (no test files — CLI main package) |
| cmd/skill | 22.4% |
| internal/audio | 87.5% |
| internal/audio/ffmpeg | 90.6% |
| internal/audio/native | **100.0%** |
| internal/config | 97.6% |
| internal/discord | 64.5% |
| internal/encoding | 86.4% |
| internal/scream | 95.0% |
| pkg/version | **100.0%** |
| **Total (all statements)** | **71.4%** |

### Coverage Notes

- `cmd/scream` (0%): This is the CLI entry-point package containing `main()`, `init()` cobra wiring, and `runPlay`/`runGenerate` functions. Coverage of CLI main packages via unit tests is generally impractical; the business logic it delegates to is covered by `internal/scream` (95%).
- `internal/discord` (64.5%): The `session.go` adapter wraps live discordgo types; its concrete methods (ChannelVoiceJoin, GuildVoiceStates, Speaking, OpusSendChannel, Disconnect, IsReady) are 0% covered because they require a live Discord connection. The `player.go` core logic is 82.4% covered via mocks. `sendSilence` is 66.7% covered.
- `cmd/skill` (22.4%): The `main()` function itself is untested (0%), but `parseOpenClawConfig` and `resolveToken` are both 100% covered.
- All performance-critical packages (`internal/audio/native`, `internal/encoding`, `internal/scream`) meet or exceed the 70% threshold.

**Overall 71.4% total meets the >70% threshold.**

---

## Benchmark Results (`go test -bench=. -benchmem`)

### internal/encoding (Apple M1, arm64)

| Benchmark | Iterations | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| BenchmarkPcmBytesToInt16_1s_Stereo48k | 13,653 | 87,413 | 196,610 | 1 |
| BenchmarkOGGEncoder_150Frames | 12,194 | 97,726 | 95,818 | 175 |
| BenchmarkGopusFrameEncoder_1s_Stereo48k | 487 | 2,462,498 | 263,808 | 60 |
| BenchmarkWAVEncoder_1s_Stereo48k | 14,676 | 81,401 | 1,104,280 | 24 |
| BenchmarkWAVEncoder_3s_Stereo48k | 4,252 | 301,958 | 3,824,040 | 29 |

**Key observations:**
- PCM conversion: 87 µs/op for 1s stereo 48kHz — very fast (single allocation).
- OGG encode (150 frames): 98 µs/op — well within real-time Discord budget.
- Opus encode (1s): 2.5 ms/op — the Opus codec is the bottleneck, expected.
- WAV encode (1s): 81 µs/op — trivially fast for file output.

### internal/audio/native (Apple M1, arm64)

| Benchmark | Iterations | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| BenchmarkFilterChain_Classic | 22,008,235 | 55.02 | 0 | 0 |
| BenchmarkNativeGenerator_Classic | 55 | 20,788,667 | 593,221 | 23 |
| BenchmarkPrimaryScreamLayer | 135,892,035 | 8.902 | 0 | 0 |
| BenchmarkLayerMixer | 16,965,146 | 70.03 | 0 | 0 |
| BenchmarkOscillator_Sin | 147,938,164 | 8.220 | 0 | 0 |
| BenchmarkOscillator_Saw | 176,296,592 | 6.804 | 0 | 0 |

**Key observations:**
- Oscillator (sin/saw): ~8 ns/op with zero allocations — optimal DSP performance.
- FilterChain: 55 ns/op, zero allocations — hot-path is allocation-free.
- LayerMixer: 70 ns/op, zero allocations — excellent.
- NativeGenerator (full 3s Classic): 20.8 ms/op, 593 KB, 23 allocs — well within a 3-second budget. A 3s scream generates in ~21ms (0.7% of real-time).

---

## Linter Output (`golangci-lint run`)

golangci-lint is installed. 16 non-critical style issues found. **None are blocking.**

### errcheck (14 issues — non-critical)

These are unchecked return values from functions where errors are either:
- Cosmetic/informational (fmt.Fprintf, fmt.Fprintln to stdout)
- Best-effort cleanup in defer statements (Close, Disconnect)

| File | Line | Issue |
|---|---|---|
| cmd/scream/generate.go:24 | `generateCmd.MarkFlagRequired("output")` unchecked |
| cmd/scream/generate.go:40 | `fmt.Fprintf(...)` unchecked |
| cmd/scream/generate.go:51 | `defer closer.Close()` unchecked |
| cmd/scream/generate.go:57 | `defer f.Close()` unchecked |
| cmd/scream/play.go:50 | `fmt.Fprintf(...)` unchecked |
| cmd/scream/play.go:52 | `fmt.Fprintf(...)` unchecked |
| cmd/scream/play.go:54 | `fmt.Fprintln(...)` unchecked |
| cmd/scream/play.go:66 | `defer closer.Close()` unchecked |
| cmd/scream/presets.go:22 | `fmt.Fprintln(...)` unchecked |
| cmd/skill/main.go:137 | `defer session.Close()` unchecked |
| internal/audio/native/generator_test.go:251 | `binary.Read(...)` unchecked (test helper) |
| internal/audio/native/generator_test.go:278 | `io.Copy(...)` unchecked (test helper) |
| internal/discord/player.go:61 | `defer vc.Disconnect()` unchecked |
| internal/encoding/ogg.go:46 | `defer oggWriter.Close()` unchecked |

### staticcheck (2 issues — non-critical)

| File | Line | Rule | Issue |
|---|---|---|---|
| internal/encoding/opus_test.go:50 | S1000 | Use simple channel send/receive instead of single-case select |
| internal/encoding/opus_test.go:309 | S1000 | Use simple channel send/receive instead of single-case select |

These are in test files only. The `select` with single case is functionally correct; it could be simplified to a direct `<-ch` send/receive.

**Recommended future cleanup** (non-blocking):
1. Wrap `defer closer.Close()` / `defer vc.Disconnect()` / `defer oggWriter.Close()` with `//nolint:errcheck` or handle via named return pattern
2. Simplify single-case selects in `internal/encoding/opus_test.go` to direct channel operations

---

## Final Verdict

**TESTS_PASS**

All functional checks passed:
- 209 tests pass, 0 failures
- 18 tests appropriately skipped (ffmpeg absent on host — by design)
- 0 race conditions
- 0 go vet warnings
- 71.4% total coverage (exceeds 70% threshold)
- 16 linter style warnings (non-critical errcheck/staticcheck — no blocking errors)
- Benchmarks confirm allocation-free hot paths in DSP layer and sub-millisecond encode times for OGG/WAV output

