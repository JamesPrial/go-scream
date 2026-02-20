# Refactor Baseline Snapshot

**Date:** 2026-02-19
**Commit:** 008595f Initial commit: go-scream audio generation tool
**Branch:** main

---

## VERDICT: BASELINE_CLEAN

All tests pass. No race conditions. No vet warnings. No lint issues.
The only note is `internal/discord` coverage is 64.1% (pre-existing, not a failure).

---

## Summary

| Check | Result |
|-------|--------|
| `go test ./...` | ALL PASS |
| `go test -race ./...` | NO RACES (3 linker warnings: macOS `ld` LC_DYSYMTAB — pre-existing OS-level warning, not a Go issue) |
| `go vet ./...` | NO WARNINGS |
| `go test -cover ./...` | See coverage table below |
| `golangci-lint run` | 0 issues |

---

## Test Results — Full List

### Package: `github.com/JamesPrial/go-scream/cmd/scream`
- No test files.

### Package: `github.com/JamesPrial/go-scream/cmd/skill`
- PASS: `Test_parseOpenClawConfig_Cases` (subtests: valid_JSON_with_token, missing_file_returns_error, invalid_JSON_returns_error, missing_token_field_returns_empty_string, empty_channels_returns_empty_string, empty_object_returns_empty_string)
- PASS: `Test_parseOpenClawConfig_ValidJSON_StructureVerification`
- PASS: `Test_parseOpenClawConfig_ExtraFieldsIgnored`
- PASS: `Test_parseOpenClawConfig_EmptyToken`
- PASS: `Test_parseOpenClawConfig_EmptyFile`
- PASS: `Test_parseOpenClawConfig_NullValues` (subtests: null_channels, null_discord, null_token)
- PASS: `Test_resolveToken_Cases` (subtests: env_var_takes_priority_over_file, env_empty_falls_back_to_file, env_empty_and_no_file_returns_empty, env_set_and_no_file_returns_env_token)
- PASS: `Test_resolveToken_EnvPriority`
- PASS: `Test_resolveToken_FallbackToFile`
- PASS: `Test_resolveToken_NoSources`
- PASS: `Test_resolveToken_InvalidJSON_FallsGracefully`
- PASS: `Test_resolveToken_EnvOverridesInvalidFile`
- PASS: `Test_resolveToken_FileWithEmptyToken`

### Package: `github.com/JamesPrial/go-scream/internal/audio`
- PASS: `TestRandomize_ProducesValidParams`
- PASS: `TestRandomize_Deterministic`
- PASS: `TestRandomize_DifferentSeeds`
- PASS: `TestRandomize_ZeroSeed`
- PASS: `TestValidate_ValidParams`
- PASS: `TestValidate_InvalidDuration`
- PASS: `TestValidate_InvalidSampleRate`
- PASS: `TestValidate_InvalidChannels`
- PASS: `TestValidate_InvalidAmplitude`
- PASS: `TestValidate_InvalidLimiterLevel` (subtests: zero, negative, above_one)
- PASS: `TestAllPresets_ReturnsAll6`
- PASS: `TestGetPreset_AllPresetsValid` (subtests: classic, whisper, death-metal, glitch, banshee, robot)
- PASS: `TestGetPreset_Unknown`
- PASS: `TestGetPreset_ParameterRanges` (subtests: classic, whisper, death-metal, glitch, banshee, robot)

### Package: `github.com/JamesPrial/go-scream/internal/audio/ffmpeg`
- PASS: `Test_BuildArgs_ContainsLavfiInput`
- PASS: `Test_BuildArgs_ContainsAevalsrc`
- PASS: `Test_BuildArgs_ContainsAudioFilter`
- PASS: `Test_BuildArgs_ContainsOutputFormat`
- PASS: `Test_BuildArgs_ContainsChannels`
- PASS: `Test_BuildArgs_ContainsSampleRate`
- PASS: `Test_BuildArgs_LastArgIsPipe`
- PASS: `Test_BuildArgs_ContainsDuration`
- PASS: `Test_BuildArgs_MonoParams`
- PASS: `Test_BuildArgs_DifferentSampleRate`
- PASS: `Test_buildAevalsrcExpr_ContainsSin`
- PASS: `Test_buildAevalsrcExpr_ContainsRandom`
- PASS: `Test_buildAevalsrcExpr_ContainsPI`
- PASS: `Test_buildAevalsrcExpr_NonEmptyForAllPresets` (subtests: seed_1, seed_42, seed_100, seed_9999, seed_12345, seed_99999)
- PASS: `Test_buildAevalsrcExpr_ZeroAmplitudeLayer`
- PASS: `Test_buildFilterChain_ContainsHighpass`
- PASS: `Test_buildFilterChain_ContainsLowpass`
- PASS: `Test_buildFilterChain_ContainsAcrusher`
- PASS: `Test_buildFilterChain_ContainsAcompressor`
- PASS: `Test_buildFilterChain_ContainsVolume`
- PASS: `Test_buildFilterChain_ContainsAlimiter`
- PASS: `Test_buildFilterChain_FilterOrder`
- PASS: `Test_layerExpr_PrimaryScream`
- PASS: `Test_layerExpr_HarmonicSweep`
- PASS: `Test_layerExpr_HighShriek`
- PASS: `Test_layerExpr_NoiseBurst`
- PASS: `Test_layerExpr_BackgroundNoise`
- PASS: `Test_layerExpr_ZeroAmplitude`
- PASS: `Test_fmtFloat_Cases` (subtests: integer_value, fractional_value, small_value, negative_value, zero, large_value)
- PASS: `Test_fmtFloat_ConsistentPrecision`
- PASS: `Test_fmtFloat_NegativeValue`
- PASS: `Test_deriveSeed_DifferentIndexes`
- PASS: `Test_deriveSeed_DifferentGlobalSeeds`
- PASS: `Test_deriveSeed_Deterministic`
- PASS: `Test_deriveSeed_NonNegative` (subtests: positive_seeds, zero_global, zero_layer, large_seeds, negative_global, negative_layer, both_negative)
- PASS: `Test_deriveSeed_DifferentLayerSeeds`
- PASS: `Test_BuildArgs_AllPresets` (subtests: classic, whisper, death-metal, glitch, banshee, robot)
- PASS: `Test_BuildArgs_WithRandomizedParams` (subtests: seed_1, seed_42, seed_100, seed_9999, seed_12345)
- SKIP: `TestNewFFmpegGenerator_Success` — ffmpeg not available on PATH
- PASS: `TestNewFFmpegGeneratorWithPath_NotNil`
- PASS: `TestNewFFmpegGenerator_NoFFmpegOnPath`
- SKIP: `TestFFmpegGenerator_CorrectByteCount` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_NonSilent` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_AllPresets` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_AllNamedPresets` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_InvalidDuration` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_NegativeDuration` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_InvalidSampleRate` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_NegativeSampleRate` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_InvalidChannels` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_ZeroChannels` — ffmpeg not available on PATH
- PASS: `TestFFmpegGenerator_BadBinaryPath`
- SKIP: `TestFFmpegGenerator_InvalidAmplitude` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_InvalidCrusherBits` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_InvalidLimiterLevel` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_EvenByteCount` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_StereoAligned` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_MonoOutput` — ffmpeg not available on PATH
- SKIP: `TestFFmpegGenerator_DeterministicOutput` — ffmpeg not available on PATH

### Package: `github.com/JamesPrial/go-scream/internal/audio/native`
- PASS: `TestHighpassFilter_RemovesDC`
- PASS: `TestHighpassFilter_PassesHighFreq`
- PASS: `TestLowpassFilter_PassesDC`
- PASS: `TestLowpassFilter_AttenuatesHighFreq`
- PASS: `TestBitcrusher_FullMix` (subtests: positive, negative, zero, one)
- PASS: `TestBitcrusher_ZeroMix`
- PASS: `TestBitcrusher_Blend`
- PASS: `TestCompressor_BelowThreshold`
- PASS: `TestCompressor_AboveThreshold`
- PASS: `TestCompressor_PreservesSign`
- PASS: `TestVolumeBoost_ZeroDB`
- PASS: `TestVolumeBoost_6dB`
- PASS: `TestVolumeBoost_NegativeDB`
- PASS: `TestLimiter_WithinRange`
- PASS: `TestLimiter_ClipsPositive`
- PASS: `TestLimiter_ClipsNegative`
- PASS: `TestFilterChain_OrderMatters`
- PASS: `TestFilterChainFromParams_ClassicPreset`
- PASS: `TestHighpassFilter_ImplementsFilter`
- PASS: `TestLowpassFilter_ImplementsFilter`
- PASS: `TestBitcrusher_ImplementsFilter`
- PASS: `TestCompressor_ImplementsFilter`
- PASS: `TestVolumeBoost_ImplementsFilter`
- PASS: `TestLimiter_ImplementsFilter`
- PASS: `TestFilterChain_ImplementsFilter`
- PASS: `TestNativeGenerator_CorrectByteCount`
- PASS: `TestNativeGenerator_NonSilent`
- PASS: `TestNativeGenerator_Deterministic`
- PASS: `TestNativeGenerator_DifferentSeeds`
- PASS: `TestNativeGenerator_AllPresets` (subtests: classic, whisper, death-metal, glitch, banshee, robot)
- PASS: `TestNativeGenerator_InvalidParams`
- PASS: `TestNativeGenerator_MonoOutput`
- PASS: `TestNativeGenerator_S16LERange`
- PASS: `TestNativeGenerator_ImplementsInterface`
- PASS: `TestPrimaryScreamLayer_NonZeroOutput`
- PASS: `TestPrimaryScreamLayer_AmplitudeBounds`
- PASS: `TestHarmonicSweepLayer_NonZeroOutput`
- PASS: `TestHighShriekLayer_NonZeroOutput`
- PASS: `TestHighShriekLayer_EnvelopeRises`
- PASS: `TestNoiseBurstLayer_HasSilentAndActiveSegments`
- PASS: `TestBackgroundNoiseLayer_ContinuousOutput`
- PASS: `TestLayerMixer_SumsLayers`
- PASS: `TestLayerMixer_ClampsOutput`
- PASS: `TestLayerMixer_ClampsNegative`
- PASS: `TestLayerMixer_ZeroLayers`
- PASS: `TestOscillator_Sin_FrequencyAccuracy`
- PASS: `TestOscillator_Sin_AmplitudeBounds`
- PASS: `TestOscillator_Sin_PhaseContinuity`
- PASS: `TestOscillator_Saw_AmplitudeBounds`
- PASS: `TestOscillator_Saw_FrequencyAccuracy`
- PASS: `TestOscillator_Reset`
- PASS: `TestOscillator_Sin_KnownValues`

### Package: `github.com/JamesPrial/go-scream/internal/config`
- PASS: `TestDefault_ReturnsExpectedValues`
- PASS: `TestDefault_BackendConstant`
- PASS: `TestDefault_FormatConstant`
- PASS: `TestMerge_ZeroOverlayPreservesBase`
- PASS: `TestMerge_NonZeroOverlayWins`
- PASS: `TestMerge_PartialOverlay`
- PASS: `TestMerge_FieldTypes` (subtests: string_field:_Token_override, BackendType_field_override, FormatType_field_override, Duration_field_override, float64_field:_Volume_override, bool_field:_DryRun_override_true, bool_field:_false_overlay_preserves_base_true)
- PASS: `TestMerge_BothZero`
- PASS: `TestMerge_DoesNotMutateInputs`
- PASS: `TestLoad_ValidYAML`
- PASS: `TestLoad_PartialYAML`
- PASS: `TestLoad_NonexistentFile`
- PASS: `TestLoad_InvalidYAML`
- PASS: `TestLoad_EmptyFile`
- PASS: `TestLoad_UnknownFieldsSilentlyIgnored`
- PASS: `TestLoad_DurationFormats` (subtests: Go_duration_string_3s, Go_duration_string_500ms, Go_duration_string_1m30s, invalid_duration_string, bare_integer_without_unit)
- PASS: `TestApplyEnv_AllVariables`
- PASS: `TestApplyEnv_EmptyEnvVarsUnset`
- PASS: `TestApplyEnv_InvalidDurationSilentlyIgnored`
- PASS: `TestApplyEnv_InvalidVolumeSilentlyIgnored`
- PASS: `TestApplyEnv_InvalidVerboseSilentlyIgnored`
- PASS: `TestApplyEnv_OverridesExistingValues`
- PASS: `TestApplyEnv_IndividualVariables` (subtests: DISCORD_TOKEN, SCREAM_GUILD_ID, SCREAM_BACKEND_native, SCREAM_PRESET, SCREAM_DURATION, SCREAM_VOLUME, SCREAM_FORMAT_ogg, SCREAM_VERBOSE_true)
- PASS: `TestApplyEnv_VerboseVariants` (subtests: true, 1, TRUE, True, false, 0, FALSE)
- PASS: `TestValidate_DefaultConfigPasses`
- PASS: `TestValidate_Backend` (subtests: native_is_valid, ffmpeg_is_valid, empty_is_invalid, unknown_backend_is_invalid, case_sensitive:_Native_is_invalid, case_sensitive:_FFMPEG_is_invalid)
- PASS: `TestValidate_Preset` (subtests: classic_is_valid, whisper_is_valid, death-metal_is_valid, glitch_is_valid, banshee_is_valid, robot_is_valid, empty_is_valid_(no_preset_selected), unknown_preset_is_invalid, case_sensitive:_Classic_is_invalid)
- PASS: `TestValidate_Duration` (subtests: positive_duration_is_valid, 1ms_is_valid, zero_duration_is_invalid, negative_duration_is_invalid)
- PASS: `TestValidate_Volume` (subtests: volume_1.0_is_valid, volume_0.0_is_valid, volume_0.5_is_valid, volume_above_1.0_is_invalid, volume_below_0.0_is_invalid, volume_way_above_range)
- PASS: `TestValidate_Format` (subtests: ogg_is_valid, wav_is_valid, empty_format_is_invalid, mp3_is_invalid, case_sensitive:_OGG_is_invalid, case_sensitive:_WAV_is_invalid)
- PASS: `TestValidate_MultipleInvalidFields`
- PASS: `TestValidate_SentinelErrorsExist` (subtests: ErrConfigNotFound, ErrConfigParse, ErrInvalidBackend, ErrInvalidPreset, ErrInvalidDuration, ErrInvalidVolume, ErrInvalidFormat)
- PASS: `TestValidate_ErrorWrapping`

### Package: `github.com/JamesPrial/go-scream/internal/discord`
- PASS: `TestFindPopulatedChannel_OneUser`
- PASS: `TestFindPopulatedChannel_MultipleUsers`
- PASS: `TestFindPopulatedChannel_MultipleChannels`
- PASS: `TestFindPopulatedChannel_OnlyBot`
- PASS: `TestFindPopulatedChannel_BotAndUser`
- PASS: `TestFindPopulatedChannel_BotInOneUserInAnother`
- PASS: `TestFindPopulatedChannel_Empty`
- PASS: `TestFindPopulatedChannel_EmptyGuildID`
- PASS: `TestFindPopulatedChannel_StateError`
- PASS: `TestFindPopulatedChannel_Cases` (subtests: single_non-bot_user_returns_their_channel, bot_user_excluded_second_user_returned, no_voice_states_returns_ErrNoPopulatedChannel, empty_guild_ID_returns_ErrEmptyGuildID, state_retrieval_error_returns_ErrGuildStateFailed)
- PASS: `TestNewDiscordPlayer_NotNil`
- PASS: `TestSilenceFrame_Content`
- PASS: `TestSilenceFrameCount_Value`
- PASS: `TestDiscordPlayer_Play_10Frames`
- PASS: `TestDiscordPlayer_Play_EmptyChannel`
- PASS: `TestDiscordPlayer_Play_1Frame`
- PASS: `TestDiscordPlayer_Play_SpeakingProtocol`
- PASS: `TestDiscordPlayer_Play_SilenceFrames`
- PASS: `TestDiscordPlayer_Play_DisconnectCalled`
- PASS: `TestDiscordPlayer_Play_DisconnectOnError`
- PASS: `TestDiscordPlayer_Play_JoinParams`
- PASS: `TestDiscordPlayer_Play_EmptyGuildID`
- PASS: `TestDiscordPlayer_Play_EmptyChannelID`
- PASS: `TestDiscordPlayer_Play_NilFrames`
- PASS: `TestDiscordPlayer_Play_JoinFails`
- PASS: `TestDiscordPlayer_Play_SpeakingTrueFails`
- PASS: `TestDiscordPlayer_Play_CancelledContext`
- PASS: `TestDiscordPlayer_Play_CancelMidPlayback`
- PASS: `TestDiscordPlayer_Play_ValidationErrors` (subtests: empty_guild_ID, empty_channel_ID, nil_frame_channel)

### Package: `github.com/JamesPrial/go-scream/internal/encoding`
- PASS: `TestConstants` (subtests: OpusFrameSamples, MaxOpusFrameBytes, OpusBitrate)
- PASS: `TestPcmBytesToInt16_KnownValues` (subtests: zero, positive_one, negative_one, 256_(0x0100), multiple_samples, little-endian_0x0201_=_513, 0x80FF_=_-32513_in_signed)
- PASS: `TestPcmBytesToInt16_EmptyInput`
- PASS: `TestPcmBytesToInt16_RoundTrip`
- PASS: `TestPcmBytesToInt16_MaxValues` (subtests: int16_max_(32767), int16_min_(-32768))
- PASS: `TestOGGEncoder_ImplementsFileEncoder`
- PASS: `TestOGGEncoder_StartsWithOggS`
- PASS: `TestOGGEncoder_NonEmptyOutput`
- PASS: `TestOGGEncoder_InvalidSampleRate` (subtests: zero, negative)
- PASS: `TestOGGEncoder_InvalidChannels` (subtests: zero, three, negative)
- PASS: `TestOGGEncoder_OpusError`
- PASS: `TestOGGEncoder_EmptyFrames`
- PASS: `TestOGGEncoder_WriterError`
- PASS: `TestNewOGGEncoder_NotNil`
- PASS: `TestNewOGGEncoderWithOpus_NotNil`
- PASS: `TestOGGEncoder_VariousFrameCounts` (subtests: 1_frame, 10_frames, 50_frames, 150_frames_(3s))
- PASS: `TestGopusFrameEncoder_ImplementsInterface`
- PASS: `TestGopusFrameEncoder_FrameCount_3s`
- PASS: `TestGopusFrameEncoder_FrameCount_1Frame`
- PASS: `TestGopusFrameEncoder_PartialFrame`
- PASS: `TestGopusFrameEncoder_SingleSample`
- PASS: `TestGopusFrameEncoder_EmptyInput`
- PASS: `TestGopusFrameEncoder_FrameSizeBounds`
- PASS: `TestGopusFrameEncoder_MonoEncoding`
- PASS: `TestGopusFrameEncoder_InvalidSampleRate` (subtests: zero, 22050_(unsupported_by_opus), negative)
- PASS: `TestGopusFrameEncoder_InvalidChannels` (subtests: zero, three, negative)
- PASS: `TestGopusFrameEncoder_ChannelsClosed`
- PASS: `TestGopusFrameEncoder_ChannelsClosed_OnError`
- PASS: `TestNewGopusFrameEncoder_NotNil`
- PASS: `TestNewGopusFrameEncoderWithBitrate_NotNil`
- PASS: `TestWAVEncoder_ImplementsFileEncoder`
- PASS: `TestWAVEncoder_HeaderByteLayout`
- PASS: `TestWAVEncoder_OutputSize` (subtests: 1_sample_stereo, 100_samples_stereo, 48000_samples_mono, 1000_samples_stereo_44100, 0_samples)
- PASS: `TestWAVEncoder_Stereo48kHz`
- PASS: `TestWAVEncoder_Mono44100`
- PASS: `TestWAVEncoder_EmptyInput`
- PASS: `TestWAVEncoder_SingleSample`
- PASS: `TestWAVEncoder_PCMDataPreserved`
- PASS: `TestWAVEncoder_InvalidSampleRate` (subtests: zero, negative, large_negative)
- PASS: `TestWAVEncoder_InvalidChannels` (subtests: zero, negative, three, large)
- PASS: `TestWAVEncoder_WriterError`
- PASS: `TestWAVEncoder_HeaderFields_TableDriven` (subtests: 48kHz_stereo_1s, 44100Hz_mono_0.5s, 48kHz_mono_20ms, 22050Hz_stereo_1s)

### Package: `github.com/JamesPrial/go-scream/internal/scream`
- PASS: `Test_NewServiceWithDeps_ReturnsNonNil`
- PASS: `Test_NewServiceWithDeps_NilPlayer`
- PASS: `Test_NewServiceWithDeps_StoresConfig`
- PASS: `Test_Play_HappyPath`
- PASS: `Test_Play_UsesPresetParams`
- PASS: `Test_Play_PassesGuildAndChannelToPlayer`
- PASS: `Test_Play_Validation` (subtests: empty_guild_ID_returns_error, nil_player_returns_ErrNoPlayer)
- PASS: `Test_Play_GeneratorError`
- PASS: `Test_Play_PlayerError`
- PASS: `Test_Play_DryRun_SkipsPlayer`
- PASS: `Test_Play_DryRun_NilPlayerOK`
- PASS: `Test_Play_ContextCancelled`
- PASS: `Test_Play_UnknownPreset`
- PASS: `Test_Play_MultiplePresets` (subtests: classic, whisper, death-metal, glitch, banshee, robot)
- PASS: `Test_Generate_HappyPath_OGG`
- PASS: `Test_Generate_HappyPath_WAV`
- PASS: `Test_Generate_NoTokenRequired`
- PASS: `Test_Generate_GeneratorError`
- PASS: `Test_Generate_FileEncoderError`
- PASS: `Test_Generate_UnknownPreset`
- PASS: `Test_Generate_PlayerNotInvoked`
- PASS: `Test_Close_WithCloser`
- PASS: `Test_Close_WithCloserError`
- PASS: `T_Close_NilCloser`
- PASS: `Test_Close_CalledTwice_NoPanic`
- PASS: `Test_ListPresets_ReturnsAllPresets`
- PASS: `Test_ListPresets_ContainsExpectedNames`
- PASS: `Test_ListPresets_NoDuplicates`
- PASS: `Test_ListPresets_Deterministic`
- PASS: `Test_ResolveParams_PresetOverridesDuration`
- PASS: `Test_ResolveParams_EmptyPresetUsesRandomize`
- PASS: `Test_SentinelErrors_Exist` (subtests: ErrNoPlayer, ErrUnknownPreset, ErrGenerateFailed, ErrEncodeFailed, ErrPlayFailed)
- PASS: `Test_Play_GeneratorError_WrapsOriginal`
- PASS: `Test_Play_PlayerError_WrapsOriginal`
- PASS: `Test_Generate_GeneratorError_WrapsOriginal`
- PASS: `Test_Generate_EncoderError_WrapsOriginal`

### Package: `github.com/JamesPrial/go-scream/pkg/version`
- PASS: `TestDefaultVersion`
- PASS: `TestDefaultCommit`
- PASS: `TestDefaultDate`
- PASS: `TestString_DefaultValues`
- PASS: `TestString_Format` (subtests: default_values, release_version, prerelease_version, empty_strings)
- PASS: `TestString_MatchesExpectedFormat`

---

## Coverage Per Package

| Package | Coverage |
|---------|----------|
| `cmd/scream` | 0.0% (no test files) |
| `cmd/skill` | 21.7% |
| `internal/audio` | 87.5% |
| `internal/audio/ffmpeg` | 90.6% |
| `internal/audio/native` | 100.0% |
| `internal/config` | 97.6% |
| `internal/discord` | 64.1% |
| `internal/encoding` | 85.7% |
| `internal/scream` | 95.0% |
| `pkg/version` | 100.0% |

---

## Race Detection

No races detected.

Three macOS linker (`ld`) warnings appeared during race-instrumented binary linking:
```
ld: warning: '/private/var/folders/.../000012.o' has malformed LC_DYSYMTAB,
    expected 98 undefined symbols to start at index 1626, found 95 undefined
    symbols starting at index 1626
```
This is a known macOS SDK/toolchain artifact (not a Go race condition). It appears for packages `internal/encoding`, `cmd/skill`, and `internal/scream` only when building race-instrumented binaries. All tests still pass. This is pre-existing and not introduced by any refactoring.

---

## Static Analysis (`go vet`)

No warnings. Clean.

---

## Linter (`golangci-lint`)

```
0 issues.
```

---

## Pre-existing Skips (not failures)

The following 14 tests are permanently skipped because `ffmpeg` is not installed on this machine. This is expected and by design — the tests guard themselves with `t.Skip()`:

- `TestNewFFmpegGenerator_Success`
- `TestFFmpegGenerator_CorrectByteCount`
- `TestFFmpegGenerator_NonSilent`
- `TestFFmpegGenerator_AllPresets`
- `TestFFmpegGenerator_AllNamedPresets`
- `TestFFmpegGenerator_InvalidDuration`
- `TestFFmpegGenerator_NegativeDuration`
- `TestFFmpegGenerator_InvalidSampleRate`
- `TestFFmpegGenerator_NegativeSampleRate`
- `TestFFmpegGenerator_InvalidChannels`
- `TestFFmpegGenerator_ZeroChannels`
- `TestFFmpegGenerator_InvalidAmplitude`
- `TestFFmpegGenerator_InvalidCrusherBits`
- `TestFFmpegGenerator_InvalidLimiterLevel`
- `TestFFmpegGenerator_EvenByteCount`
- `TestFFmpegGenerator_StereoAligned`
- `TestFFmpegGenerator_MonoOutput`
- `TestFFmpegGenerator_DeterministicOutput`

---

## Total Test Count

- **Passed:** ~200+ individual test functions/subtests
- **Failed:** 0
- **Skipped:** 18 (all ffmpeg-dependent, no ffmpeg on PATH)
- **Race conditions:** 0
- **Vet warnings:** 0
- **Lint issues:** 0
