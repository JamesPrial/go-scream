## Test Execution Report — Stage 1

### Summary
- **Verdict:** TESTS_PASS
- **Tests Run:** 65 passed, 0 failed
- **Coverage:** `internal/audio` 90.0% | `internal/audio/native` 100.0%
- **Race Conditions:** None
- **Vet Warnings:** None
- **Linter (golangci-lint):** 2 non-critical `errcheck` warnings in test files only

---

### Test Results

#### `go test -v ./internal/audio/...`

```
=== RUN   TestRandomize_ProducesValidParams
--- PASS: TestRandomize_ProducesValidParams (0.00s)
=== RUN   TestRandomize_Deterministic
--- PASS: TestRandomize_Deterministic (0.00s)
=== RUN   TestRandomize_DifferentSeeds
--- PASS: TestRandomize_DifferentSeeds (0.00s)
=== RUN   TestRandomize_ZeroSeed
--- PASS: TestRandomize_ZeroSeed (0.00s)
=== RUN   TestValidate_ValidParams
--- PASS: TestValidate_ValidParams (0.00s)
=== RUN   TestValidate_InvalidDuration
--- PASS: TestValidate_InvalidDuration (0.00s)
=== RUN   TestValidate_InvalidSampleRate
--- PASS: TestValidate_InvalidSampleRate (0.00s)
=== RUN   TestValidate_InvalidChannels
--- PASS: TestValidate_InvalidChannels (0.00s)
=== RUN   TestValidate_InvalidAmplitude
--- PASS: TestValidate_InvalidAmplitude (0.00s)
=== RUN   TestValidate_InvalidLimiterLevel
    --- PASS: TestValidate_InvalidLimiterLevel/zero (0.00s)
    --- PASS: TestValidate_InvalidLimiterLevel/negative (0.00s)
    --- PASS: TestValidate_InvalidLimiterLevel/above_one (0.00s)
--- PASS: TestValidate_InvalidLimiterLevel (0.00s)
=== RUN   TestAllPresets_ReturnsAll6
--- PASS: TestAllPresets_ReturnsAll6 (0.00s)
=== RUN   TestGetPreset_AllPresetsValid
    --- PASS: TestGetPreset_AllPresetsValid/classic (0.00s)
    --- PASS: TestGetPreset_AllPresetsValid/whisper (0.00s)
    --- PASS: TestGetPreset_AllPresetsValid/death-metal (0.00s)
    --- PASS: TestGetPreset_AllPresetsValid/glitch (0.00s)
    --- PASS: TestGetPreset_AllPresetsValid/banshee (0.00s)
    --- PASS: TestGetPreset_AllPresetsValid/robot (0.00s)
--- PASS: TestGetPreset_AllPresetsValid (0.00s)
=== RUN   TestGetPreset_Unknown
--- PASS: TestGetPreset_Unknown (0.00s)
=== RUN   TestGetPreset_ParameterRanges
    --- PASS: TestGetPreset_ParameterRanges/classic (0.00s)
    --- PASS: TestGetPreset_ParameterRanges/whisper (0.00s)
    --- PASS: TestGetPreset_ParameterRanges/death-metal (0.00s)
    --- PASS: TestGetPreset_ParameterRanges/glitch (0.00s)
    --- PASS: TestGetPreset_ParameterRanges/banshee (0.00s)
    --- PASS: TestGetPreset_ParameterRanges/robot (0.00s)
--- PASS: TestGetPreset_ParameterRanges (0.00s)
PASS
ok  github.com/JamesPrial/go-scream/internal/audio  (cached)
```

#### `go test -v ./internal/audio/native/...`

```
=== RUN   TestHighpassFilter_RemovesDC
--- PASS: TestHighpassFilter_RemovesDC (0.00s)
=== RUN   TestHighpassFilter_PassesHighFreq
--- PASS: TestHighpassFilter_PassesHighFreq (0.00s)
=== RUN   TestLowpassFilter_PassesDC
--- PASS: TestLowpassFilter_PassesDC (0.00s)
=== RUN   TestLowpassFilter_AttenuatesHighFreq
--- PASS: TestLowpassFilter_AttenuatesHighFreq (0.00s)
=== RUN   TestBitcrusher_FullMix
    --- PASS: TestBitcrusher_FullMix/positive (0.00s)
    --- PASS: TestBitcrusher_FullMix/negative (0.00s)
    --- PASS: TestBitcrusher_FullMix/zero (0.00s)
    --- PASS: TestBitcrusher_FullMix/one (0.00s)
--- PASS: TestBitcrusher_FullMix (0.00s)
=== RUN   TestBitcrusher_ZeroMix
--- PASS: TestBitcrusher_ZeroMix (0.00s)
=== RUN   TestBitcrusher_Blend
--- PASS: TestBitcrusher_Blend (0.00s)
=== RUN   TestCompressor_BelowThreshold
--- PASS: TestCompressor_BelowThreshold (0.00s)
=== RUN   TestCompressor_AboveThreshold
--- PASS: TestCompressor_AboveThreshold (0.00s)
=== RUN   TestCompressor_PreservesSign
--- PASS: TestCompressor_PreservesSign (0.00s)
=== RUN   TestVolumeBoost_ZeroDB
--- PASS: TestVolumeBoost_ZeroDB (0.00s)
=== RUN   TestVolumeBoost_6dB
--- PASS: TestVolumeBoost_6dB (0.00s)
=== RUN   TestVolumeBoost_NegativeDB
--- PASS: TestVolumeBoost_NegativeDB (0.00s)
=== RUN   TestLimiter_WithinRange
--- PASS: TestLimiter_WithinRange (0.00s)
=== RUN   TestLimiter_ClipsPositive
--- PASS: TestLimiter_ClipsPositive (0.00s)
=== RUN   TestLimiter_ClipsNegative
--- PASS: TestLimiter_ClipsNegative (0.00s)
=== RUN   TestFilterChain_OrderMatters
--- PASS: TestFilterChain_OrderMatters (0.00s)
=== RUN   TestFilterChainFromParams_ClassicPreset
--- PASS: TestFilterChainFromParams_ClassicPreset (0.00s)
=== RUN   TestHighpassFilter_ImplementsFilter
--- PASS: TestHighpassFilter_ImplementsFilter (0.00s)
=== RUN   TestLowpassFilter_ImplementsFilter
--- PASS: TestLowpassFilter_ImplementsFilter (0.00s)
=== RUN   TestBitcrusher_ImplementsFilter
--- PASS: TestBitcrusher_ImplementsFilter (0.00s)
=== RUN   TestCompressor_ImplementsFilter
--- PASS: TestCompressor_ImplementsFilter (0.00s)
=== RUN   TestVolumeBoost_ImplementsFilter
--- PASS: TestVolumeBoost_ImplementsFilter (0.00s)
=== RUN   TestLimiter_ImplementsFilter
--- PASS: TestLimiter_ImplementsFilter (0.00s)
=== RUN   TestFilterChain_ImplementsFilter
--- PASS: TestFilterChain_ImplementsFilter (0.00s)
=== RUN   TestNativeGenerator_CorrectByteCount
--- PASS: TestNativeGenerator_CorrectByteCount (8.47s)
=== RUN   TestNativeGenerator_NonSilent
--- PASS: TestNativeGenerator_NonSilent (8.55s)
=== RUN   TestNativeGenerator_Deterministic
--- PASS: TestNativeGenerator_Deterministic (17.35s)
=== RUN   TestNativeGenerator_DifferentSeeds
--- PASS: TestNativeGenerator_DifferentSeeds (17.12s)
=== RUN   TestNativeGenerator_AllPresets
    --- PASS: TestNativeGenerator_AllPresets/classic (8.47s)
    --- PASS: TestNativeGenerator_AllPresets/whisper (5.48s)
    --- PASS: TestNativeGenerator_AllPresets/death-metal (11.70s)
    --- PASS: TestNativeGenerator_AllPresets/glitch (8.76s)
    --- PASS: TestNativeGenerator_AllPresets/banshee (11.10s)
    --- PASS: TestNativeGenerator_AllPresets/robot (8.67s)
--- PASS: TestNativeGenerator_AllPresets (54.18s)
=== RUN   TestNativeGenerator_InvalidParams
--- PASS: TestNativeGenerator_InvalidParams (0.00s)
=== RUN   TestNativeGenerator_MonoOutput
--- PASS: TestNativeGenerator_MonoOutput (8.49s)
=== RUN   TestNativeGenerator_S16LERange
--- PASS: TestNativeGenerator_S16LERange (8.65s)
=== RUN   TestNativeGenerator_ImplementsInterface
--- PASS: TestNativeGenerator_ImplementsInterface (0.00s)
=== RUN   TestPrimaryScreamLayer_NonZeroOutput
--- PASS: TestPrimaryScreamLayer_NonZeroOutput (0.00s)
=== RUN   TestPrimaryScreamLayer_AmplitudeBounds
--- PASS: TestPrimaryScreamLayer_AmplitudeBounds (1.65s)
=== RUN   TestHarmonicSweepLayer_NonZeroOutput
--- PASS: TestHarmonicSweepLayer_NonZeroOutput (0.00s)
=== RUN   TestHighShriekLayer_NonZeroOutput
--- PASS: TestHighShriekLayer_NonZeroOutput (0.00s)
=== RUN   TestHighShriekLayer_EnvelopeRises
--- PASS: TestHighShriekLayer_EnvelopeRises (1.65s)
=== RUN   TestNoiseBurstLayer_HasSilentAndActiveSegments
--- PASS: TestNoiseBurstLayer_HasSilentAndActiveSegments (0.00s)
=== RUN   TestBackgroundNoiseLayer_ContinuousOutput
--- PASS: TestBackgroundNoiseLayer_ContinuousOutput (0.55s)
=== RUN   TestLayerMixer_SumsLayers
--- PASS: TestLayerMixer_SumsLayers (0.00s)
=== RUN   TestLayerMixer_ClampsOutput
--- PASS: TestLayerMixer_ClampsOutput (0.00s)
=== RUN   TestLayerMixer_ClampsNegative
--- PASS: TestLayerMixer_ClampsNegative (0.00s)
=== RUN   TestLayerMixer_ZeroLayers
--- PASS: TestLayerMixer_ZeroLayers (0.00s)
=== RUN   TestOscillator_Sin_FrequencyAccuracy
--- PASS: TestOscillator_Sin_FrequencyAccuracy (0.00s)
=== RUN   TestOscillator_Sin_AmplitudeBounds
--- PASS: TestOscillator_Sin_AmplitudeBounds (0.00s)
=== RUN   TestOscillator_Sin_PhaseContinuity
--- PASS: TestOscillator_Sin_PhaseContinuity (0.01s)
=== RUN   TestOscillator_Saw_AmplitudeBounds
--- PASS: TestOscillator_Saw_AmplitudeBounds (0.00s)
=== RUN   TestOscillator_Saw_FrequencyAccuracy
--- PASS: TestOscillator_Saw_FrequencyAccuracy (0.00s)
=== RUN   TestOscillator_Reset
--- PASS: TestOscillator_Reset (0.00s)
=== RUN   TestOscillator_Sin_KnownValues
--- PASS: TestOscillator_Sin_KnownValues (0.00s)
PASS
ok  github.com/JamesPrial/go-scream/internal/audio/native  127.190s
```

---

### Race Detection

Command: `go test -race ./internal/audio/... ./internal/audio/native/...`

```
ok  github.com/JamesPrial/go-scream/internal/audio         1.479s
ok  github.com/JamesPrial/go-scream/internal/audio/native  160.390s
```

No races detected.

---

### Static Analysis

Command: `go vet ./...`

No output. No warnings. Exit status 0.

---

### Coverage Details

Command: `go test -cover ./internal/audio/... ./internal/audio/native/...`

```
ok  github.com/JamesPrial/go-scream/internal/audio         0.714s   coverage: 90.0% of statements
ok  github.com/JamesPrial/go-scream/internal/audio/native  126.101s coverage: 100.0% of statements
```

Both packages exceed the 70% threshold.

---

### Linter Output

Command: `golangci-lint run`

```
internal/audio/native/generator_test.go:251:14: Error return value of `binary.Read` is not checked (errcheck)
        binary.Read(buf, binary.LittleEndian, &sample)
                   ^
internal/audio/native/generator_test.go:278:10: Error return value of `io.Copy` is not checked (errcheck)
        io.Copy(io.Discard, reader)
               ^
2 issues:
* errcheck: 2
```

Both warnings are in **test files only** (`generator_test.go`), not in production code. These are low-severity style warnings:
- Line 251: `binary.Read` into a local variable during a range-check scan (in `TestNativeGenerator_S16LERange`)
- Line 278: `io.Copy` to `io.Discard` during a drain operation (in `TestNativeGenerator_Deterministic` or similar)

Neither indicates a logic flaw; both calls cannot meaningfully fail in their test context. Non-blocking.

---

### Issues to Address

None. All production code passes cleanly. The 2 linter findings are confined to test helper code and do not affect correctness. If desired, they can be silenced with `//nolint:errcheck` annotations or by capturing and asserting the returned errors.

---

### Final Verdict

**TESTS_PASS**

- **65 tests** across 2 packages, all passing
- **Coverage:** 90.0% (`internal/audio`) | 100.0% (`internal/audio/native`)
- **Race conditions:** None
- **Vet warnings:** None
- **Linter:** 2 non-critical `errcheck` warnings in test files only — no production code issues
