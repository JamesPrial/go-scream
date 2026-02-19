# Design Review: Stage 5c -- CLI (`cmd/scream/`)

**Reviewer:** Go Reviewer Agent (Opus 4.6)
**Date:** 2026-02-18
**Scope:** `/Users/jamesprial/code/go-scream/cmd/scream/` (main.go, flags.go, play.go, generate.go, presets.go)

---

## Verdict: REQUEST_CHANGES

The CLI is well-structured overall with clean separation of concerns, correct cobra patterns, and a solid `buildConfig` implementation that faithfully follows the Defaults -> YAML -> env -> flags resolution order. However, there are several issues that should be addressed before merge.

---

## Issues Requiring Changes

### 1. MEDIUM: Package-level mutable state for flags creates shared state hazard

**Files:** `/Users/jamesprial/code/go-scream/cmd/scream/flags.go` (lines 11-19), `/Users/jamesprial/code/go-scream/cmd/scream/main.go` (lines 13-16)

All flag variables (`tokenFlag`, `presetFlag`, `durationFlag`, etc.) are package-level `var` declarations. While this is a common cobra pattern for simple CLIs, it means `addAudioFlags` binds every command's flags to the same global variables. Today the commands are `play` and `generate`, which are mutually exclusive at runtime, so this works. However, it will be fragile if sub-commands are ever composed or if tests need to exercise commands in parallel within the same process.

**Recommendation:** This is acceptable for now given the simple CLI structure, but add a brief comment on `flags.go` acknowledging that the package-level vars are safe because cobra enforces single-command execution. This documents the design intent and warns future contributors.

### 2. HIGH: No test files exist for the CLI package

**File:** `/Users/jamesprial/code/go-scream/cmd/scream/` (no `*_test.go` files found)

There are zero tests for the CLI layer. The `buildConfig` function is the most critical piece of logic in this package -- it implements the four-layer config resolution (Default -> YAML -> env -> flags) -- and it has no test coverage at all. The `runPlay` and `runGenerate` functions also contain validation logic (e.g., checking for empty token, empty output) that should be verified.

**Recommendation:** Add at minimum:
- Table-driven tests for `buildConfig` verifying the precedence chain (a flag set via `cmd.Flags().Set("token", "xxx")` should override env, which overrides YAML, which overrides defaults).
- Tests for `runPlay` error paths: missing token, missing guildID (0 args -- though cobra handles this), verbose output.
- Tests for `runGenerate` error paths: missing output, validation failure.
- Test for `runPresets` verifying output format.

Cobra commands are straightforward to test by calling `cmd.Execute()` after setting `cmd.SetArgs(...)` and `cmd.SetOut(...)`.

### 3. MEDIUM: Redundant output-file assignment in `generate.go`

**File:** `/Users/jamesprial/code/go-scream/cmd/scream/generate.go` (lines 33-36)

```go
cfg.OutputFile = outputFlag  // line 33

if cfg.OutputFile == "" {    // line 36
    return config.ErrMissingOutput
}
```

Line 33 unconditionally sets `cfg.OutputFile = outputFlag`, but `buildConfig` (called on line 29) already handles the `--output` flag at line 55-57 of `flags.go`:

```go
if cmd.Flags().Changed("output") {
    cfg.OutputFile = outputFlag
}
```

Since `--output` is marked as required via `generateCmd.MarkFlagRequired("output")` (line 23), cobra will reject invocations without it before `runGenerate` is ever called. This means:
- The assignment on line 33 is redundant with `buildConfig`.
- The `cfg.OutputFile == ""` check on line 36 is unreachable (cobra enforces the required flag).

**Recommendation:** Remove the redundant assignment on line 33 and the dead `ErrMissingOutput` check on lines 36-38. If you want defense-in-depth (e.g., in case `buildConfig` is called from tests without cobra's required-flag enforcement), keep the empty check but remove the redundant assignment, and add a comment explaining why.

### 4. MEDIUM: `play.go` sets `cfg.GuildID` after `buildConfig` but before `Validate` -- inconsistent with flag pattern

**File:** `/Users/jamesprial/code/go-scream/cmd/scream/play.go` (lines 33-38)

```go
cfg.GuildID = args[0]

channelID := ""
if len(args) > 1 {
    channelID = args[1]
}
```

The `GuildID` is set directly from positional args, which is correct. However, `channelID` is extracted as a local variable and never placed into `cfg`. This means the future TODO integration will need to pass `channelID` separately to `svc.Play(ctx, cfg.GuildID, channelID)`, which is fine and matches the service API. But it is worth noting that `channelID` is not validated at all -- an empty string is silently passed through. If the service requires a non-empty `channelID` in non-discovery mode, that validation should exist somewhere.

**Recommendation:** Add a brief doc comment explaining that `channelID` is optional because the service supports channel auto-discovery when it is empty. If that is not the case, add validation.

### 5. LOW: Verbose flag uses different override pattern than other flags

**File:** `/Users/jamesprial/code/go-scream/cmd/scream/flags.go` (lines 61-63)

```go
if verbose {
    cfg.Verbose = true
}
```

All other flags use `cmd.Flags().Changed(...)` to check whether the user explicitly set the flag. The `verbose` flag instead checks the boolean value directly. This means `--verbose=false` will not override a `verbose: true` in the config file (because `if verbose` is false, the block is skipped, and the YAML/env value persists). This is arguably correct UX (you rarely want `--verbose=false`), but it is inconsistent with the other flags and could surprise a user.

**Recommendation:** Either use `cmd.Flags().Changed("verbose")` for consistency:

```go
if cmd.Flags().Changed("verbose") {
    cfg.Verbose = verbose
}
```

Or document why verbose is intentionally one-directional (can only be turned on, not off, via CLI).

### 6. LOW: `play.go` verbose output inconsistency -- missing newline in format string

**File:** `/Users/jamesprial/code/go-scream/cmd/scream/play.go` (lines 50-55)

```go
fmt.Fprintf(cmd.OutOrStdout(), "Playing scream in guild %s", cfg.GuildID)
if channelID != "" {
    fmt.Fprintf(cmd.OutOrStdout(), " channel %s", channelID)
}
fmt.Fprintln(cmd.OutOrStdout())
```

Compare with `generate.go` line 45:

```go
fmt.Fprintf(cmd.OutOrStdout(), "Generating scream to %s (format: %s)\n", cfg.Format)
```

The play command uses three separate print calls to build one line of output. This is fragile if any concurrent output occurs (unlikely in a CLI, but still). The generate command uses a single `Fprintf` with `\n`.

**Recommendation:** Consolidate the play verbose output into a single `Fprintf` call:

```go
if cfg.Verbose {
    msg := fmt.Sprintf("Playing scream in guild %s", cfg.GuildID)
    if channelID != "" {
        msg += fmt.Sprintf(" channel %s", channelID)
    }
    fmt.Fprintln(cmd.OutOrStdout(), msg)
}
```

---

## Positive Observations

### Config resolution chain is correctly implemented

The `buildConfig` function in `flags.go` faithfully implements the specified precedence: `Default() -> Load(yaml) -> ApplyEnv() -> CLI flags`. The use of `cmd.Flags().Changed(...)` for CLI flag overrides is the correct cobra idiom -- it ensures that unset flags do not clobber values from earlier layers.

### Clean separation of concerns

The CLI layer correctly delegates to `config.Validate`, `config.Load`, `config.ApplyEnv`, and `scream.ListPresets`. The run functions do not contain business logic -- they build config, validate, and hand off to the service. The TODO stubs for service wiring are well-placed.

### Cobra best practices followed

- `PersistentFlags` used correctly for `--config` and `--verbose` (apply to all subcommands).
- `RangeArgs(1, 2)` on `play` correctly enforces the `<guildID> [channelID]` argument structure.
- `MarkFlagRequired("output")` on `generate` leverages cobra's built-in validation.
- Commands registered via `init()` in their own files is standard cobra project layout.
- `RunE` used for commands that can fail; `Run` used for `presets` which cannot.

### Good use of `cmd.OutOrStdout()`

All output uses `cmd.OutOrStdout()` rather than `os.Stdout`, making the commands testable by injecting a buffer via `cmd.SetOut()`.

### Error types from config package reused correctly

`config.ErrMissingToken` and `config.ErrMissingOutput` are used directly rather than defining duplicate errors in the CLI package.

### Shared flag helper avoids duplication

`addAudioFlags` cleanly applies the common flag set (preset, duration, volume, backend) to both `play` and `generate` commands, avoiding copy-paste.

---

## Summary

| # | Severity | Issue | File |
|---|----------|-------|------|
| 1 | MEDIUM | Package-level flag vars need documenting | `flags.go` |
| 2 | HIGH | No test files for CLI package | `cmd/scream/` |
| 3 | MEDIUM | Redundant output assignment and dead code in generate | `generate.go:33-38` |
| 4 | MEDIUM | channelID not documented as optional | `play.go:37` |
| 5 | LOW | Verbose flag override inconsistent with other flags | `flags.go:61-63` |
| 6 | LOW | Verbose output in play uses fragile multi-print pattern | `play.go:50-55` |

The most critical item is **#2 (missing tests)**. The `buildConfig` function is the heart of the CLI and must have test coverage verifying the four-layer precedence chain. Items #3 and #5 are correctness concerns (dead code and inconsistent flag semantics). The remaining items are polish.
