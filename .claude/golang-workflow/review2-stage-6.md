# Design Review: Stage 6 -- OpenClaw Skill Wrapper

**Verdict: REQUEST_CHANGES**

## Files Reviewed

- `/Users/jamesprial/code/go-scream/cmd/skill/main.go`
- `/Users/jamesprial/code/go-scream/cmd/skill/main_test.go`
- `/Users/jamesprial/code/go-scream/SKILL.md`

## Summary

The skill wrapper is well-structured overall. The separation of `parseOpenClawConfig` and `resolveToken` into testable functions is good design. The test coverage for those extracted functions is thorough with proper table-driven tests. However, there are several issues that should be addressed before merge.

## Issues Requiring Changes

### 1. Token double-resolution via ApplyEnv (Medium -- Correctness)

**File:** `/Users/jamesprial/code/go-scream/cmd/skill/main.go`, lines 86-89

```go
cfg := config.Default()
config.ApplyEnv(&cfg)
cfg.Token = token
cfg.GuildID = guildID
```

`resolveToken` already checks `DISCORD_TOKEN` as its highest priority source (line 53). Then `config.ApplyEnv(&cfg)` on line 87 also reads `DISCORD_TOKEN` and sets `cfg.Token`. Then line 88 overwrites `cfg.Token` with the result of `resolveToken`. The final value is correct because the manual assignment on line 88 wins, but this creates a confusing control flow where the token is resolved twice through two different mechanisms.

More importantly, `ApplyEnv` also reads `SCREAM_GUILD_ID` which would set `cfg.GuildID`, but then line 89 unconditionally overwrites it with the positional argument. If someone sets `SCREAM_GUILD_ID` in the environment, the skill wrapper silently ignores it with no indication. This is acceptable for the skill wrapper (where `guildID` is always a required positional arg), but the interaction between `ApplyEnv` and the manual overrides should be documented with a brief comment explaining why both are needed (ApplyEnv for audio params like SCREAM_PRESET/SCREAM_DURATION/SCREAM_VOLUME/SCREAM_BACKEND, manual overrides for skill-specific fields).

**Suggested fix:** Add a comment above the `ApplyEnv` call:

```go
cfg := config.Default()
// ApplyEnv loads audio parameter overrides (SCREAM_PRESET, SCREAM_DURATION,
// SCREAM_VOLUME, SCREAM_BACKEND). Token and GuildID are set explicitly below
// from skill-specific sources, overriding any env values ApplyEnv may set.
config.ApplyEnv(&cfg)
cfg.Token = token
cfg.GuildID = guildID
```

### 2. channelID not wired into config (Low -- Incomplete)

**File:** `/Users/jamesprial/code/go-scream/cmd/skill/main.go`, lines 74-77, 107

The `channelID` is parsed from `os.Args[2]` but is only suppressed with `_ = channelID` in the TODO block. The `config.Config` struct does not have a `ChannelID` field -- channelID is instead passed directly as an argument to `Service.Play(ctx, guildID, channelID)`. This is fine given the TODO comment, but it is worth noting that the channelID will need to flow through to `svc.Play()` when the wiring is completed, not through the config struct. The TODO comment on lines 101-104 already captures this correctly.

No change needed, but flagging for awareness.

### 3. SKILL.md: `bins` should reference `skill` not `node` -- Verified Correct

**File:** `/Users/jamesprial/code/go-scream/SKILL.md`, line 9

The SKILL.md correctly lists `skill` under `requires.bins`:

```yaml
requires:
  bins:
    - skill
```

This correctly reflects the Go binary instead of the reference SKILL.md's `node`. Good.

### 4. SKILL.md: YAML frontmatter style inconsistency with reference (Low -- Convention)

**File:** `/Users/jamesprial/code/go-scream/SKILL.md`, lines 1-12

The reference SKILL.md at `/tmp/scream-reference/SKILL.md` uses inline JSON for the metadata block:

```yaml
metadata: { "openclaw": { "emoji": "...", "requires": { ... } } }
```

The project's SKILL.md uses expanded YAML block style:

```yaml
metadata:
  openclaw:
    emoji: "..."
    requires:
      bins:
        - skill
      config:
        - channels.discord.token
```

Both are valid YAML, and the expanded form is actually more readable. If OpenClaw's parser supports both, this is fine and arguably an improvement. However, if the OpenClaw tooling specifically expects inline JSON metadata, this could be a compatibility issue. **Verify that the OpenClaw parser handles expanded YAML metadata.**

### 5. Missing test for `resolveToken` when `DISCORD_TOKEN` env var is truly unset (Low -- Test gap)

**File:** `/Users/jamesprial/code/go-scream/cmd/skill/main_test.go`

All `resolveToken` tests use `t.Setenv("DISCORD_TOKEN", "")` which sets the variable to an empty string. There is no test case where `DISCORD_TOKEN` is truly absent from the environment (unset). Since `os.Getenv` returns `""` for both unset and empty-string cases, the behavior is identical in practice, but for documentation purposes a test case where the env var is explicitly not set (using `os.Unsetenv` before the test, or simply not calling `t.Setenv`) would demonstrate the intended fallback path more clearly.

This is a minor point -- `t.Setenv` with `""` does exercise the correct code path since the `resolveToken` function checks `v != ""`.

### 6. No test coverage for main() argument parsing (Low -- Acceptable)

The `main()` function's argument parsing (lines 67-78) is not unit tested. This is typical for Go `main` packages where the `main` function itself is hard to test. The testable logic has been properly extracted into `parseOpenClawConfig` and `resolveToken`. If desired, the argument parsing and config building could be extracted into a `run(args []string) error` function for testability, but given this is a thin wrapper with a TODO for service wiring, this is acceptable for now.

### 7. Error message style consistency (Nit)

**File:** `/Users/jamesprial/code/go-scream/cmd/skill/main.go`, line 82

```go
fmt.Fprintln(os.Stderr, "skill: discord token is required (set DISCORD_TOKEN or configure ~/.openclaw/openclaw.json)")
```

The `config` package uses the sentinel error `ErrMissingToken` with message `"config: discord token is required"`. The skill wrapper's error message on line 82 is more helpful because it tells the user how to fix the problem, but it would be better to reference or wrap the sentinel error to maintain consistency:

```go
fmt.Fprintf(os.Stderr, "skill: %v (set DISCORD_TOKEN or configure ~/.openclaw/openclaw.json)\n", config.ErrMissingToken)
```

This is a nit, not a blocker.

## Strengths

- **Testable function extraction:** `parseOpenClawConfig` and `resolveToken` are cleanly separated from `main()` and fully testable.
- **Thorough test coverage:** The tests for `parseOpenClawConfig` cover valid JSON, missing file, invalid JSON, missing fields, empty fields, extra fields, null values, empty file, and round-trip verification. The `resolveToken` tests cover env priority, file fallback, no sources, invalid JSON graceful handling, and edge cases.
- **Proper use of `t.Setenv`:** Tests correctly use `t.Setenv` which automatically restores the original value on cleanup, avoiding test pollution.
- **Proper use of `t.TempDir`:** Tests use `t.TempDir()` for file-based test fixtures, which handles cleanup automatically.
- **Error wrapping with %w:** `parseOpenClawConfig` wraps errors with `%w` format verb consistently.
- **Doc comments:** Both exported-level and unexported helper functions have clear documentation.
- **Signal handling:** The `signal.NotifyContext` pattern for graceful shutdown is idiomatic.
- **SKILL.md is well-structured:** Clear usage, arguments, configuration resolution order, and environment variable documentation.

## Verdict Details

**REQUEST_CHANGES** -- The only actionable item is issue #1: add a clarifying comment above the `config.ApplyEnv(&cfg)` call explaining the intentional interaction between `ApplyEnv` (for audio params) and the manual overrides (for token and guildID). The remaining items are nits or awareness flags. Once the comment is added, this is ready to approve.
