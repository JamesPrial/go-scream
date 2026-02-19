## Test Execution Report — Stage 5a (Configuration + Version)

### Summary
- **Verdict:** TESTS_PASS
- **Tests Run:** 75 passed, 0 failed (config: 62 tests + 3 benchmarks; version: 10 tests + 1 benchmark)
- **Coverage:** internal/config: 96.5% | pkg/version: 100.0%
- **Race Conditions:** None
- **Vet Warnings:** None
- **Linter:** 0 issues (golangci-lint)

---

### Test Results

```
=== RUN   TestDefault_ReturnsExpectedValues
--- PASS: TestDefault_ReturnsExpectedValues (0.00s)
=== RUN   TestDefault_BackendConstant
--- PASS: TestDefault_BackendConstant (0.00s)
=== RUN   TestDefault_FormatConstant
--- PASS: TestDefault_FormatConstant (0.00s)
=== RUN   TestMerge_ZeroOverlayPreservesBase
--- PASS: TestMerge_ZeroOverlayPreservesBase (0.00s)
=== RUN   TestMerge_NonZeroOverlayWins
--- PASS: TestMerge_NonZeroOverlayWins (0.00s)
=== RUN   TestMerge_PartialOverlay
--- PASS: TestMerge_PartialOverlay (0.00s)
=== RUN   TestMerge_FieldTypes
--- PASS: TestMerge_FieldTypes (0.00s)
    --- PASS: TestMerge_FieldTypes/string_field:_Token_override (0.00s)
    --- PASS: TestMerge_FieldTypes/BackendType_field_override (0.00s)
    --- PASS: TestMerge_FieldTypes/FormatType_field_override (0.00s)
    --- PASS: TestMerge_FieldTypes/Duration_field_override (0.00s)
    --- PASS: TestMerge_FieldTypes/float64_field:_Volume_override (0.00s)
    --- PASS: TestMerge_FieldTypes/bool_field:_DryRun_override_true (0.00s)
    --- PASS: TestMerge_FieldTypes/bool_field:_false_overlay_preserves_base_true (0.00s)
=== RUN   TestMerge_BothZero
--- PASS: TestMerge_BothZero (0.00s)
=== RUN   TestMerge_DoesNotMutateInputs
--- PASS: TestMerge_DoesNotMutateInputs (0.00s)
=== RUN   TestLoad_ValidYAML
--- PASS: TestLoad_ValidYAML (0.00s)
=== RUN   TestLoad_PartialYAML
--- PASS: TestLoad_PartialYAML (0.00s)
=== RUN   TestLoad_NonexistentFile
--- PASS: TestLoad_NonexistentFile (0.00s)
=== RUN   TestLoad_InvalidYAML
--- PASS: TestLoad_InvalidYAML (0.00s)
=== RUN   TestLoad_EmptyFile
--- PASS: TestLoad_EmptyFile (0.00s)
=== RUN   TestLoad_UnknownFieldsSilentlyIgnored
--- PASS: TestLoad_UnknownFieldsSilentlyIgnored (0.00s)
=== RUN   TestLoad_DurationFormats
--- PASS: TestLoad_DurationFormats (0.00s)
    --- PASS: TestLoad_DurationFormats/Go_duration_string_3s (0.00s)
    --- PASS: TestLoad_DurationFormats/Go_duration_string_500ms (0.00s)
    --- PASS: TestLoad_DurationFormats/Go_duration_string_1m30s (0.00s)
=== RUN   TestApplyEnv_AllVariables
--- PASS: TestApplyEnv_AllVariables (0.00s)
=== RUN   TestApplyEnv_EmptyEnvVarsUnset
--- PASS: TestApplyEnv_EmptyEnvVarsUnset (0.00s)
=== RUN   TestApplyEnv_InvalidDurationSilentlyIgnored
--- PASS: TestApplyEnv_InvalidDurationSilentlyIgnored (0.00s)
=== RUN   TestApplyEnv_InvalidVolumeSilentlyIgnored
--- PASS: TestApplyEnv_InvalidVolumeSilentlyIgnored (0.00s)
=== RUN   TestApplyEnv_InvalidVerboseSilentlyIgnored
--- PASS: TestApplyEnv_InvalidVerboseSilentlyIgnored (0.00s)
=== RUN   TestApplyEnv_OverridesExistingValues
--- PASS: TestApplyEnv_OverridesExistingValues (0.00s)
=== RUN   TestApplyEnv_IndividualVariables
--- PASS: TestApplyEnv_IndividualVariables (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/DISCORD_TOKEN (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/SCREAM_GUILD_ID (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/SCREAM_BACKEND_native (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/SCREAM_PRESET (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/SCREAM_DURATION (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/SCREAM_VOLUME (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/SCREAM_FORMAT_ogg (0.00s)
    --- PASS: TestApplyEnv_IndividualVariables/SCREAM_VERBOSE_true (0.00s)
=== RUN   TestApplyEnv_VerboseVariants
--- PASS: TestApplyEnv_VerboseVariants (0.00s)
    --- PASS: TestApplyEnv_VerboseVariants/true (0.00s)
    --- PASS: TestApplyEnv_VerboseVariants/1 (0.00s)
    --- PASS: TestApplyEnv_VerboseVariants/TRUE (0.00s)
    --- PASS: TestApplyEnv_VerboseVariants/True (0.00s)
    --- PASS: TestApplyEnv_VerboseVariants/false (0.00s)
    --- PASS: TestApplyEnv_VerboseVariants/0 (0.00s)
    --- PASS: TestApplyEnv_VerboseVariants/FALSE (0.00s)
=== RUN   TestValidate_DefaultConfigPasses
--- PASS: TestValidate_DefaultConfigPasses (0.00s)
=== RUN   TestValidate_Backend
--- PASS: TestValidate_Backend (0.00s)
    --- PASS: TestValidate_Backend/native_is_valid (0.00s)
    --- PASS: TestValidate_Backend/ffmpeg_is_valid (0.00s)
    --- PASS: TestValidate_Backend/empty_is_invalid (0.00s)
    --- PASS: TestValidate_Backend/unknown_backend_is_invalid (0.00s)
    --- PASS: TestValidate_Backend/case_sensitive:_Native_is_invalid (0.00s)
    --- PASS: TestValidate_Backend/case_sensitive:_FFMPEG_is_invalid (0.00s)
=== RUN   TestValidate_Preset
--- PASS: TestValidate_Preset (0.00s)
    --- PASS: TestValidate_Preset/classic_is_valid (0.00s)
    --- PASS: TestValidate_Preset/whisper_is_valid (0.00s)
    --- PASS: TestValidate_Preset/death-metal_is_valid (0.00s)
    --- PASS: TestValidate_Preset/glitch_is_valid (0.00s)
    --- PASS: TestValidate_Preset/banshee_is_valid (0.00s)
    --- PASS: TestValidate_Preset/robot_is_valid (0.00s)
    --- PASS: TestValidate_Preset/empty_is_valid_(no_preset_selected) (0.00s)
    --- PASS: TestValidate_Preset/unknown_preset_is_invalid (0.00s)
    --- PASS: TestValidate_Preset/case_sensitive:_Classic_is_invalid (0.00s)
=== RUN   TestValidate_Duration
--- PASS: TestValidate_Duration (0.00s)
    --- PASS: TestValidate_Duration/positive_duration_is_valid (0.00s)
    --- PASS: TestValidate_Duration/1ms_is_valid (0.00s)
    --- PASS: TestValidate_Duration/zero_duration_is_invalid (0.00s)
    --- PASS: TestValidate_Duration/negative_duration_is_invalid (0.00s)
=== RUN   TestValidate_Volume
--- PASS: TestValidate_Volume (0.00s)
    --- PASS: TestValidate_Volume/volume_1.0_is_valid (0.00s)
    --- PASS: TestValidate_Volume/volume_0.0_is_valid (0.00s)
    --- PASS: TestValidate_Volume/volume_0.5_is_valid (0.00s)
    --- PASS: TestValidate_Volume/volume_above_1.0_is_invalid (0.00s)
    --- PASS: TestValidate_Volume/volume_below_0.0_is_invalid (0.00s)
    --- PASS: TestValidate_Volume/volume_way_above_range (0.00s)
=== RUN   TestValidate_Format
--- PASS: TestValidate_Format (0.00s)
    --- PASS: TestValidate_Format/ogg_is_valid (0.00s)
    --- PASS: TestValidate_Format/wav_is_valid (0.00s)
    --- PASS: TestValidate_Format/empty_format_is_invalid (0.00s)
    --- PASS: TestValidate_Format/mp3_is_invalid (0.00s)
    --- PASS: TestValidate_Format/case_sensitive:_OGG_is_invalid (0.00s)
    --- PASS: TestValidate_Format/case_sensitive:_WAV_is_invalid (0.00s)
=== RUN   TestValidate_MultipleInvalidFields
--- PASS: TestValidate_MultipleInvalidFields (0.00s)
=== RUN   TestValidate_SentinelErrorsExist
--- PASS: TestValidate_SentinelErrorsExist (0.00s)
    --- PASS: TestValidate_SentinelErrorsExist/ErrConfigNotFound (0.00s)
    --- PASS: TestValidate_SentinelErrorsExist/ErrConfigParse (0.00s)
    --- PASS: TestValidate_SentinelErrorsExist/ErrInvalidBackend (0.00s)
    --- PASS: TestValidate_SentinelErrorsExist/ErrInvalidPreset (0.00s)
    --- PASS: TestValidate_SentinelErrorsExist/ErrInvalidDuration (0.00s)
    --- PASS: TestValidate_SentinelErrorsExist/ErrInvalidVolume (0.00s)
    --- PASS: TestValidate_SentinelErrorsExist/ErrInvalidFormat (0.00s)
=== RUN   TestValidate_ErrorWrapping
--- PASS: TestValidate_ErrorWrapping (0.00s)
PASS
ok  github.com/JamesPrial/go-scream/internal/config   0.261s

=== RUN   TestDefaultVersion
--- PASS: TestDefaultVersion (0.00s)
=== RUN   TestDefaultCommit
--- PASS: TestDefaultCommit (0.00s)
=== RUN   TestDefaultDate
--- PASS: TestDefaultDate (0.00s)
=== RUN   TestString_DefaultValues
--- PASS: TestString_DefaultValues (0.00s)
=== RUN   TestString_Format
--- PASS: TestString_Format (0.00s)
    --- PASS: TestString_Format/default_values (0.00s)
    --- PASS: TestString_Format/release_version (0.00s)
    --- PASS: TestString_Format/prerelease_version (0.00s)
    --- PASS: TestString_Format/empty_strings (0.00s)
=== RUN   TestString_MatchesExpectedFormat
--- PASS: TestString_MatchesExpectedFormat (0.00s)
PASS
ok  github.com/JamesPrial/go-scream/pkg/version   0.454s
```

---

### Race Detection

```
ok  github.com/JamesPrial/go-scream/internal/config   1.573s
ok  github.com/JamesPrial/go-scream/pkg/version       1.325s
```

No race conditions detected.

---

### Static Analysis

```
(no output)
```

`go vet` produced no warnings.

---

### Coverage Details

```
ok  github.com/JamesPrial/go-scream/internal/config   0.295s  coverage: 96.5% of statements
ok  github.com/JamesPrial/go-scream/pkg/version       0.533s  coverage: 100.0% of statements
```

Both packages exceed the 70% threshold.

---

### Linter Output

```
0 issues.
```

golangci-lint reported no issues.

---

### Regression Tests (audio / encoding / ffmpeg / discord)

```
ok  github.com/JamesPrial/go-scream/internal/audio          0.267s
ok  github.com/JamesPrial/go-scream/internal/audio/ffmpeg   0.508s
ok  github.com/JamesPrial/go-scream/internal/audio/native   1.157s
ok  github.com/JamesPrial/go-scream/internal/encoding       0.471s
ok  github.com/JamesPrial/go-scream/internal/discord        0.756s
```

All regression packages continue to pass — no regressions introduced.

---

### TESTS_PASS

All checks pass, coverage meets threshold.

- **Total tests:** 75 passed, 0 failed (across internal/config and pkg/version)
- **Coverage:** internal/config 96.5% | pkg/version 100.0%
- **Race conditions:** None
- **Vet warnings:** None
- **Linter issues:** None
- **Regressions:** None (audio, encoding, ffmpeg, discord all green)
