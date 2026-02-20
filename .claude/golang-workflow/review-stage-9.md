# Stage 9 Review: Extract signal-context + closer defer pattern

**Reviewer:** Go Code Reviewer (Opus 4.6)
**Date:** 2026-02-19
**Files reviewed:**
- `/Users/jamesprial/code/go-scream/cmd/scream/service.go`
- `/Users/jamesprial/code/go-scream/cmd/scream/play.go`
- `/Users/jamesprial/code/go-scream/cmd/scream/generate.go`

---

## 1. Behavior Preservation

### Signal handling -- PRESERVED

The original `play.go` and `generate.go` both contained identical signal setup:

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
```

The new `runWithService` in `service.go` (lines 47-48) reproduces this exactly: same signals (`SIGINT`, `SIGTERM`), same base context (`context.Background()`), same deferred `stop()`. The context is passed through to the callback `fn`, so downstream functions receive the same cancellation-aware context.

### Service construction -- PRESERVED

The original code called `newServiceFromConfig(cfg)` and checked the error. The new code does the same at line 50. The `newServiceFromConfig` function itself was also refactored (from inline wiring to calling `app.NewGenerator`, `app.NewFileEncoder`, and `app.NewDiscordDeps`), but that was part of a prior stage. The behavior of the factory is unchanged.

### Closer defer pattern -- PRESERVED

Both originals used this pattern:

```go
if closer != nil {
    defer func() {
        if cerr := closer.Close(); cerr != nil {
            fmt.Fprintf(os.Stderr, "warning: failed to close session: %v\n", cerr)
        }
    }()
}
```

The new `runWithService` (lines 54-59) reproduces this verbatim. The nil guard, error message format string, and stderr destination are all identical.

### Execution order -- PRESERVED

In the originals, the order was:
1. Create signal context
2. Build service via `newServiceFromConfig`
3. Defer closer (if non-nil)
4. Execute command-specific logic

The new `runWithService` preserves this exact order. The callback `fn` runs at line 61 after all setup is complete.

### Error propagation -- PRESERVED

- `newServiceFromConfig` errors are returned directly (line 52).
- The callback's error is returned directly (line 61).
- No error wrapping or transformation was added or removed.

### play.go command-specific behavior -- PRESERVED

The `runPlay` function still:
- Calls `buildConfig`, sets `GuildID` and `channelID` from args (lines 28-38)
- Checks for missing token (line 40-42)
- Validates config (lines 44-46)
- Prints verbose output to `cmd.OutOrStdout()` (lines 48-54)
- Calls `svc.Play(ctx, cfg.GuildID, channelID)` (line 57)

All of this is identical to before. The only change is that the 10-line boilerplate was replaced with `runWithService`.

### generate.go command-specific behavior -- PRESERVED

The `runGenerate` function still:
- Calls `buildConfig`, validates config (lines 28-37)
- Prints verbose output (lines 38-40)
- Creates the output file with `os.Create` (line 43)
- Defers file close with stderr warning (lines 47-51)
- Calls `svc.Generate(ctx, f)` (line 52)

The error message `"failed to create output file: %w"` is preserved (line 45). The file close warning message `"warning: failed to close output file: %v\n"` is preserved (line 49).

**Minor observation:** In the original `generate.go`, the file was created *after* the service was built but *before* `svc.Generate` was called. In the refactored version, file creation happens inside the callback, which still executes after service creation. The execution order is unchanged: context -> service -> file create -> generate.

---

## 2. API Break Detection

**Planned changes from api-changes.md:**

| Change Type | Package | Symbol | Breaking? |
|-------------|---------|--------|-----------|
| NEW | `cmd/scream` | `runWithService` (unexported helper) | No |
| SIGNATURE | `cmd/scream` | `newServiceFromConfig` (internal wiring replaced) | No |

**Actual changes observed:**

- `runWithService` added as unexported function -- matches plan.
- `newServiceFromConfig` signature unchanged (`(config.Config) (*scream.Service, io.Closer, error)`) -- matches plan.
- No exported symbols added, removed, or modified in any file.
- `runPlay` and `runGenerate` signatures unchanged -- both remain `func(cmd *cobra.Command, args []string) error`.

**Verdict: No unplanned API changes detected.**

---

## 3. Refactoring Quality

### Duplication consolidated

The 10-line pattern (signal context + service creation + nil-guarded closer defer) was duplicated identically in `play.go` and `generate.go`. It is now in a single location in `service.go:runWithService`. Both callers pass a focused closure containing only their command-specific logic.

### Abstraction level is appropriate

The `runWithService` helper:
- Takes `config.Config` (the data needed to build the service)
- Takes `func(ctx context.Context, svc *scream.Service) error` (the command-specific work)
- Returns `error`

This is a natural "setup + delegate" pattern. The callback signature provides exactly what callers need (a cancellable context and a ready-to-use service). No over-abstraction.

### File placement is logical

The helper lives in `service.go` alongside `newServiceFromConfig`, which it calls. Both are service-lifecycle concerns. Good cohesion.

---

## 4. Code Quality

### Error handling

- `newServiceFromConfig` error returned directly without double-wrapping (line 52) -- correct.
- `closer.Close()` error logged to stderr with "warning:" prefix -- follows the user's preference for CLI code (per MEMORY.md).
- Callback errors returned transparently (line 61) -- correct.

### Nil safety

- `closer != nil` guard at line 54 before deferring `Close()` -- present and correct.
- This is essential because `newServiceFromConfig` returns `nil` closer when `cfg.Token` is empty.

### Documentation

- `runWithService` has a clear doc comment (lines 43-45) explaining all three things it does: signal context, service creation, closer defer.
- `newServiceFromConfig` retains its existing doc comment.

### Naming

- `runWithService` -- clear, descriptive, follows Go conventions for unexported helpers.
- `fn` parameter name is concise but clear in context.
- `cfg` parameter name is consistent with usage throughout the codebase.

### Imports

- `service.go` imports `context`, `fmt`, `io`, `os`, `os/signal`, `syscall` -- all used.
- `play.go` removed `os`, `os/signal`, `syscall` -- no longer needed. Added `scream` for the callback type signature.
- `generate.go` removed `os/signal`, `syscall` -- no longer needed. Added `scream` for the callback type signature. Retains `os` for `os.Create` and `os.Stderr`.
- All imports are clean with no unused imports.

### Defer ordering correctness

In `runWithService`, the defer stack is:
1. `defer stop()` (line 48) -- registered first, runs last
2. `defer closer.Close()` (line 55) -- registered second, runs first

This means the closer is cleaned up *before* the signal notification is stopped, which is the correct order: you want the Discord session closed while the context is still "live" (not yet cleaned up by `stop()`).

---

## 5. Cross-cutting Observations

### `cmd/skill/main.go` does not use `runWithService`

The skill binary continues to wire dependencies inline in `main()`. This is intentional and appropriate -- the skill binary uses `os.Exit` error handling rather than cobra's `RunE`, always requires a Discord token (no optional closer), and has its own config resolution path. The `runWithService` helper is correctly scoped to `cmd/scream` only.

### Consistency with `app` package usage

Both `service.go` (in `newServiceFromConfig`) and `cmd/skill/main.go` now use the same `app.NewGenerator`, `app.NewFileEncoder`, and `app.NewDiscordDeps` helpers, achieving consistency in dependency wiring while keeping the lifecycle management (signal context, defer patterns) at the appropriate CLI-specific level.

---

## Verdict

**APPROVE**

The refactoring cleanly extracts the duplicated signal-context + closer defer pattern into `runWithService` without any behavioral changes. Signal handling, error propagation, execution order, and cleanup semantics are all preserved exactly. No unplanned API changes were introduced. The abstraction level is appropriate, the code is well-documented, and the implementation follows all Go conventions and project-specific preferences (stderr warnings for deferred close in CLI code).
