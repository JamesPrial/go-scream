# Stage 7 Review: Break config->audio coupling

**Reviewer:** Go Reviewer Agent
**Date:** 2026-02-19
**Verdict:** APPROVE

---

## 1. Decoupling Achieved -- PASS

No file under `/Users/jamesprial/code/go-scream/internal/config/` imports `internal/audio`. Verified by searching all `.go` files in the package. The only reference to `audio` is the documentation comment on lines 3-5 of `/Users/jamesprial/code/go-scream/internal/config/validate.go`, which serves as a maintenance breadcrumb rather than a code dependency.

## 2. Behavior Preservation -- PASS

The hardcoded `knownPresets` slice in `/Users/jamesprial/code/go-scream/internal/config/validate.go` (lines 6-13) exactly matches `AllPresets()` in `/Users/jamesprial/code/go-scream/internal/audio/presets.go` (lines 18-27):

| Index | `knownPresets` (validate.go) | `AllPresets()` (presets.go) |
|-------|------------------------------|----------------------------|
| 0 | `"classic"` | `PresetClassic` = `"classic"` |
| 1 | `"whisper"` | `PresetWhisper` = `"whisper"` |
| 2 | `"death-metal"` | `PresetDeathMetal` = `"death-metal"` |
| 3 | `"glitch"` | `PresetGlitch` = `"glitch"` |
| 4 | `"banshee"` | `PresetBanshee` = `"banshee"` |
| 5 | `"robot"` | `PresetRobot` = `"robot"` |

Values and order match exactly. The `Validate` function signature is unchanged (`func Validate(cfg Config) error`). Validation logic is identical: empty preset is accepted, unknown presets return `ErrInvalidPreset`.

## 3. Maintenance Concern -- NOTED (acceptable trade-off)

The sync comment is present and clear:

```go
// knownPresets lists every valid preset name accepted by Validate.
// This list must be kept in sync with the preset constants defined in
// internal/audio/presets.go (audio.AllPresets).
```

There is no automated cross-package sync guard. Since `knownPresets` is unexported, a test in a third package cannot compare it to `audio.AllPresets()` directly. The existing tests in `/Users/jamesprial/code/go-scream/internal/config/validate_test.go` cover all six preset names individually (lines 76-145), providing a partial safety net. This is an acceptable trade-off for a small, stable list of six presets with a clear sync comment.

## 4. No Unintended Side Effects -- PASS

- Only `/Users/jamesprial/code/go-scream/internal/config/validate.go` was modified.
- The `Validate` function signature is unchanged.
- No test files were modified.
- No other files in `internal/config/` were affected (`config.go`, `errors.go`, `load.go` all unchanged).

## 5. Code Quality -- PASS

- **Documentation:** `knownPresets`, `Validate`, and `isValidPreset` all have doc comments.
- **Error handling:** Returns sentinel errors consistent with the package's `errors.go`.
- **No unused variables or imports.**
- **Helper design:** `isValidPreset` is private, clean, and uses a linear scan over 6 elements (appropriate for the list size).

## Checklist

- [x] Decoupling goal achieved (no `internal/audio` import)
- [x] Preset list matches `audio.AllPresets()` exactly
- [x] Sync comment present pointing to source of truth
- [x] `Validate` signature unchanged
- [x] Only `validate.go` modified
- [x] All exported items documented
- [x] Error handling follows project patterns
- [x] Code is readable and well-organized
