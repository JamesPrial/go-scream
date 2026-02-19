# Code Review: Stage 3 - FFmpeg Backend (Retry)

**Reviewer:** Go Reviewer Agent
**Date:** 2026-02-18
**Files Reviewed:**
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/errors.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command_test.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator_test.go`

---

## Fix Verification

### Fix 1: OS error inclusion in NewFFmpegGenerator

**Status: VERIFIED**

At `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go`, line 25:

```go
return nil, fmt.Errorf("%w: %w", ErrFFmpegNotFound, err)
```

This correctly wraps both the sentinel error (`ErrFFmpegNotFound`) and the underlying OS error from `exec.LookPath` using Go 1.20+ multi-error wrapping with `%w`. Callers can use `errors.Is(err, ErrFFmpegNotFound)` to detect the condition and can also inspect the original OS error for diagnostics. Correct.

### Fix 2: Subtest names use `fmt.Sprintf("seed_%d", seed)` for clarity

**Status: VERIFIED**

Found in two locations:

- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command_test.go`, line 721:
  ```go
  t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
  ```

- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator_test.go`, line 159:
  ```go
  t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
  ```

Both use `fmt.Sprintf("seed_%d", seed)` for human-readable subtest names. The table-driven test `Test_buildAevalsrcExpr_NonEmptyForAllPresets` in `command_test.go` (line 218-239) also uses named subtests with the `seed_N` convention via its struct field (`tt.name`). Correct and consistent.

---

## Full Code Review

### errors.go

Clean and minimal. Two sentinel errors are defined with descriptive messages and proper documentation. Package-level doc comment is present.

No issues.

### command.go

**Documentation:** All exported functions (`BuildArgs`) are documented. All unexported helpers (`buildAevalsrcExpr`, `layerExpr`, `buildFilterChain`, `fmtFloat`, `deriveSeed`) have doc comments explaining their purpose. This exceeds Go standards for unexported items, which is appreciated.

**Error handling:** `BuildArgs` is a pure function that constructs arguments; validation is correctly deferred to the `Generate` method where `params.Validate()` is called before `BuildArgs`. This separation is clean.

**Nil safety:** `buildAevalsrcExpr` pre-allocates the slice with `make([]string, 0, len(params.Layers))` -- safe since `Layers` is a fixed-size `[5]LayerParams` array. The `deriveSeed` function handles the `math.MinInt64` edge case explicitly at line 160-161 (negating `math.MinInt64` overflows in two's complement), which is excellent attention to detail.

**Logic review:**
- `layerExpr` correctly returns `"0"` for zero-amplitude layers across all layer types.
- The `LayerNoiseBurst` and `LayerBackgroundNoise` cases correctly pull amplitude from `NoiseParams` rather than `LayerParams`, matching the data model intent.
- The `default` case returns `"0"`, which is a safe fallback for unknown layer types.
- `fmtFloat` uses 6 decimal places (`'f', 6`) which provides sufficient precision for audio parameters without scientific notation that could confuse FFmpeg.
- Filter chain ordering (highpass, lowpass, acrusher, acompressor, volume, alimiter) is a sensible audio processing pipeline order.

**Minor observation (not blocking):** The `sampleRate` constant on line 54 is hardcoded to `"48000"` with a comment explaining it is for random seeding within aevalsrc, not for the output sample rate. This is correct -- FFmpeg's `random()` function in aevalsrc uses sample-rate-dependent time indexing, and keeping this constant ensures seed stability regardless of the output sample rate. The comment adequately explains the intent.

No issues.

### generator.go

**Documentation:** All exported types and functions have doc comments. The compile-time interface check `var _ audio.AudioGenerator = (*FFmpegGenerator)(nil)` at line 13 is good practice.

**Error handling:**
- `NewFFmpegGenerator` (line 22-28): Wraps both `ErrFFmpegNotFound` and the OS error using multi-`%w` (verified fix).
- `Generate` (line 38-55): Validates params first, then wraps ffmpeg failures with `ErrFFmpegFailed` and stderr output. The error at line 51 uses `%w` for the sentinel and `%s` for stderr, which is correct -- stderr is diagnostic text, not an error to unwrap.
- `NewFFmpegGeneratorWithPath` (line 32-34): Documented as performing no validation, which is appropriate for a "bring your own path" constructor.

**Pattern consistency:** The `Generate` method follows the same pattern as the native generator in `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go` (line 28-29): `params.Validate()` is called first with the same wrapping format `"invalid params: %w"`. Good consistency.

**Resource management:** `exec.Command` with `cmd.Run()` (which waits for completion) paired with `bytes.Buffer` for stdout/stderr is correct. No resource leaks -- the process completes before the function returns.

No issues.

### command_test.go

**Test naming:** All test functions follow the `TestXxx` or `Test_xxx` convention. Table-driven subtests use descriptive names.

**Coverage breadth:**
- `BuildArgs`: 10 test functions covering lavfi input, aevalsrc presence, audio filter flag, output format, channels, sample rate, pipe output, duration, mono, and different sample rates. Two comprehensive table-driven tests cover all presets and randomized params.
- `buildAevalsrcExpr`: Tests for sin, random, PI presence; non-empty output across seeds; zero-amplitude edge case.
- `buildFilterChain`: Individual tests for each of the 6 filters plus a filter-ordering test.
- `layerExpr`: One test per layer type (5 total) plus zero-amplitude edge case.
- `fmtFloat`: Table-driven test with 6 cases, precision consistency test, negative value test.
- `deriveSeed`: Different indexes, different global seeds, determinism, non-negative (table-driven with 7 cases), different layer seeds.
- Benchmarks for `BuildArgs`, `buildAevalsrcExpr`, and `buildFilterChain`.

**Edge cases covered:**
- Zero amplitude layers
- Mono vs stereo
- Different sample rates
- Multiple randomized seeds
- All named presets
- Negative seed inputs for `deriveSeed`

**Test quality:** Tests validate behavior rather than implementation details. The `Test_buildFilterChain_FilterOrder` test (line 396-436) is particularly well-structured, verifying the relative ordering of all 6 filters using position comparison.

No issues.

### generator_test.go

**Test naming:** All test functions follow `TestXxx` convention. Proper use of `t.Helper()` in `skipIfNoFFmpeg`.

**Coverage breadth:**
- Constructor: success, with-path, sentinel error existence.
- Generate output: correct byte count, non-silence, all random seeds, all named presets.
- Error conditions: zero/negative duration, zero/negative sample rate, invalid channels (0, 3), bad binary path, invalid amplitude, invalid crusher bits, invalid limiter level.
- Output format: even byte count (s16le alignment), stereo 4-byte frame alignment, mono byte count.
- Determinism: same-params-same-output verification.
- Benchmark for 1s generation.

**Integration test guards:** All tests that invoke ffmpeg use `skipIfNoFFmpeg(t)` appropriately. The `TestFFmpegGenerator_BadBinaryPath` test correctly does not call `skipIfNoFFmpeg` since it tests the error path without needing a real ffmpeg.

**Error checking pattern:** All error condition tests verify both that an error is returned and that it wraps the correct sentinel error using `errors.Is`. This is thorough.

**Observation (not blocking):** The `TestNewFFmpegGenerator_NoFFmpegOnPath` test (line 73-87) documents the expected behavior in a comment but only verifies the sentinel errors are non-nil rather than actually testing the LookPath failure path. This is acknowledged in the comment as a PATH-manipulation limitation. The actual error wrapping is indirectly tested by the constructor success test and the verified fix. Acceptable.

No issues.

---

## Review Checklist

- [x] All exported items have documentation
- [x] Error handling follows patterns (sentinel wrapping with `%w`, multi-error wrapping)
- [x] Nil safety guards present (`math.MinInt64` edge case, zero-amplitude checks)
- [x] Table tests structured correctly with descriptive subtest names
- [x] Code is readable and well-organized
- [x] Naming conventions followed (Go idioms, `TestXxx`, unexported helpers)
- [x] No logic errors or edge case gaps detected
- [x] Consistency with existing codebase patterns (native generator)
- [x] Compile-time interface check present
- [x] Integration tests properly guarded with skip helpers

---

## Verdict

**APPROVE**

Both requested fixes have been correctly applied. The code is well-documented, follows idiomatic Go patterns, handles errors properly with sentinel wrapping, covers edge cases in both implementation and tests, and is consistent with the existing native generator codebase. No remaining issues found.
