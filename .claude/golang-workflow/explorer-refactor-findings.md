# Refactoring Analysis: go-scream Full Codebase Cleanup

Date: 2026-02-19  
Mode: Aggressive — sweeping structural changes allowed

---

## Affected Files

Files that need refactoring changes (in priority order):

1. `/Users/jamesprial/code/go-scream/internal/discord/player.go`
2. `/Users/jamesprial/code/go-scream/internal/audio/native/layers.go`
3. `/Users/jamesprial/code/go-scream/internal/scream/service.go`
4. `/Users/jamesprial/code/go-scream/internal/scream/resolve.go`
5. `/Users/jamesprial/code/go-scream/cmd/skill/main.go`
6. `/Users/jamesprial/code/go-scream/cmd/scream/service.go`
7. `/Users/jamesprial/code/go-scream/cmd/scream/play.go`
8. `/Users/jamesprial/code/go-scream/cmd/scream/generate.go`
9. `/Users/jamesprial/code/go-scream/internal/audio/params.go`
10. `/Users/jamesprial/code/go-scream/internal/config/validate.go`
11. `/Users/jamesprial/code/go-scream/internal/encoding/opus.go`
12. `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command.go`

---

## Duplication Hotspots

### 1. Generator-selection block duplicated verbatim across both binaries

`cmd/scream/service.go:22-31` and `cmd/skill/main.go:111-121` contain
identical logic:

```go
var gen audio.AudioGenerator
if cfg.Backend == config.BackendFFmpeg {
    g, err := ffmpeg.NewFFmpegGenerator()
    if err != nil { ... }
    gen = g
} else {
    gen = native.NewNativeGenerator()
}
```

The only difference is error handling style (return vs. os.Exit). This
block could be extracted to a function in `internal/scream` (e.g.,
`NewGenerator(cfg config.Config) (audio.AudioGenerator, error)`), or the
skill binary could call `newServiceFromConfig` from a shared internal
function.

### 2. Discord session creation duplicated across both binaries

`cmd/scream/service.go:48-59` and `cmd/skill/main.go:128-141` share the
same pattern:

```go
session, err := discordgo.New("Bot " + cfg.Token)
if err != nil { ... }
if err := session.Open(); err != nil { ... }
// wrap in DiscordGoSession, pass to NewDiscordPlayer
```

This could be extracted to a helper in `internal/discord` (e.g.,
`NewDiscordGoPlayer(token string) (*DiscordPlayer, io.Closer, error)`).

### 3. Signal-context + closer defer pattern duplicated in play.go and generate.go

`cmd/scream/play.go:59-73` and `cmd/scream/generate.go:43-57` both contain:

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

svc, closer, err := newServiceFromConfig(cfg)
if err != nil { return err }
if closer != nil {
    defer func() {
        if cerr := closer.Close(); cerr != nil {
            fmt.Fprintf(os.Stderr, "warning: failed to close session: %v\n", cerr)
        }
    }()
}
```

This 10-line block is identical between both commands. Extract to a helper
such as `runWithService(cfg config.Config, fn func(ctx, svc) error) error`.

### 4. PrimaryScreamLayer and HighShriekLayer are structural duplicates

`internal/audio/native/layers.go:14-51` (PrimaryScreamLayer) and
`internal/audio/native/layers.go:92-129` (HighShriekLayer) are identical
structs with identical constructors and nearly identical `Sample` methods.
The ONLY difference is the coprime constant (137 vs 89) passed to
`seededRandom`. Both have the same fields: `osc`, `seed`, `base`,
`freqRange`, `jump`, `amp`, `rise`, `curStep`, `curFreq`.

These two types can be collapsed into a single `SweepJumpLayer` struct
parameterized by the coprime value:

```go
type SweepJumpLayer struct {
    // ... same fields ...
    coprime int64  // distinguishes PrimaryScream (137) from HighShriek (89)
}
```

This eliminates ~80 lines of near-duplicate code and reduces the struct
count from 5 types to 4.

### 5. Coprime constants are duplicated between native and ffmpeg backends

The coprime values (137, 251, 89, 173) that govern frequency-step RNG
appear in two places with no shared constants:

- `internal/audio/native/layers.go:47,86,125,160`
- `internal/audio/ffmpeg/command.go:67,81,95,110`

If these ever diverge, the two backends will produce different audio from
the same seed. Extract to package-level constants in `internal/audio`
(e.g., `LayerCoprimePrimaryScream = 137`).

---

## Dependency Issues

### Import graph (non-test)

```
cmd/scream  --> internal/{audio,audio/ffmpeg,audio/native,config,discord,encoding,scream}, pkg/version
cmd/skill   --> internal/{audio,audio/ffmpeg,audio/native,config,discord,encoding,scream}
internal/scream     --> internal/{audio,config,discord,encoding}
internal/config     --> internal/audio        [see coupling smell below]
internal/audio/ffmpeg --> internal/audio
internal/audio/native --> internal/audio
internal/audio      --> (stdlib only)
internal/discord    --> (stdlib + discordgo)
internal/encoding   --> (stdlib + gopus + pion)
pkg/version         --> (stdlib only)
```

No import cycles. No package exceeds 10 direct deps.

### config -> audio coupling (moderate smell)

`internal/config/validate.go` imports `internal/audio` solely to call
`audio.AllPresets()` in `isValidPreset()`. This couples the config
package—which should be a pure data layer—to the audio domain. The fix
is to inject the list of valid preset names rather than importing audio:

Option A: Add a `ValidPresets []string` field to `Config` (set from
`audio.AllPresets()` at the call site).  
Option B: Accept a `func() []string` validator closure in `Validate`.  
Option C: Register valid preset names in the config package via an
`RegisterPresets(names []string)` init-time call.

Option A is the most idiomatic for this codebase.

### cmd/skill/main.go is a god function

`cmd/skill/main.go:74-151` is a 78-line `main()` that manually
re-implements everything `newServiceFromConfig` does in `cmd/scream`. This
is the most severe structural duplication. The skill binary should call
`newServiceFromConfig` (or a shared equivalent moved to `internal/scream`)
instead of wiring dependencies by hand.

---

## Interface Issues

### 1. `//nolint:errcheck` violations in player.go (violates project rules)

`internal/discord/player.go:82` and `:94`:

```go
vc.Speaking(false) //nolint:errcheck // best-effort
```

The project memory explicitly prohibits `//nolint` directives. These two
call sites are in the context cancellation path, where the result cannot be
propagated without changing the function signature. The correct fix is to
log the error to stderr (consistent with CLI conventions):

```go
if serr := vc.Speaking(false); serr != nil {
    fmt.Fprintf(os.Stderr, "discord: failed to clear speaking state: %v\n", serr)
}
```

### 2. `reflect` used to normalize typed-nil interface in scream/service.go

`internal/scream/service.go:38-41`:

```go
v := reflect.ValueOf(player)
if v.Kind() == reflect.Ptr && v.IsNil() {
    player = nil
}
```

This is a runtime workaround for a Go interface pitfall. The reflect import
is the only non-stdlib import added solely to patch a caller contract
problem. A cleaner fix is to document and enforce at the call site: callers
must pass an untyped nil (not a typed nil concrete pointer) when no player
is intended. Removing this reflect check eliminates the import entirely.

The alternative—if defensive normalization is desired—is to move it to the
cmd layer where the player is constructed, so it never arrives as a typed
nil in the first place.

### 3. `Service.closer` field is dead production code

`internal/scream/service.go:22` defines `closer io.Closer` and
`service.go:127-134` implements `Service.Close()`. However:

- `closer` is never set by `NewServiceWithDeps` (the only public
  constructor).
- `Service.Close()` is never called from any cmd file.
- `closer` is only set via direct struct field access in tests
  (`service_test.go:700,720,754`).

This means `Service.Close()` is unreachable dead code from production. The
`closer` field and `Close()` method should either be removed, or
`NewServiceWithDeps` should accept an `io.Closer` parameter and call it
when the service is done (making the test pattern legitimate).

### 4. `NoiseBurstLayer` accepts an unused `sampleRate int` parameter

`internal/audio/native/layers.go:143`:

```go
func NewNoiseBurstLayer(p audio.LayerParams, noise audio.NoiseParams, sampleRate int) *NoiseBurstLayer {
```

The `sampleRate` argument is never used inside the function body. Remove it
from the signature and update `buildLayers` in
`internal/audio/native/generator.go:104`.

### 5. `LayerParams.Amplitude` is unused for noise layers

In every preset and in `Randomize()`, `LayerNoiseBurst` sets both
`Layers[3].Amplitude` and `Noise.BurstAmp` to the same value, and
`LayerBackgroundNoise` sets both `Layers[4].Amplitude` and
`Noise.FloorAmp` to the same value. The native backend uses `noise.BurstAmp`
/ `noise.FloorAmp` (ignoring `p.Amplitude`), and the ffmpeg backend does
the same. The `Amplitude` field on noise layers is dead data that creates a
maintenance hazard (they could drift apart). Either:

- Remove `Amplitude` from the noise layers and update all presets +
  `Randomize`; or
- Have the native constructors read `p.Amplitude` (consistent with other
  layers) and eliminate the duplicate `noise.BurstAmp` / `noise.FloorAmp`.

### 6. `Config.Volume` is accepted, validated, and stored—but never applied

`Config.Volume` appears in `Default()`, `Merge()`, `Validate()`,
`ApplyEnv()`, and CLI flags. However, `resolveParams()` in
`internal/scream/resolve.go` never uses `cfg.Volume`. It is never
translated into `FilterParams.VolumeBoostDB` or any multiplier. This
means the `--volume` flag, `SCREAM_VOLUME` env var, and `volume:` YAML key
are all silently ignored at audio generation time.

Fix: either apply `cfg.Volume` as a multiplier over the preset/random
`VolumeBoostDB` in `resolveParams`, or remove the field with a deprecation
note in SKILL.md.

---

## Code Smells

### Functions over 50 lines (non-test):

| File | Line | Length | Function |
|------|------|--------|----------|
| `cmd/skill/main.go` | 74 | 78 lines | `main()` |
| `internal/discord/player.go` | 39 | 70 lines | `(*DiscordPlayer).Play()` |
| `internal/encoding/opus.go` | 43 | 87 lines | `(*GopusFrameEncoder).EncodeFrames()` |
| `internal/audio/ffmpeg/command.go` | 51 | 78 lines | `layerExpr()` |
| `internal/audio/params.go` | 67 | 74 lines | `Randomize()` |
| `internal/encoding/ogg.go` | 35 | 62 lines | `(*OGGEncoder).Encode()` |

**Priority targets:**

- `cmd/skill/main.go:main()` — 78 lines, manually re-implements service
  wiring. Refactor by calling a shared constructor.
- `internal/encoding/opus.go:EncodeFrames()` — 87 lines. The partial-frame
  and full-frame encode branches duplicate the encode+send logic (lines
  97-107 vs 114-122). Extract a `encodeAndSend(encoder, pcmBuf, frameCh)
  error` helper to eliminate ~10 lines of duplication inside this function.
- `internal/audio/ffmpeg/command.go:layerExpr()` — 78 lines, a large
  switch. Acceptable given it dispatches 5 distinct cases, but the
  `LayerPrimaryScream` and `LayerHighShriek` cases share 90% of their
  bodies (same `fmt.Sprintf` template, different coprime). Extract a shared
  format string or helper.

### Hardcoded `"48000"` constant in ffmpeg/command.go

`internal/audio/ffmpeg/command.go:54`:
```go
sampleRate := "48000" // aevalsrc uses its own sample rate; use a constant for the random seeding
```

This ignores `params.SampleRate` entirely. The comment is accurate but the
intent is subtle. A named constant (`const aevalsrcSeedSampleRate = 48000`)
with an explanatory comment would be clearer and more searchable.

### rawConfig duplication with Config in config/config.go

`internal/config/config.go` defines both `Config` (lines 33-44) and
`rawConfig` (lines 49-60) with identical fields except `Duration`. The
`UnmarshalYAML` method (lines 65-91) manually copies every field. As new
config fields are added, both structs and the copy loop must be kept in
sync—a classic maintenance hazard. Consider replacing `rawConfig` with
an embedded approach or using a custom `Duration` type that implements
`yaml.Unmarshaler`.

---

## Existing Test Coverage

### Test files (20 total)

```
/Users/jamesprial/code/go-scream/cmd/skill/main_test.go
/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/command_test.go
/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator_test.go
/Users/jamesprial/code/go-scream/internal/audio/native/filters_test.go
/Users/jamesprial/code/go-scream/internal/audio/native/generator_test.go
/Users/jamesprial/code/go-scream/internal/audio/native/layers_test.go
/Users/jamesprial/code/go-scream/internal/audio/native/oscillator_test.go
/Users/jamesprial/code/go-scream/internal/audio/params_test.go
/Users/jamesprial/code/go-scream/internal/audio/presets_test.go
/Users/jamesprial/code/go-scream/internal/config/config_test.go
/Users/jamesprial/code/go-scream/internal/config/load_test.go
/Users/jamesprial/code/go-scream/internal/config/validate_test.go
/Users/jamesprial/code/go-scream/internal/discord/channel_test.go
/Users/jamesprial/code/go-scream/internal/discord/player_test.go
/Users/jamesprial/code/go-scream/internal/encoding/encoder_test.go
/Users/jamesprial/code/go-scream/internal/encoding/ogg_test.go
/Users/jamesprial/code/go-scream/internal/encoding/opus_test.go
/Users/jamesprial/code/go-scream/internal/encoding/wav_test.go
/Users/jamesprial/code/go-scream/internal/scream/service_test.go
/Users/jamesprial/code/go-scream/pkg/version/version_test.go
```

### Coverage gaps

| Gap | Severity |
|-----|----------|
| `cmd/scream` has ZERO test files (`main.go`, `flags.go`, `service.go`, `play.go`, `generate.go`, `presets.go`) | High |
| `cmd/scream/service.go:newServiceFromConfig` — the factory function that wires all dependencies — is completely untested | High |
| `internal/scream/service.go:Service.Close()` is tested only via direct struct field mutation (`svc.closer = mc`), not through the constructor API | Medium |
| `internal/audio/params.go:Randomize()` is tested for output ranges but not for the `seed=0` → time-based seed path | Low |
| `internal/discord/session.go` (the `DiscordGoSession` and `DiscordGoVoiceConn` adapters) has no unit tests | Low — requires discordgo mocking |

---

## Suggested Refactoring Priority

### P0 — Correctness / Rule Violations (do first)

1. **Remove `//nolint:errcheck` in `internal/discord/player.go:82,94`**  
   Replace with stderr logging. Violates explicit project rules.

2. **Apply `Config.Volume` in `internal/scream/resolve.go`**  
   The `--volume` flag / `SCREAM_VOLUME` / `volume:` YAML key are silently
   no-ops. This is a bug-class issue: users set a value and nothing happens.
   Fix in `resolveParams`: `params.Filter.VolumeBoostDB *= cfg.Volume` (or
   an appropriate linear scaling).

### P1 — Structural Duplication (highest ROI)

3. **Collapse `PrimaryScreamLayer` and `HighShriekLayer` in `layers.go`**  
   They are the same struct. Merge into `SweepJumpLayer` with a `coprime`
   field. Eliminates ~80 lines, tested by existing layer tests.

4. **Extract shared generator + session wiring from both cmd binaries**  
   Move the generator-selection block and Discord session construction to
   helper functions (possibly in `internal/scream` or a new
   `internal/wiring` package). This removes ~50 lines of duplication
   between `cmd/scream/service.go` and `cmd/skill/main.go`.

5. **Extract signal-context + closer defer from `play.go` and `generate.go`**  
   Both commands repeat the same 10-line setup. A `runWithService` helper
   in `cmd/scream` eliminates the duplication.

### P2 — Interface / API Cleanup

6. **Remove or expose `Service.closer` / `Service.Close()`**  
   The `closer` field is never set in production. Either add `io.Closer` as
   a parameter to `NewServiceWithDeps` (clean, testable), or delete
   `Close()` and the field entirely. Stop requiring test code to reach into
   unexported struct fields.

7. **Remove unused `sampleRate` parameter from `NewNoiseBurstLayer`**  
   One-line fix; eliminates a misleading API surface.

8. **Resolve `LayerParams.Amplitude` for noise layers**  
   Decide and enforce one source of truth: either `LayerParams.Amplitude`
   drives noise amplitude (consistent with other layers) and
   `NoiseParams.BurstAmp/FloorAmp` is removed, or vice versa. Current state
   is two fields that happen to agree but could silently diverge.

9. **Remove `reflect` from `internal/scream/service.go`**  
   Move typed-nil normalization to the call site in `cmd/scream/service.go`
   and `cmd/skill/main.go`, or enforce the untyped nil contract via a
   comment and `var _ discord.VoicePlayer = (*DiscordPlayer)(nil)` guard.

### P3 — Coupling / Layering

10. **Break `config` -> `audio` coupling in `config/validate.go`**  
    Pass valid preset names as a slice parameter (or via an option) rather
    than importing the audio package. Keeps config as a pure data layer.

11. **Extract coprime constants to `internal/audio`**  
    Define `LayerCoprimePrimary = 137`, `LayerCoprimeHarmonicSweep = 251`,
    etc. and use them in both backends. Prevents silent divergence.

### P4 — Test Coverage

12. **Add tests for `cmd/scream` package**  
    Specifically `newServiceFromConfig` (the untested wiring function) and
    the `buildConfig` flag-precedence logic. Use `cobra.Command.Execute()`
    with a captured output writer.

13. **Replace `svc.closer = mc` test pattern with a proper constructor**  
    Once P2-item 6 is resolved, the test code becomes idiomatic.

---

## Dependency Graph (ASCII)

```
pkg/version
    ^
    |
cmd/scream ──────────────────────────────────────────────────┐
    |                                                        |
cmd/skill ───────────────────────────────────────────────┐  |
    |                                                     |  |
    └──► internal/scream ──► internal/audio (interface)   |  |
    |         |              internal/audio/presets       |  |
    |         |              internal/audio/params        |  |
    |         ├──► internal/config ──► internal/audio*    |  |
    |         ├──► internal/discord                       |  |
    |         └──► internal/encoding                      |  |
    |                                                     |  |
    ├──► internal/audio/native ──► internal/audio         |  |
    └──► internal/audio/ffmpeg ──► internal/audio         |  |
                                                          |  |
    * coupling smell: config should not import audio      |  |
```

The import graph is clean (no cycles) but the `config -> audio` edge and
the duplication in `cmd/scream` vs `cmd/skill` wiring are the two
structural concerns worth fixing in an aggressive cleanup.
