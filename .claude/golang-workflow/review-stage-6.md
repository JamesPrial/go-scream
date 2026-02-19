# Code Review: Stage 6 -- OpenClaw Skill Wrapper

**Reviewer:** Go Reviewer Agent
**Files reviewed:**
- `/Users/jamesprial/code/go-scream/cmd/skill/main.go`
- `/Users/jamesprial/code/go-scream/cmd/skill/main_test.go`
- `/Users/jamesprial/code/go-scream/SKILL.md`

**Reference:** `/tmp/scream-reference/scripts/scream.mjs` (Node.js getToken pattern)

---

## Verdict: **APPROVE**

The code is clean, well-documented, idiomatic Go, and the tests are thorough with excellent edge case coverage. No blocking issues found.

---

## Detailed Review

### 1. Token Resolution Logic (`resolveToken` / `parseOpenClawConfig`)

**Correctness: PASS**

The token resolution correctly mirrors the Node.js reference implementation at `/tmp/scream-reference/scripts/scream.mjs` lines 30-44:

| Priority | Node.js reference | Go skill wrapper |
|----------|-------------------|------------------|
| 1 | `process.env.DISCORD_TOKEN` | `os.Getenv("DISCORD_TOKEN")` |
| 2 | `readFileSync(~/.openclaw/openclaw.json)` -> `.channels.discord.token` | `parseOpenClawConfig(openclawPath)` -> `.Channels.Discord.Token` |

Both implementations:
- Check env var first, returning immediately if non-empty
- Fall back to parsing the JSON config file
- Silently ignore file read/parse errors
- Fail if no token is found from either source

The Go implementation splits the concern cleanly into two functions (`parseOpenClawConfig` for file I/O and `resolveToken` for the priority chain), which is a good design improvement over the monolithic Node.js `getToken()`.

### 2. JSON Parsing Edge Cases

**Correctness: PASS**

The `openclawConfig` struct uses nested anonymous structs with proper JSON tags, which correctly handles:
- Full path present with token (`{"channels":{"discord":{"token":"..."}}}`)
- Missing nested fields (Go's `json.Unmarshal` zero-initializes missing struct fields)
- Extra fields in the JSON (silently ignored by default, as documented in the function comment)
- Explicit `null` values (Go deserializes these as zero values for the field type)
- Empty file (returns `json.Unmarshal` error, since empty bytes are not valid JSON)
- Invalid JSON (returns wrapped parse error)

The struct approach is preferable to map-based access because it provides compile-time safety on the field path.

### 3. Error Handling

**Correctness: PASS**

- `parseOpenClawConfig`: Returns wrapped errors using `%w` format verb for both read and parse failures. The error messages include the file path, which aids debugging.
- `resolveToken`: Silently discards file errors, which matches the documented behavior and the Node.js reference (`catch { /* fall through */ }`). This is the correct choice -- the file is a fallback, and its absence should not be an error.
- `main`: Exits with status 1 and a helpful stderr message for missing args, missing token, and invalid config. This matches the Node.js reference behavior (`process.exit(1)`).

### 4. Interaction with `config.ApplyEnv`

**Observation (non-blocking):** There is a subtle ordering interaction worth noting. The `main` function calls:

```go
cfg := config.Default()
config.ApplyEnv(&cfg)     // This sets cfg.Token from DISCORD_TOKEN if set
cfg.Token = token          // This overwrites with resolveToken() result
```

Since `resolveToken()` also reads `DISCORD_TOKEN` as its first priority, the final value of `cfg.Token` will always be correct. However, `ApplyEnv` does redundant work on the token field that is immediately overwritten. This is harmless -- `ApplyEnv` also handles `SCREAM_BACKEND`, `SCREAM_PRESET`, `SCREAM_DURATION`, `SCREAM_VOLUME`, etc., so the call is necessary regardless. The overwrite pattern is consistent with how `cmd/scream/play.go` handles guildID (`cfg.GuildID = args[0]` after `buildConfig` may have set it).

### 5. Code Quality Checklist

- [x] **Package documentation**: Line 1-3 provide a clear package-level doc comment
- [x] **Exported items**: No exported items (all functions are unexported, appropriate for a `main` package)
- [x] **Error wrapping**: Uses `%w` verb in `fmt.Errorf` (lines 35, 40)
- [x] **Nil safety**: No pointer receivers or dereferences that could panic. `json.Unmarshal` on a struct value is nil-safe.
- [x] **Resource cleanup**: `signal.NotifyContext` stop function is deferred (line 98)
- [x] **No unused imports**: All imports are used
- [x] **Naming**: Function names are descriptive (`parseOpenClawConfig`, `resolveToken`); struct names match the domain
- [x] **Consistency**: The arg-parsing pattern (`os.Args` with positional args) and signal-context pattern match `cmd/scream/play.go`

### 6. Test Quality

**22 tests across two function groups -- comprehensive and well-structured.**

**`Test_parseOpenClawConfig_Cases` (table-driven, 6 sub-cases):**
- Valid JSON with token
- Missing file (error path)
- Invalid JSON (error path)
- Missing token field (empty string, no error)
- Empty channels object
- Empty root object

**Additional standalone `parseOpenClawConfig` tests:**
- `Test_parseOpenClawConfig_ValidJSON_StructureVerification` -- round-trip via `json.Marshal`/file write/parse. Validates the struct's JSON tags match expectations.
- `Test_parseOpenClawConfig_ExtraFieldsIgnored` -- ensures extra fields like `guild_id`, `slack`, `version` do not cause errors. Matches the documented behavior.
- `Test_parseOpenClawConfig_EmptyToken` -- explicit empty string token
- `Test_parseOpenClawConfig_EmptyFile` -- empty file is not valid JSON
- `Test_parseOpenClawConfig_NullValues` (table-driven, 3 sub-cases) -- null channels, null discord, null token

**`Test_resolveToken_Cases` (table-driven, 4 sub-cases):**
- Env var takes priority over file
- Empty env falls back to file
- Empty env and no file returns empty
- Env set with no file returns env token

**Additional standalone `resolveToken` tests:**
- `Test_resolveToken_EnvPriority` -- explicit priority verification
- `Test_resolveToken_FallbackToFile` -- fallback path
- `Test_resolveToken_NoSources` -- neither source available
- `Test_resolveToken_InvalidJSON_FallsGracefully` -- broken JSON does not panic
- `Test_resolveToken_EnvOverridesInvalidFile` -- env works even with broken file
- `Test_resolveToken_FileWithEmptyToken` -- file present but token is empty string

**Test quality observations:**
- [x] All tests use `t.TempDir()` for file isolation (auto-cleanup)
- [x] Environment variables are set via `t.Setenv()` (auto-restore)
- [x] Error paths verify both the error and the return value
- [x] Test names follow `TestXxx` / `Test_xxx_Yyy` conventions
- [x] Table tests use `t.Run` with descriptive sub-test names
- [x] No test pollution -- each test case is independent

### 7. SKILL.md Manifest

**Correctness: PASS**

The manifest correctly adapts the Node.js reference:

| Field | Reference (`/tmp/scream-reference/SKILL.md`) | Go version (`/Users/jamesprial/code/go-scream/SKILL.md`) |
|-------|----------------------------------------------|----------------------------------------------------------|
| `name` | `scream` | `scream` |
| `description` | Same trigger phrases | Same trigger phrases |
| `emoji` | `"😱"` | `"😱"` |
| `requires.bins` | `["node"]` | `["skill"]` (correct -- binary name changed) |
| `requires.config` | `["channels.discord.token"]` | `["channels.discord.token"]` |

The Go version uses YAML block style for the metadata (vs inline JSON in the reference), which is equivalent and more readable. The usage section correctly documents the Go binary name (`skill`) with the same positional arg pattern. The documentation of environment variable overrides (`SCREAM_PRESET`, `SCREAM_DURATION`, `SCREAM_VOLUME`, `SCREAM_BACKEND`) accurately reflects what `config.ApplyEnv` supports.

### 8. Minor Observations (Non-blocking)

**8a.** The `channelID` variable is captured but suppressed with `_ = channelID` pending service wiring. This is appropriate for a TODO stub and consistent with `cmd/scream/play.go`.

**8b.** The `main` function does not validate `guildID` beyond checking `len(os.Args) < 2`. In practice, the shell cannot pass an empty string as a positional arg, so `guildID` will always be non-empty. The `config.Validate` function does not check for empty `GuildID` (that check lives in the service layer), which is consistent with the existing architecture where field-presence checks are context-specific.

**8c.** The `openclawPath` construction uses `os.Getenv("HOME")`, which could be empty in unusual environments (e.g., running as a system service without HOME set). The Node.js reference uses `process.env.HOME` identically. Since this is a fallback path and the primary token source is the env var, this is acceptable.

---

## Summary

The skill wrapper is a clean, minimal binary that faithfully ports the Node.js token resolution pattern to idiomatic Go. The code is well-documented, error handling is appropriate (graceful fallback for missing files, hard fail for missing token), and the test suite is thorough with 22 tests covering all branches including null JSON values, empty files, invalid JSON, and priority ordering. The SKILL.md manifest correctly adapts the reference with the updated binary name.

No changes required.
