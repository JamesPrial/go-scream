# Test Execution Report - Stage 3 (FFmpeg Backend)

**Date:** 2026-02-18  
**Package:** `github.com/JamesPrial/go-scream/internal/audio/ffmpeg`  
**Implementation files:**
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/errors.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go`

---

## Summary

- **Verdict:** TESTS_PASS
- **Tests Run:** 41 passed, 0 failed, 18 skipped (ffmpeg not on PATH)
- **Coverage:** 90.6%
- **Race Conditions:** None
- **Vet Warnings:** None
- **Linter:** 0 issues (golangci-lint)

---

## Test Results

```
=== RUN   Test_BuildArgs_ContainsLavfiInput
--- PASS: Test_BuildArgs_ContainsLavfiInput (0.00s)
=== RUN   Test_BuildArgs_ContainsAevalsrc
--- PASS: Test_BuildArgs_ContainsAevalsrc (0.00s)
=== RUN   Test_BuildArgs_ContainsAudioFilter
--- PASS: Test_BuildArgs_ContainsAudioFilter (0.00s)
=== RUN   Test_BuildArgs_ContainsOutputFormat
--- PASS: Test_BuildArgs_ContainsOutputFormat (0.00s)
=== RUN   Test_BuildArgs_ContainsChannels
--- PASS: Test_BuildArgs_ContainsChannels (0.00s)
=== RUN   Test_BuildArgs_ContainsSampleRate
--- PASS: Test_BuildArgs_ContainsSampleRate (0.00s)
=== RUN   Test_BuildArgs_LastArgIsPipe
--- PASS: Test_BuildArgs_LastArgIsPipe (0.00s)
=== RUN   Test_BuildArgs_ContainsDuration
--- PASS: Test_BuildArgs_ContainsDuration (0.00s)
=== RUN   Test_BuildArgs_MonoParams
--- PASS: Test_BuildArgs_MonoParams (0.00s)
=== RUN   Test_BuildArgs_DifferentSampleRate
--- PASS: Test_BuildArgs_DifferentSampleRate (0.00s)
=== RUN   Test_buildAevalsrcExpr_ContainsSin
--- PASS: Test_buildAevalsrcExpr_ContainsSin (0.00s)
=== RUN   Test_buildAevalsrcExpr_ContainsRandom
--- PASS: Test_buildAevalsrcExpr_ContainsRandom (0.00s)
=== RUN   Test_buildAevalsrcExpr_ContainsPI
--- PASS: Test_buildAevalsrcExpr_ContainsPI (0.00s)
=== RUN   Test_buildAevalsrcExpr_NonEmptyForAllPresets
=== RUN   Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_1
=== RUN   Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_42
=== RUN   Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_100
=== RUN   Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_9999
=== RUN   Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_12345
=== RUN   Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_99999
--- PASS: Test_buildAevalsrcExpr_NonEmptyForAllPresets (0.00s)
    --- PASS: Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_1 (0.00s)
    --- PASS: Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_42 (0.00s)
    --- PASS: Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_100 (0.00s)
    --- PASS: Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_9999 (0.00s)
    --- PASS: Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_12345 (0.00s)
    --- PASS: Test_buildAevalsrcExpr_NonEmptyForAllPresets/seed_99999 (0.00s)
=== RUN   Test_buildAevalsrcExpr_ZeroAmplitudeLayer
--- PASS: Test_buildAevalsrcExpr_ZeroAmplitudeLayer (0.00s)
=== RUN   Test_buildFilterChain_ContainsHighpass
--- PASS: Test_buildFilterChain_ContainsHighpass (0.00s)
=== RUN   Test_buildFilterChain_ContainsLowpass
--- PASS: Test_buildFilterChain_ContainsLowpass (0.00s)
=== RUN   Test_buildFilterChain_ContainsAcrusher
--- PASS: Test_buildFilterChain_ContainsAcrusher (0.00s)
=== RUN   Test_buildFilterChain_ContainsAcompressor
--- PASS: Test_buildFilterChain_ContainsAcompressor (0.00s)
=== RUN   Test_buildFilterChain_ContainsVolume
--- PASS: Test_buildFilterChain_ContainsVolume (0.00s)
=== RUN   Test_buildFilterChain_ContainsAlimiter
--- PASS: Test_buildFilterChain_ContainsAlimiter (0.00s)
=== RUN   Test_buildFilterChain_FilterOrder
--- PASS: Test_buildFilterChain_FilterOrder (0.00s)
=== RUN   Test_layerExpr_PrimaryScream
--- PASS: Test_layerExpr_PrimaryScream (0.00s)
=== RUN   Test_layerExpr_HarmonicSweep
--- PASS: Test_layerExpr_HarmonicSweep (0.00s)
=== RUN   Test_layerExpr_HighShriek
--- PASS: Test_layerExpr_HighShriek (0.00s)
=== RUN   Test_layerExpr_NoiseBurst
--- PASS: Test_layerExpr_NoiseBurst (0.00s)
=== RUN   Test_layerExpr_BackgroundNoise
--- PASS: Test_layerExpr_BackgroundNoise (0.00s)
=== RUN   Test_layerExpr_ZeroAmplitude
--- PASS: Test_layerExpr_ZeroAmplitude (0.00s)
=== RUN   Test_fmtFloat_Cases
=== RUN   Test_fmtFloat_Cases/integer_value
=== RUN   Test_fmtFloat_Cases/fractional_value
=== RUN   Test_fmtFloat_Cases/small_value
=== RUN   Test_fmtFloat_Cases/negative_value
=== RUN   Test_fmtFloat_Cases/zero
=== RUN   Test_fmtFloat_Cases/large_value
--- PASS: Test_fmtFloat_Cases (0.00s)
    --- PASS: Test_fmtFloat_Cases/integer_value (0.00s)
    --- PASS: Test_fmtFloat_Cases/fractional_value (0.00s)
    --- PASS: Test_fmtFloat_Cases/small_value (0.00s)
    --- PASS: Test_fmtFloat_Cases/negative_value (0.00s)
    --- PASS: Test_fmtFloat_Cases/zero (0.00s)
    --- PASS: Test_fmtFloat_Cases/large_value (0.00s)
=== RUN   Test_fmtFloat_ConsistentPrecision
--- PASS: Test_fmtFloat_ConsistentPrecision (0.00s)
=== RUN   Test_fmtFloat_NegativeValue
--- PASS: Test_fmtFloat_NegativeValue (0.00s)
=== RUN   Test_deriveSeed_DifferentIndexes
--- PASS: Test_deriveSeed_DifferentIndexes (0.00s)
=== RUN   Test_deriveSeed_DifferentGlobalSeeds
--- PASS: Test_deriveSeed_DifferentGlobalSeeds (0.00s)
=== RUN   Test_deriveSeed_Deterministic
--- PASS: Test_deriveSeed_Deterministic (0.00s)
=== RUN   Test_deriveSeed_NonNegative
=== RUN   Test_deriveSeed_NonNegative/positive_seeds
=== RUN   Test_deriveSeed_NonNegative/zero_global
=== RUN   Test_deriveSeed_NonNegative/zero_layer
=== RUN   Test_deriveSeed_NonNegative/large_seeds
=== RUN   Test_deriveSeed_NonNegative/negative_global
=== RUN   Test_deriveSeed_NonNegative/negative_layer
=== RUN   Test_deriveSeed_NonNegative/both_negative
--- PASS: Test_deriveSeed_NonNegative (0.00s)
    --- PASS: Test_deriveSeed_NonNegative/positive_seeds (0.00s)
    --- PASS: Test_deriveSeed_NonNegative/zero_global (0.00s)
    --- PASS: Test_deriveSeed_NonNegative/zero_layer (0.00s)
    --- PASS: Test_deriveSeed_NonNegative/large_seeds (0.00s)
    --- PASS: Test_deriveSeed_NonNegative/negative_global (0.00s)
    --- PASS: Test_deriveSeed_NonNegative/negative_layer (0.00s)
    --- PASS: Test_deriveSeed_NonNegative/both_negative (0.00s)
=== RUN   Test_deriveSeed_DifferentLayerSeeds
--- PASS: Test_deriveSeed_DifferentLayerSeeds (0.00s)
=== RUN   Test_BuildArgs_AllPresets
=== RUN   Test_BuildArgs_AllPresets/classic
=== RUN   Test_BuildArgs_AllPresets/whisper
=== RUN   Test_BuildArgs_AllPresets/death-metal
=== RUN   Test_BuildArgs_AllPresets/glitch
=== RUN   Test_BuildArgs_AllPresets/banshee
=== RUN   Test_BuildArgs_AllPresets/robot
--- PASS: Test_BuildArgs_AllPresets (0.00s)
    --- PASS: Test_BuildArgs_AllPresets/classic (0.00s)
    --- PASS: Test_BuildArgs_AllPresets/whisper (0.00s)
    --- PASS: Test_BuildArgs_AllPresets/death-metal (0.00s)
    --- PASS: Test_BuildArgs_AllPresets/glitch (0.00s)
    --- PASS: Test_BuildArgs_AllPresets/banshee (0.00s)
    --- PASS: Test_BuildArgs_AllPresets/robot (0.00s)
=== RUN   Test_BuildArgs_WithRandomizedParams
=== RUN   Test_BuildArgs_WithRandomizedParams/seed_1
=== RUN   Test_BuildArgs_WithRandomizedParams/seed_42
=== RUN   Test_BuildArgs_WithRandomizedParams/seed_100
=== RUN   Test_BuildArgs_WithRandomizedParams/seed_9999
=== RUN   Test_BuildArgs_WithRandomizedParams/seed_12345
--- PASS: Test_BuildArgs_WithRandomizedParams (0.00s)
    --- PASS: Test_BuildArgs_WithRandomizedParams/seed_1 (0.00s)
    --- PASS: Test_BuildArgs_WithRandomizedParams/seed_42 (0.00s)
    --- PASS: Test_BuildArgs_WithRandomizedParams/seed_100 (0.00s)
    --- PASS: Test_BuildArgs_WithRandomizedParams/seed_9999 (0.00s)
    --- PASS: Test_BuildArgs_WithRandomizedParams/seed_12345 (0.00s)
=== RUN   TestNewFFmpegGenerator_Success
    generator_test.go:54: ffmpeg not available
--- SKIP: TestNewFFmpegGenerator_Success (0.00s)
=== RUN   TestNewFFmpegGeneratorWithPath_NotNil
--- PASS: TestNewFFmpegGeneratorWithPath_NotNil (0.00s)
=== RUN   TestNewFFmpegGenerator_NoFFmpegOnPath
--- PASS: TestNewFFmpegGenerator_NoFFmpegOnPath (0.00s)
=== RUN   TestFFmpegGenerator_CorrectByteCount
    generator_test.go:91: ffmpeg not available
--- SKIP: TestFFmpegGenerator_CorrectByteCount (0.00s)
=== RUN   TestFFmpegGenerator_NonSilent
    generator_test.go:117: ffmpeg not available
--- SKIP: TestFFmpegGenerator_NonSilent (0.00s)
=== RUN   TestFFmpegGenerator_AllPresets
    generator_test.go:148: ffmpeg not available
--- SKIP: TestFFmpegGenerator_AllPresets (0.00s)
=== RUN   TestFFmpegGenerator_AllNamedPresets
    generator_test.go:181: ffmpeg not available
--- SKIP: TestFFmpegGenerator_AllNamedPresets (0.00s)
=== RUN   TestFFmpegGenerator_InvalidDuration
    generator_test.go:217: ffmpeg not available
--- SKIP: TestFFmpegGenerator_InvalidDuration (0.00s)
=== RUN   TestFFmpegGenerator_NegativeDuration
    generator_test.go:237: ffmpeg not available
--- SKIP: TestFFmpegGenerator_NegativeDuration (0.00s)
=== RUN   TestFFmpegGenerator_InvalidSampleRate
    generator_test.go:257: ffmpeg not available
--- SKIP: TestFFmpegGenerator_InvalidSampleRate (0.00s)
=== RUN   TestFFmpegGenerator_NegativeSampleRate
    generator_test.go:277: ffmpeg not available
--- SKIP: TestFFmpegGenerator_NegativeSampleRate (0.00s)
=== RUN   TestFFmpegGenerator_InvalidChannels
    generator_test.go:297: ffmpeg not available
--- SKIP: TestFFmpegGenerator_InvalidChannels (0.00s)
=== RUN   TestFFmpegGenerator_ZeroChannels
    generator_test.go:317: ffmpeg not available
--- SKIP: TestFFmpegGenerator_ZeroChannels (0.00s)
=== RUN   TestFFmpegGenerator_BadBinaryPath
--- PASS: TestFFmpegGenerator_BadBinaryPath (0.00s)
=== RUN   TestFFmpegGenerator_InvalidAmplitude
    generator_test.go:350: ffmpeg not available
--- SKIP: TestFFmpegGenerator_InvalidAmplitude (0.00s)
=== RUN   TestFFmpegGenerator_InvalidCrusherBits
    generator_test.go:370: ffmpeg not available
--- SKIP: TestFFmpegGenerator_InvalidCrusherBits (0.00s)
=== RUN   TestFFmpegGenerator_InvalidLimiterLevel
    generator_test.go:390: ffmpeg not available
--- SKIP: TestFFmpegGenerator_InvalidLimiterLevel (0.00s)
=== RUN   TestFFmpegGenerator_EvenByteCount
    generator_test.go:412: ffmpeg not available
--- SKIP: TestFFmpegGenerator_EvenByteCount (0.00s)
=== RUN   TestFFmpegGenerator_StereoAligned
    generator_test.go:436: ffmpeg not available
--- SKIP: TestFFmpegGenerator_StereoAligned (0.00s)
=== RUN   TestFFmpegGenerator_MonoOutput
    generator_test.go:462: ffmpeg not available
--- SKIP: TestFFmpegGenerator_MonoOutput (0.00s)
=== RUN   TestFFmpegGenerator_DeterministicOutput
    generator_test.go:491: ffmpeg not available
--- SKIP: TestFFmpegGenerator_DeterministicOutput (0.00s)
PASS
ok      github.com/JamesPrial/go-scream/internal/audio/ffmpeg   0.325s
```

### Skip Explanation

18 integration tests were skipped because `ffmpeg` is not installed on this system's PATH. All skipped tests use the `skipIfNoFFmpeg(t)` helper which correctly calls `t.Skip()`. This is expected behaviour - the tests are correctly written to degrade gracefully when the runtime dependency is absent. All 41 non-integration tests (command building, expression building, filter chain, seed derivation, float formatting) executed and passed.

---

## Race Detection

```
ok      github.com/JamesPrial/go-scream/internal/audio/ffmpeg   1.354s
```

No races detected.

---

## Static Analysis

```
(no output)
```

`go vet ./internal/audio/ffmpeg/...` produced no warnings.

---

## Coverage Details

```
ok      github.com/JamesPrial/go-scream/internal/audio/ffmpeg   0.304s  coverage: 90.6% of statements
```

Coverage: **90.6%** - well above the 70% threshold.

The uncovered statements are integration paths that require a live `ffmpeg` binary (the `NewFFmpegGenerator` success path and the `Generate` execution path when ffmpeg is available). These are intentionally skipped, not untested by design.

---

## Linter Output

```
0 issues.
```

`golangci-lint` reported 0 issues.

---

## Regression Tests (Prior Stages)

All prior-stage packages continue to pass:

```
ok      github.com/JamesPrial/go-scream/internal/audio          0.487s
ok      github.com/JamesPrial/go-scream/internal/audio/ffmpeg   0.245s
ok      github.com/JamesPrial/go-scream/internal/audio/native   1.112s
ok      github.com/JamesPrial/go-scream/internal/encoding       0.510s
```

No regressions introduced.

---

## Pass Criteria Checklist

- [x] All `go test` commands exit with status 0
- [x] No race conditions detected by `-race`
- [x] No warnings from `go vet`
- [x] Coverage 90.6% (threshold: >70%)
- [x] No critical linter errors (golangci-lint: 0 issues)
- [x] All prior-stage packages pass regression

---

## TESTS_PASS

All checks pass. Coverage: **90.6%**. 41 tests run, 0 failed, 18 skipped (ffmpeg binary not present in test environment - correct skip behaviour via `t.Skip()`).
