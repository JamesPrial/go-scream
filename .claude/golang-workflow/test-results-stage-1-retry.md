## Test Execution Report — Stage 1 Retry

### Summary
- **Verdict:** TESTS_PASS
- **Tests Run:** 62 passed, 0 failed
- **Coverage:** internal/audio: 87.5% | internal/audio/native: 100.0%
- **Race Conditions:** None
- **Vet Warnings:** None
- **Linter:** 2 non-critical errcheck warnings in test/benchmark code only (not implementation)

---

### Test Results

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
=== RUN   TestValidate_InvalidLimiterLevel/zero
=== RUN   TestValidate_InvalidLimiterLevel/negative
=== RUN   TestValidate_InvalidLimiterLevel/above_one
--- PASS: TestValidate_InvalidLimiterLevel (0.00s)
    --- PASS: TestValidate_InvalidLimiterLevel/zero (0.00s)
    --- PASS: TestValidate_InvalidLimiterLevel/negative (0.00s)
    --- PASS: TestValidate_InvalidLimiterLevel/above_one (0.00s)
=== RUN   TestAllPresets_ReturnsAll6
--- PASS: TestAllPresets_ReturnsAll6 (0.00s)
=== RUN   TestGetPreset_AllPresetsValid
=== RUN   TestGetPreset_AllPresetsValid/classic
=== RUN   TestGetPreset_AllPresetsValid/whisper
=== RUN   TestGetPreset_AllPresetsValid/death-metal
=== RUN   TestGetPreset_AllPresetsValid/glitch
=== RUN   TestGetPreset_AllPresetsValid/banshee
=== RUN   TestGetPreset_AllPresetsValid/robot
--- PASS: TestGetPreset_AllPresetsValid (0.00s)
=== RUN   TestGetPreset_Unknown
--- PASS: TestGetPreset_Unknown (0.00s)
=== RUN   TestGetPreset_ParameterRanges
--- PASS: TestGetPreset_ParameterRanges (0.00s)
PASS
ok  github.com/JamesPrial/go-scream/internal/audio   0.501s

=== RUN   TestHighpassFilter_RemovesDC
--- PASS: TestHighpassFilter_RemovesDC (0.00s)
=== RUN   TestHighpassFilter_PassesHighFreq
--- PASS: TestHighpassFilter_PassesHighFreq (0.00s)
=== RUN   TestLowpassFilter_PassesDC
--- PASS: TestLowpassFilter_PassesDC (0.00s)
=== RUN   TestLowpassFilter_AttenuatesHighFreq
--- PASS: TestLowpassFilter_AttenuatesHighFreq (0.00s)
=== RUN   TestBitcrusher_FullMix
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
--- PASS: TestFilterChainFromParams_ClassicPreset (0.01s)
=== RUN   TestHighpassFilter_ImplementsFilter ... (all interface checks pass)
=== RUN   TestNativeGenerator_CorrectByteCount
--- PASS: TestNativeGenerator_CorrectByteCount (0.04s)
=== RUN   TestNativeGenerator_NonSilent
--- PASS: TestNativeGenerator_NonSilent (0.03s)
=== RUN   TestNativeGenerator_Deterministic
--- PASS: TestNativeGenerator_Deterministic (0.05s)
=== RUN   TestNativeGenerator_DifferentSeeds
--- PASS: TestNativeGenerator_DifferentSeeds (0.05s)
=== RUN   TestNativeGenerator_AllPresets
--- PASS: TestNativeGenerator_AllPresets (0.14s)
=== RUN   TestNativeGenerator_InvalidParams
--- PASS: TestNativeGenerator_InvalidParams (0.00s)
=== RUN   TestNativeGenerator_MonoOutput
--- PASS: TestNativeGenerator_MonoOutput (0.02s)
=== RUN   TestNativeGenerator_S16LERange
--- PASS: TestNativeGenerator_S16LERange (0.03s)
=== RUN   TestNativeGenerator_ImplementsInterface
--- PASS: TestNativeGenerator_ImplementsInterface (0.00s)
=== RUN   TestPrimaryScreamLayer_NonZeroOutput
--- PASS: TestPrimaryScreamLayer_NonZeroOutput (0.00s)
=== RUN   TestPrimaryScreamLayer_AmplitudeBounds
--- PASS: TestPrimaryScreamLayer_AmplitudeBounds (0.00s)
=== RUN   TestHarmonicSweepLayer_NonZeroOutput
--- PASS: TestHarmonicSweepLayer_NonZeroOutput (0.00s)
=== RUN   TestHighShriekLayer_NonZeroOutput
--- PASS: TestHighShriekLayer_NonZeroOutput (0.00s)
=== RUN   TestHighShriekLayer_EnvelopeRises
--- PASS: TestHighShriekLayer_EnvelopeRises (0.00s)
=== RUN   TestNoiseBurstLayer_HasSilentAndActiveSegments
--- PASS: TestNoiseBurstLayer_HasSilentAndActiveSegments (0.00s)
=== RUN   TestBackgroundNoiseLayer_ContinuousOutput
--- PASS: TestBackgroundNoiseLayer_ContinuousOutput (0.00s)
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
ok  github.com/JamesPrial/go-scream/internal/audio/native   1.102s
```

---

### Race Detection

```
ok  github.com/JamesPrial/go-scream/internal/audio         1.296s
ok  github.com/JamesPrial/go-scream/internal/audio/native  4.933s
```

No races detected.

---

### Static Analysis

```
(no output — zero warnings)
```

`go vet ./...` exited with status 0 and produced no warnings.

---

### Coverage Details

```
ok  github.com/JamesPrial/go-scream/internal/audio         0.350s  coverage: 87.5% of statements
ok  github.com/JamesPrial/go-scream/internal/audio/native  1.012s  coverage: 100.0% of statements
```

Both packages exceed the 70% threshold.

---

### Linter Output

`golangci-lint run` found 2 non-critical `errcheck` warnings, both located in test/benchmark code in `/Users/jamesprial/code/go-scream/internal/audio/native/generator_test.go`:

```
internal/audio/native/generator_test.go:251:14: Error return value of `binary.Read` is not checked (errcheck)
        binary.Read(buf, binary.LittleEndian, &sample)
                   ^
internal/audio/native/generator_test.go:278:10: Error return value of `io.Copy` is not checked (errcheck)
        io.Copy(io.Discard, reader)
               ^
```

- Line 251: `binary.Read` inside a loop that is reading already-validated data — the prior test assertion confirmed the buffer length is correct.
- Line 278: `io.Copy` draining a reader into `io.Discard` inside a benchmark loop — drain-to-discard errors are not actionable in a benchmark context.

Neither warning is in production code. No critical linter errors.

---

### Issues to Address

None. All pass criteria are satisfied:

- [x] All `go test` commands exit with status 0
- [x] No race conditions detected by `-race`
- [x] No warnings from `go vet`
- [x] Coverage exceeds 70% threshold (87.5% and 100.0%)
- [x] No critical linter errors (2 non-critical test-only errcheck warnings)
