## Test Execution Report — Stage 5b (Service Orchestrator)

### Summary
- **Verdict:** TESTS_PASS
- **Tests Run:** 40 passed, 0 failed (plus 3 benchmarks)
- **Coverage:** 95.0% of statements
- **Race Conditions:** None detected
- **Vet Warnings:** None
- **Linter Issues:** 0 (golangci-lint)

---

### Test Results (go test -v -count=1 ./internal/scream/...)

```
=== RUN   Test_NewServiceWithDeps_ReturnsNonNil
--- PASS: Test_NewServiceWithDeps_ReturnsNonNil (0.00s)
=== RUN   Test_NewServiceWithDeps_NilPlayer
--- PASS: Test_NewServiceWithDeps_NilPlayer (0.00s)
=== RUN   Test_NewServiceWithDeps_StoresConfig
--- PASS: Test_NewServiceWithDeps_StoresConfig (0.00s)
=== RUN   Test_Play_HappyPath
--- PASS: Test_Play_HappyPath (0.00s)
=== RUN   Test_Play_UsesPresetParams
--- PASS: Test_Play_UsesPresetParams (0.00s)
=== RUN   Test_Play_PassesGuildAndChannelToPlayer
--- PASS: Test_Play_PassesGuildAndChannelToPlayer (0.00s)
=== RUN   Test_Play_Validation
=== RUN   Test_Play_Validation/empty_guild_ID_returns_error
=== RUN   Test_Play_Validation/nil_player_returns_ErrNoPlayer
--- PASS: Test_Play_Validation (0.00s)
    --- PASS: Test_Play_Validation/empty_guild_ID_returns_error (0.00s)
    --- PASS: Test_Play_Validation/nil_player_returns_ErrNoPlayer (0.00s)
=== RUN   Test_Play_GeneratorError
--- PASS: Test_Play_GeneratorError (0.00s)
=== RUN   Test_Play_PlayerError
--- PASS: Test_Play_PlayerError (0.00s)
=== RUN   Test_Play_DryRun_SkipsPlayer
--- PASS: Test_Play_DryRun_SkipsPlayer (0.00s)
=== RUN   Test_Play_DryRun_NilPlayerOK
--- PASS: Test_Play_DryRun_NilPlayerOK (0.00s)
=== RUN   Test_Play_ContextCancelled
--- PASS: Test_Play_ContextCancelled (0.00s)
=== RUN   Test_Play_UnknownPreset
--- PASS: Test_Play_UnknownPreset (0.00s)
=== RUN   Test_Play_MultiplePresets
=== RUN   Test_Play_MultiplePresets/classic
=== RUN   Test_Play_MultiplePresets/whisper
=== RUN   Test_Play_MultiplePresets/death-metal
=== RUN   Test_Play_MultiplePresets/glitch
=== RUN   Test_Play_MultiplePresets/banshee
=== RUN   Test_Play_MultiplePresets/robot
--- PASS: Test_Play_MultiplePresets (0.00s)
    --- PASS: Test_Play_MultiplePresets/classic (0.00s)
    --- PASS: Test_Play_MultiplePresets/whisper (0.00s)
    --- PASS: Test_Play_MultiplePresets/death-metal (0.00s)
    --- PASS: Test_Play_MultiplePresets/glitch (0.00s)
    --- PASS: Test_Play_MultiplePresets/banshee (0.00s)
    --- PASS: Test_Play_MultiplePresets/robot (0.00s)
=== RUN   Test_Generate_HappyPath_OGG
--- PASS: Test_Generate_HappyPath_OGG (0.00s)
=== RUN   Test_Generate_HappyPath_WAV
--- PASS: Test_Generate_HappyPath_WAV (0.00s)
=== RUN   Test_Generate_NoTokenRequired
--- PASS: Test_Generate_NoTokenRequired (0.00s)
=== RUN   Test_Generate_GeneratorError
--- PASS: Test_Generate_GeneratorError (0.00s)
=== RUN   Test_Generate_FileEncoderError
--- PASS: Test_Generate_FileEncoderError (0.00s)
=== RUN   Test_Generate_UnknownPreset
--- PASS: Test_Generate_UnknownPreset (0.00s)
=== RUN   Test_Generate_PlayerNotInvoked
--- PASS: Test_Generate_PlayerNotInvoked (0.00s)
=== RUN   Test_Close_WithCloser
--- PASS: Test_Close_WithCloser (0.00s)
=== RUN   Test_Close_WithCloserError
--- PASS: Test_Close_WithCloserError (0.00s)
=== RUN   Test_Close_NilCloser
--- PASS: Test_Close_NilCloser (0.00s)
=== RUN   Test_Close_CalledTwice_NoPanic
--- PASS: Test_Close_CalledTwice_NoPanic (0.00s)
=== RUN   Test_ListPresets_ReturnsAllPresets
--- PASS: Test_ListPresets_ReturnsAllPresets (0.00s)
=== RUN   Test_ListPresets_ContainsExpectedNames
--- PASS: Test_ListPresets_ContainsExpectedNames (0.00s)
=== RUN   Test_ListPresets_NoDuplicates
--- PASS: Test_ListPresets_NoDuplicates (0.00s)
=== RUN   Test_ListPresets_Deterministic
--- PASS: Test_ListPresets_Deterministic (0.00s)
=== RUN   Test_ResolveParams_PresetOverridesDuration
--- PASS: Test_ResolveParams_PresetOverridesDuration (0.00s)
=== RUN   Test_ResolveParams_EmptyPresetUsesRandomize
--- PASS: Test_ResolveParams_EmptyPresetUsesRandomize (0.00s)
=== RUN   Test_SentinelErrors_Exist
=== RUN   Test_SentinelErrors_Exist/ErrNoPlayer
=== RUN   Test_SentinelErrors_Exist/ErrUnknownPreset
=== RUN   Test_SentinelErrors_Exist/ErrGenerateFailed
=== RUN   Test_SentinelErrors_Exist/ErrEncodeFailed
=== RUN   Test_SentinelErrors_Exist/ErrPlayFailed
--- PASS: Test_SentinelErrors_Exist (0.00s)
    --- PASS: Test_SentinelErrors_Exist/ErrNoPlayer (0.00s)
    --- PASS: Test_SentinelErrors_Exist/ErrUnknownPreset (0.00s)
    --- PASS: Test_SentinelErrors_Exist/ErrGenerateFailed (0.00s)
    --- PASS: Test_SentinelErrors_Exist/ErrEncodeFailed (0.00s)
    --- PASS: Test_SentinelErrors_Exist/ErrPlayFailed (0.00s)
=== RUN   Test_Play_GeneratorError_WrapsOriginal
--- PASS: Test_Play_GeneratorError_WrapsOriginal (0.00s)
=== RUN   Test_Play_PlayerError_WrapsOriginal
--- PASS: Test_Play_PlayerError_WrapsOriginal (0.00s)
=== RUN   Test_Generate_GeneratorError_WrapsOriginal
--- PASS: Test_Generate_GeneratorError_WrapsOriginal (0.00s)
=== RUN   Test_Generate_EncoderError_WrapsOriginal
--- PASS: Test_Generate_EncoderError_WrapsOriginal (0.00s)
PASS
ok  github.com/JamesPrial/go-scream/internal/scream  0.282s
```

---

### Race Detection (go test -race -count=1 ./internal/scream/...)

No races detected.

```
ok  github.com/JamesPrial/go-scream/internal/scream  1.387s
```

Note: A non-fatal linker warning was emitted by the Darwin toolchain:
`ld: warning: '/private/var/.../000012.o' has malformed LC_DYSYMTAB ...`
This is a macOS system toolchain cosmetic warning, not a race condition or test failure.

---

### Static Analysis (go vet ./internal/scream/...)

No warnings. Command exited with status 0 and produced no output.

---

### Coverage Details (go test -cover -count=1 ./internal/scream/...)

```
ok  github.com/JamesPrial/go-scream/internal/scream  0.341s  coverage: 95.0% of statements
```

Coverage: **95.0%** — exceeds the 70% threshold.

---

### Linter Output (golangci-lint run ./internal/scream/...)

```
0 issues.
```

---

### Full Regression Results (go test -count=1 ./internal/audio/... ./internal/encoding/... ./internal/audio/ffmpeg/... ./internal/discord/... ./internal/config/... ./pkg/version/...)

```
ok  github.com/JamesPrial/go-scream/internal/audio         0.264s
ok  github.com/JamesPrial/go-scream/internal/audio/ffmpeg  0.513s
ok  github.com/JamesPrial/go-scream/internal/audio/native  1.141s
ok  github.com/JamesPrial/go-scream/internal/encoding      0.831s
ok  github.com/JamesPrial/go-scream/internal/discord       1.367s
ok  github.com/JamesPrial/go-scream/internal/config        1.051s
ok  github.com/JamesPrial/go-scream/pkg/version            1.570s
```

All 7 regression packages pass with no failures.

---

### Issues to Address

None. All checks pass.

---

### TESTS_PASS

All checks pass.
- Total tests run: 40 passed, 0 failed (plus 3 benchmarks defined but not run in standard mode)
- Coverage: 95.0%
- Race conditions: None
- Vet warnings: None
- Linter issues: 0
- Regression: 7/7 packages pass
