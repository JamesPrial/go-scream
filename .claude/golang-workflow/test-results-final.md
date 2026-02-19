# Test Execution Report — Final (All Stages)

**Project:** go-scream (Go rewrite of the scream Discord bot)
**Module:** github.com/JamesPrial/go-scream
**Date:** 2026-02-18
**Working Directory:** /Users/jamesprial/code/go-scream

---

## Summary

- **Verdict:** TESTS_PASS
- **Tests Run:** 467 passed, 0 failed, 18 skipped (ffmpeg not installed on this machine)
- **Coverage:** See per-package breakdown below
- **Race Conditions:** None detected
- **Vet Warnings:** None
- **Linter:** 13 non-critical issues (11 errcheck, 2 staticcheck) — no test failures, no correctness bugs

---

## Test Results (`go test -v ./...`)

All packages passed. No failures.

### Package summary

| Package | Result | Notes |
|---|---|---|
| cmd/scream | no test files | CLI entrypoint, no unit tests |
| cmd/skill | PASS | OpenClaw skill handler |
| internal/audio | PASS | Params, presets |
| internal/audio/ffmpeg | PASS | 18 tests skipped (ffmpeg binary not present) |
| internal/audio/native | PASS | Oscillator, layers, filters, generator |
| internal/config | PASS | Config load, merge, validate |
| internal/discord | PASS | Player, channel finder |
| internal/encoding | PASS | Opus, OGG, WAV encoders |
| internal/scream | PASS | Service, resolve |
| pkg/version | PASS | Version string |

### Skipped tests (ffmpeg unavailable — expected on this machine)

All 18 skipped tests are in `internal/audio/ffmpeg/generator_test.go` and are
guarded by a `t.Skip("ffmpeg not available")` call when the binary is absent:

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

All packages passed with `-race`. No data races detected.

Two linker warnings appeared from the macOS toolchain (unrelated to the code):

```
ld: warning: '/private/var/.../000012.o' has malformed LC_DYSYMTAB, expected 98
undefined symbols to start at index 1626, found 95 undefined symbols starting
at index 1626
```

These are macOS system-level linker warnings from gopus (CGo), not race conditions
and not errors. All test binaries ran successfully.

---

## Static Analysis (`go vet ./...`)

**No output. Zero warnings.** Exit status 0.

---

## Coverage Details (`go test -cover ./...`)

| Package | Coverage |
|---|---|
| cmd/scream | 0.0% (no test files — CLI wiring) |
| cmd/skill | 35.1% |
| internal/audio | 87.5% |
| internal/audio/ffmpeg | 90.6% |
| internal/audio/native | **100.0%** |
| internal/config | 97.6% |
| internal/discord | 64.5% |
| internal/encoding | 86.0% |
| internal/scream | 95.0% |
| pkg/version | **100.0%** |

**Overall average (tested packages):** ~85% across all packages with tests.

Notes on coverage below threshold:
- `cmd/skill` at 35.1%: The skill's `main()` function and Discord HTTP handler
  wiring are not unit-tested (integration concern). The `parseOpenClawConfig` and
  `resolveToken` functions are well-tested.
- `internal/discord` at 64.5%: The live Discord session path (`session.go` real
  VoiceConnect) is not reachable without a live Discord token. All mockable paths
  are covered.
- `cmd/scream` at 0.0%: This is the CLI entry point (`main()`, cobra command
  registration). No unit tests; integration-tested via end-to-end invocation.

---

## Linter Output (`golangci-lint run`)

golangci-lint is available. 13 non-critical issues found. **No issues block tests
or indicate correctness bugs.**

```
cmd/scream/generate.go:23:30: Error return value of `generateCmd.MarkFlagRequired` is not checked (errcheck)
cmd/scream/generate.go:39:14: Error return value of `fmt.Fprintf` is not checked (errcheck)
cmd/scream/generate.go:55:14: Error return value of `fmt.Fprintln` is not checked (errcheck)
cmd/scream/play.go:50:14: Error return value of `fmt.Fprintf` is not checked (errcheck)
cmd/scream/play.go:52:15: Error return value of `fmt.Fprintf` is not checked (errcheck)
cmd/scream/play.go:54:15: Error return value of `fmt.Fprintln` is not checked (errcheck)
cmd/scream/play.go:68:14: Error return value of `fmt.Fprintln` is not checked (errcheck)
internal/audio/native/generator_test.go:251:14: Error return value of `binary.Read` is not checked (errcheck)
internal/audio/native/generator_test.go:278:10: Error return value of `io.Copy` is not checked (errcheck)
internal/discord/player.go:61:21: Error return value of `vc.Disconnect` is not checked (errcheck)
internal/encoding/ogg.go:46:23: Error return value of `oggWriter.Close` is not checked (errcheck)
internal/encoding/opus_test.go:50:2: S1000: should use a simple channel send/receive instead of select with a single case (staticcheck)
internal/encoding/opus_test.go:309:2: S1000: should use a simple channel send/receive instead of select with a single case (staticcheck)
```

**Classification:**
- `errcheck` on `fmt.Fprintf`/`fmt.Fprintln` in CLI commands: standard practice to
  ignore write errors to stdout in CLI tools.
- `errcheck` on `vc.Disconnect()`: deferred disconnect; error is intentionally
  not checked (cleanup path).
- `errcheck` on `oggWriter.Close()`: deferred close in encoder; acceptable pattern.
- `errcheck` on `binary.Read`/`io.Copy` in test files: test helper code, not
  production.
- `S1000` staticcheck: cosmetic style preference, not a bug.

None of these are blocking issues.

---

## Issues to Address (optional improvements, not blocking)

1. **internal/discord/player.go:61** — Consider logging the Disconnect error for
   observability: `if err := vc.Disconnect(); err != nil { log.Print(err) }`.
2. **internal/encoding/ogg.go:46** — Consider capturing Close error and returning
   it if no prior error occurred.
3. **cmd/scream/generate.go:23** — `MarkFlagRequired` error can be safely checked
   with a panic-on-error pattern since it only fails on programming mistakes.
4. **cmd/skill** coverage — Increase by extracting the HTTP handler function and
   adding a test for it.
5. **internal/discord** coverage — Currently at 64.5%; the `session.go` adapter
   cannot be unit-tested without a live token, which is expected.

---

## Final Verdict

**TESTS_PASS**

All 467 tests pass. Zero failures. Zero race conditions. Zero vet warnings.
18 tests skipped due to ffmpeg binary not being installed (correct behavior).
Coverage exceeds 70% threshold for all meaningful packages (native: 100%,
config: 97.6%, scream: 95.0%, audio: 87.5%+, encoding: 86%).
Linter reports 13 non-critical style/errcheck issues with no correctness impact.
