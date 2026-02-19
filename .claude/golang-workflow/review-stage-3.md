# Code Review: Stage 3 -- FFmpeg Backend

**Verdict: REQUEST_CHANGES**

## Files Reviewed

- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/errors.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command_test.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator_test.go`

## Summary

The FFmpeg backend is well-structured and follows the project patterns established by the NativeGenerator. The code demonstrates good separation of concerns (command building vs. process execution), proper error wrapping, compile-time interface verification, and thorough test coverage. Two issues must be addressed before approval.

---

## Required Changes

### 1. Unnecessary error wrapping in `NewFFmpegGenerator` (minor)

**File:** `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go`, line 25

```go
return nil, fmt.Errorf("%w", ErrFFmpegNotFound)
```

This wraps `ErrFFmpegNotFound` with `fmt.Errorf("%w", ...)` but adds no additional context. The format string is just `"%w"`, so the resulting error message is identical to the sentinel itself. Either add meaningful context (e.g., the underlying `exec.LookPath` error) or return the sentinel directly:

**Option A** -- return sentinel directly:
```go
return nil, ErrFFmpegNotFound
```

**Option B** -- wrap with context from exec.LookPath:
```go
return nil, fmt.Errorf("%w: %v", ErrFFmpegNotFound, err)
```

Option B is preferred because it preserves the underlying OS-level error message (e.g., "executable file not found in $PATH") for debugging, while still allowing callers to match with `errors.Is(err, ErrFFmpegNotFound)`.

### 2. Confusing subtest names in `TestFFmpegGenerator_AllPresets`

**File:** `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator_test.go`, line 158

```go
t.Run("seed_"+string(rune('0'+seed%10)), func(t *testing.T) {
```

This produces subtest names that do not match the actual seed values. For example, seed `42` becomes subtest `seed_2`, and seed `100` becomes `seed_0`. This harms test output readability and makes failure diagnosis harder. Use `fmt.Sprintf` for clarity:

```go
t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
```

Note: the `fmt` package is already imported in this file.

---

## Observations (non-blocking, for awareness)

### Hardcoded sample rate constant in `layerExpr`

**File:** `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go`, line 54

```go
sampleRate := "48000" // aevalsrc uses its own sample rate; use a constant for the random seeding
```

The `random(t*48000)` inside noise layer expressions uses a hardcoded `"48000"` rather than the params sample rate. The comment explains the intent: this value seeds the random function's sampling density, not the output sample rate. The `aevalsrc` `s=` parameter on line 21 does correctly use `params.SampleRate`. This is acceptable as-is since `48000` is effectively a constant for the expression's random granularity, but it would be slightly cleaner as a named package-level constant with a more descriptive name (e.g., `exprRandomSampleRate`).

### Duplicated `FilterParams` literal across filter chain tests

**File:** `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command_test.go`, lines 260-394

The same `audio.FilterParams{...}` struct literal is repeated verbatim in six test functions (`Test_buildFilterChain_ContainsHighpass` through `Test_buildFilterChain_FilterOrder`). Extracting a `testFilterParams()` helper (similar to the existing `classicParams()`) would reduce duplication and make it easier to update the test values in one place.

### Memory usage for long durations

**File:** `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go`, line 46-54

The entire FFmpeg output is buffered into `bytes.Buffer` before returning. For a 4-second stereo 48kHz s16le output, this is approximately 768KB, which is acceptable. However, if the duration is ever extended significantly, this could become a concern. The NativeGenerator uses the same buffered approach, so this is consistent with the project pattern. No action needed now.

### Process lifecycle

The use of `cmd.Run()` (line 50) is correct -- it waits for process completion and collects all output. There are no resource leaks: stdout and stderr are captured via `bytes.Buffer` on the command struct, and `cmd.Run()` handles pipe cleanup internally. No goroutines or manual pipe management are needed.

---

## Checklist

**Code Quality:**
- [x] All exported items have documentation (`FFmpegGenerator`, `NewFFmpegGenerator`, `NewFFmpegGeneratorWithPath`, `Generate`, `BuildArgs`, `ErrFFmpegNotFound`, `ErrFFmpegFailed`)
- [x] Error handling follows patterns -- sentinel errors, `%w` wrapping, `errors.Is` compatibility
- [x] Nil safety guards present -- `params.Validate()` called before use, ffmpeg path checked at construction time
- [x] Table tests structured correctly -- `Test_BuildArgs_AllPresets`, `Test_deriveSeed_NonNegative`, `Test_fmtFloat_Cases` use proper table-driven patterns
- [x] Code is readable and well-organized -- clear separation between command building and process execution
- [x] Naming conventions followed -- `FFmpegGenerator`, `BuildArgs`, `ErrFFmpegNotFound` all follow Go conventions
- [ ] No obvious logic errors or edge case gaps -- subtest naming issue in `TestFFmpegGenerator_AllPresets`

**Pattern Consistency with NativeGenerator:**
- [x] Same `Generate(ScreamParams) (io.Reader, error)` signature
- [x] Same `params.Validate()` call at the top of Generate
- [x] Same error wrapping pattern: `fmt.Errorf("invalid params: %w", err)`
- [x] Same compile-time interface check: `var _ audio.AudioGenerator = (*FFmpegGenerator)(nil)`
- [x] Same return type: `io.Reader` backed by `bytes.NewReader`

**Test Coverage Assessment:**
- [x] Constructor success and error paths tested
- [x] All five layer types tested individually in `layerExpr` tests
- [x] Zero-amplitude edge case covered
- [x] Filter chain order verified
- [x] Output byte count, alignment, and non-silence verified
- [x] All named presets and randomized params exercised
- [x] Determinism verified (same params produce same output)
- [x] Invalid param error paths tested (duration, sample rate, channels, amplitude, crusher bits, limiter level)
- [x] Bad binary path error path tested
- [x] Benchmarks included for `BuildArgs`, `buildAevalsrcExpr`, `buildFilterChain`, and full `Generate`
