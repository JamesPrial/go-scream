# Code Review: Stage 5a (Configuration + Version)

## Files Reviewed

**Implementation:**
- `/Users/jamesprial/code/go-scream/internal/config/errors.go`
- `/Users/jamesprial/code/go-scream/internal/config/config.go`
- `/Users/jamesprial/code/go-scream/internal/config/load.go`
- `/Users/jamesprial/code/go-scream/internal/config/validate.go`
- `/Users/jamesprial/code/go-scream/pkg/version/version.go`

**Tests:**
- `/Users/jamesprial/code/go-scream/internal/config/config_test.go`
- `/Users/jamesprial/code/go-scream/internal/config/load_test.go`
- `/Users/jamesprial/code/go-scream/internal/config/validate_test.go`
- `/Users/jamesprial/code/go-scream/pkg/version/version_test.go`

**Reference patterns:**
- `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go`
- `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/errors.go`
- `/Users/jamesprial/code/go-scream/internal/discord/errors.go`

---

## Verdict: REQUEST_CHANGES

---

## Issues Found

### 1. [MUST FIX] Unused sentinel errors: `ErrMissingToken`, `ErrMissingGuildID`, `ErrMissingOutput`

**File:** `/Users/jamesprial/code/go-scream/internal/config/errors.go` (lines 30-37)

Three sentinel errors are declared but never referenced anywhere in the codebase outside their declarations:

```go
ErrMissingToken   = errors.New("config: discord token is required")
ErrMissingGuildID = errors.New("config: guild ID is required")
ErrMissingOutput  = errors.New("config: output file path is required")
```

`Validate()` in `/Users/jamesprial/code/go-scream/internal/config/validate.go` does not check `Token`, `GuildID`, or `OutputFile` at all. These sentinel errors are dead code.

**Resolution:** Either (a) add validation logic in `Validate()` for these fields and corresponding test cases, or (b) remove the unused sentinels. Given that Token and GuildID are required for the bot to function, option (a) is strongly preferred. If these are intentionally deferred to a later stage (e.g., CLI wiring), add a comment explaining that.

### 2. [MUST FIX] Missing test coverage for `UnmarshalYAML` error path (invalid duration string)

**File:** `/Users/jamesprial/code/go-scream/internal/config/load_test.go` (lines 178-225)

The `TestLoad_DurationFormats` table test defines a `wantErr bool` field in its struct, but no test case sets `wantErr: true`. The `UnmarshalYAML` error path at `/Users/jamesprial/code/go-scream/internal/config/config.go` line 85:

```go
return fmt.Errorf("config: invalid duration %q: %w", raw.Duration.Value, err)
```

...is never exercised by any test. Add a test case such as:

```go
{
    name:    "invalid duration string",
    yaml:    `duration: notaduration`,
    wantErr: true,
},
{
    name:    "bare integer without unit",
    yaml:    `duration: "5"`,
    wantErr: true,
},
```

### 3. [SHOULD FIX] `Merge()` boolean semantics prevent unsetting `DryRun`/`Verbose`

**File:** `/Users/jamesprial/code/go-scream/internal/config/config.go` (lines 137-142)

The `Merge()` function uses zero-value semantics, which means `false` (the zero value for `bool`) can never override a `true` base value:

```go
if overlay.DryRun {
    result.DryRun = overlay.DryRun
}
if overlay.Verbose {
    result.Verbose = overlay.Verbose
}
```

This is documented and tested (`TestMerge_FieldTypes`, "bool field: false overlay preserves base true"), so it is an intentional design choice. However, it creates an asymmetry: once `DryRun` or `Verbose` is set to `true` by a lower-precedence source (e.g., YAML file), a higher-precedence source (e.g., env vars or CLI flags) cannot turn it off via `Merge`. The same limitation exists for `Volume` (cannot set to exactly `0.0` via overlay).

This is documented in the function comment, so it is acceptable as-is, but should be noted as a known limitation. If the CLI layer uses a separate mechanism (e.g., pointer-based optionality or a separate "explicitly set" bitfield), this is fine. Otherwise, consider documenting this limitation in the package doc or adding a `// NOTE:` comment.

### 4. [SHOULD FIX] `ApplyEnv` missing support for `SCREAM_DRY_RUN` and `SCREAM_OUTPUT_FILE`

**File:** `/Users/jamesprial/code/go-scream/internal/config/load.go` (lines 47-78)

The `Config` struct has `DryRun` and `OutputFile` fields, but `ApplyEnv` does not support environment variables for them. `ApplyEnv` handles `SCREAM_VERBOSE` (a bool field) but not `SCREAM_DRY_RUN` (also a bool field). Similarly, `OutputFile` (a string field) has no corresponding env var while other string fields like `SCREAM_GUILD_ID`, `SCREAM_PRESET`, and `SCREAM_FORMAT` do.

If this is intentional (e.g., `DryRun` and `OutputFile` are CLI-only flags), add a comment explaining the omission. Otherwise, add support for `SCREAM_DRY_RUN` and `SCREAM_OUTPUT_FILE`.

### 5. [MINOR] `Load` non-existence fallback wraps `ErrConfigNotFound` for all read errors

**File:** `/Users/jamesprial/code/go-scream/internal/config/load.go` (lines 19-24)

```go
if os.IsNotExist(err) {
    return Config{}, fmt.Errorf("%w: %s: %w", ErrConfigNotFound, path, err)
}
return Config{}, fmt.Errorf("%w: %w", ErrConfigNotFound, err)
```

The fallback (line 23) wraps all non-"not found" read errors (e.g., permission denied) as `ErrConfigNotFound`. A permission error is semantically different from a file-not-found error. Consider either (a) introducing an `ErrConfigRead` sentinel for generic read failures, or (b) returning the raw error wrapped without the `ErrConfigNotFound` sentinel. This is minor because it only affects error classification for callers using `errors.Is`.

---

## Positive Observations

### Error handling patterns: Consistent with codebase

The `fmt.Errorf("%w: %w", sentinel, err)` double-wrapping pattern in `load.go` is consistent with the established patterns across the codebase (e.g., `internal/encoding/opus.go`, `internal/encoding/wav.go`, `internal/audio/ffmpeg/generator.go`, `internal/discord/channel.go`). Go 1.25 fully supports multiple `%w` verbs.

### Sentinel errors: Well-documented and well-structured

All sentinel errors in `/Users/jamesprial/code/go-scream/internal/config/errors.go` follow the `config: description` prefix convention, consistent with other packages (`encoding: ...`, `ffmpeg: ...`, `discord: ...`). Each has a doc comment.

### Custom UnmarshalYAML: Correct approach

Using `rawConfig` with `yaml.Node` for the duration field is an appropriate technique. It avoids the default YAML integer-nanoseconds interpretation and provides human-readable duration strings. The implementation correctly handles the empty case (line 82: `if raw.Duration.Value != ""`).

### Validate(): Clean short-circuit design

`Validate()` returns on the first error encountered, which is a clean pattern. Tests verify the behavior for each field independently using table-driven tests with `errors.Is`.

### Merge(): Correct non-mutation semantics

`Merge()` takes value parameters and returns a new value, ensuring neither input is mutated. This is verified by `TestMerge_DoesNotMutateInputs`.

### Test quality: Thorough and well-structured

- Table-driven tests follow Go conventions (`TestValidate_Backend`, `TestValidate_Volume`, etc.)
- Edge cases covered: empty config, partial YAML, unknown fields silently ignored, case sensitivity
- `t.Setenv` is correctly used for env var tests (auto-cleaned up)
- Version tests correctly save/restore package-level vars with `t.Cleanup` and avoid `t.Parallel`
- Benchmarks included for hot-path functions
- `t.Helper()` correctly used in table test check functions

### Version package: Simple and correct

The `pkg/version` package is minimal and appropriate for `ldflags` injection. Tests cover default values, format string, and custom values.

### Documentation: Complete

All exported types, functions, constants, and variables have doc comments. Package-level documentation is present on both packages.

---

## Checklist

**Code Quality:**
- [x] All exported items have documentation
- [x] Error handling follows `%w` wrapping patterns
- [x] Nil safety guards present (pointer receiver on `ApplyEnv`, value semantics on `Merge`)
- [x] Table tests structured correctly with named sub-tests
- [x] Code is readable and well-organized
- [x] Naming conventions followed
- [ ] No dead code -- three unused sentinel errors (Issue #1)
- [ ] All error paths tested -- `UnmarshalYAML` error path untested (Issue #2)

---

## Summary

The implementation is well-crafted and consistent with the rest of the codebase. The two must-fix issues are:

1. Three sentinel errors (`ErrMissingToken`, `ErrMissingGuildID`, `ErrMissingOutput`) are declared but never used -- either wire them into `Validate()` or remove them.
2. The `UnmarshalYAML` error path for invalid duration strings has zero test coverage despite the test table being structured to support it.

The should-fix items (boolean merge semantics documentation and `ApplyEnv` completeness) are lower priority but worth addressing for consistency.
