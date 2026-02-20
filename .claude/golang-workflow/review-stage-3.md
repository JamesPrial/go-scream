# Stage 3 Review: Remove `reflect` dependency and dead `closer` field

**Reviewer:** Go Reviewer Agent
**Date:** 2026-02-19
**Files reviewed:**
- `/Users/jamesprial/code/go-scream/internal/scream/service.go`
- `/Users/jamesprial/code/go-scream/internal/scream/service_test.go`
- `/Users/jamesprial/code/go-scream/internal/scream/errors.go`
- `/Users/jamesprial/code/go-scream/internal/scream/resolve.go`

---

## 1. Behavior Preservation

**PASS.** All remaining methods are unchanged in behavior:

- `Play()` (lines 46-92 of `service.go`): Identical logic -- validates guildID, checks player nil guard, checks context, resolves params, generates audio, encodes frames, plays or drains in dry-run mode. Error wrapping with `%w` is preserved for all sentinel errors (`ErrGenerateFailed`, `ErrEncodeFailed`, `ErrPlayFailed`).

- `Generate()` (lines 96-116 of `service.go`): Identical logic -- checks context, resolves params, generates, encodes to file. Same error wrapping.

- `ListPresets()` (lines 119-126 of `service.go`): Unchanged.

- `NewServiceWithDeps()` (lines 27-41 of `service.go`): The function body is simplified to a direct struct literal return. The typed-nil normalization block using `reflect` has been removed, and the doc comment now explicitly states callers must pass untyped nil. This is the intended behavioral shift -- the responsibility moves to the caller.

No new functionality was added.

## 2. API Break Detection

**PASS.** Only the two approved deletions were made:

| Approved Deletion | Status | Verified |
|---|---|---|
| `Service.closer` (unexported field) | Removed from struct (line 21 area) | Yes |
| `Service.Close()` (exported method) | Removed entirely (was lines 127-134) | Yes |

Additional changes in the diff that are consistent with Stage 2 (already approved):
- `audio.AudioGenerator` -> `audio.Generator` in both the struct field type and constructor parameter. This is from Stage 2 and is expected.

No other structural changes to the `Service` type or its remaining methods.

### reflect import removal

The `"reflect"` import is completely gone from `service.go`. A codebase-wide search confirms no other Go source file imports `"reflect"` -- only a workflow documentation file references it in prose.

## 3. Code Quality

### Production code (`service.go`)

- **Import block** (lines 3-12): Clean. Contains only `context`, `fmt`, `io`, and the four internal packages. No unused imports. The `"reflect"` import is gone.

- **Service struct** (lines 15-21): Five fields, all necessary. No `closer` field. Clean and minimal.

- **Doc comment on `NewServiceWithDeps`** (lines 23-26): Updated to include:
  > Callers must pass an untyped nil (not a typed-nil interface value) when no player is needed.

  This is the correct replacement for the reflect-based normalization. The comment is clear and actionable.

- **No leftover references**: A codebase-wide search for `svc.Close`, `Service.Close`, and the `closer` field in Go source files found zero results in production or test code. All hits are in `.claude/golang-workflow/` documentation files only.

### Support files

- `errors.go`: Unchanged. All five sentinel errors remain with their `"scream:"` prefixed messages and doc comments.
- `resolve.go`: Unchanged. Param resolution logic is intact.

## 4. Test Quality

### Removals (all correct)

- **`mockCloser` type**: Removed (was ~20 lines). No longer needed since `Service.Close()` is gone.

- **Close tests removed** (4 tests, ~75 lines):
  - `Test_Close_WithCloser`
  - `Test_Close_WithCloserError`
  - `Test_Close_NilCloser`
  - `Test_Close_CalledTwice_NoPanic`

  All four correctly removed. These tested the deleted `Close()` method and relied on direct `svc.closer = mc` field assignment.

- **No `svc.closer` assignments remain**: Confirmed via grep. Zero occurrences in any Go source file.

### Preserved tests (all intact)

The test file retains 925 lines with full coverage of all remaining functionality:

- **NewServiceWithDeps**: 3 tests (non-nil return, nil player, config storage)
- **Play**: 10 tests (happy path, preset params, guild/channel forwarding, validation table test, generator error, player error, dry-run skip, dry-run nil player, context cancelled, unknown preset, multiple presets)
- **Generate**: 7 tests (happy path OGG, happy path WAV, no token required, generator error, file encoder error, unknown preset, player not invoked)
- **ListPresets**: 4 tests (returns all, contains expected names, no duplicates, deterministic)
- **resolveParams** (indirect): 2 tests (preset overrides duration, empty preset uses randomize)
- **Sentinel errors**: 1 table-driven test covering all 5 errors
- **Error wrapping**: 4 tests (play generator, play player, generate generator, generate encoder)
- **Benchmarks**: 3 (Play, Generate, ListPresets)

### Minor comment update

Line 22 of the test file updated the mockGenerator doc comment from `audio.AudioGenerator` to `audio.Generator`, consistent with Stage 2 rename. Correct.

## 5. Caller Impact Assessment

The `Service.Close()` method deletion is marked as breaking in the API changes document. However, a codebase-wide search confirms that `Service.Close()` was never called from any production code (`cmd/scream/`, `cmd/skill/`). The CLI manages the Discord session's `io.Closer` lifecycle separately via `newServiceFromConfig` returning a standalone `io.Closer`. The `closer` references in `cmd/scream/play.go` and `cmd/scream/generate.go` are local variables for the Discord session, not the `Service.closer` field. No callers are broken by this deletion.

---

## Verdict

**APPROVE**

All changes are strictly within the approved Stage 3 scope. The `reflect` import and typed-nil normalization are cleanly removed. The `closer` field and `Close()` method are deleted with no residual references. The doc comment on `NewServiceWithDeps` properly documents the untyped-nil requirement. All remaining tests are preserved and structurally sound. No behavioral regressions detected.
