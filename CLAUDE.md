# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build ./...                          # Build all packages
go build ./cmd/scream                   # Build CLI binary
go build ./cmd/skill                    # Build OpenClaw skill binary
go test ./...                           # Run all tests
go test -race ./...                     # Run tests with race detector
go test -v ./internal/scream/...        # Run tests for a specific package
go test -run TestGenerator ./internal/audio/native/           # Run a single test
go test -bench=. -benchmem ./...        # Run benchmarks
go vet ./...                            # Static analysis
staticcheck ./...                       # Extended linting (if installed)
```

**CGO required:** `layeh.com/gopus` links against libopus. Install it first:
```bash
brew install opus          # macOS
sudo apt-get install libopus-dev  # Ubuntu/Debian
```

**FFmpeg tests:** ~18 tests are skipped when `ffmpeg` is not on `$PATH`. The native backend requires no external tools.

**Docker:** Multi-arch images (amd64 + arm64) are built by CI (`.github/workflows/docker-publish.yml`) and published to `ghcr.io/jamesprial/go-scream`.

## Architecture

Two binaries consume a shared service layer:

```
cmd/scream (Cobra CLI)  ─┐
cmd/skill  (OpenClaw)   ─┤
                         ├── internal/app        (wiring: NewGenerator, NewFileEncoder, NewDiscordDeps)
                         ├── internal/scream     (Service orchestrator: Play, Generate)
                         ├── internal/audio      (Generator interface, params, presets, coprime constants)
                         │   ├── native          (pure-Go PCM synthesis: 5 layer types + filters)
                         │   └── ffmpeg          (subprocess-based synthesis)
                         ├── internal/encoding   (OpusFrameEncoder, FileEncoder: OGG, WAV)
                         ├── internal/discord    (Session/VoicePlayer interfaces, Player)
                         ├── internal/config     (Config struct, YAML/env/flag loading, validation)
                         └── pkg/version         (build-time version info)
```

**Data flow:** `Config` → `audio.Generator` produces s16le PCM → `encoding.OpusFrameEncoder` produces Opus frames → either `discord.Player` streams to Discord or `encoding.FileEncoder` writes OGG/WAV.

**Dependency injection:** All components wired via constructor injection in `internal/app/wire.go`. `scream.NewServiceWithDeps` accepts interfaces, making the service fully testable with mocks. Every constructor takes `*slog.Logger` as a parameter — use `slog.New(slog.NewTextHandler(io.Discard, nil))` in tests.

**Config cascade:** Defaults → YAML file → env vars (`SCREAM_*`, `DISCORD_TOKEN`) → CLI flags. Note: `internal/config` has a custom `UnmarshalYAML` that parses Go duration strings (e.g., `"5s"`, `"500ms"`) manually.

**Structured logging:** `log/slog` with `TextHandler` writing to stderr. Log level resolved by `config.ParseLogLevel(cfg)`: explicit `LogLevel` field wins, then `Verbose` bool → info, default → warn. CLI exposes `--log-level` flag and `SCREAM_LOG_LEVEL` env var.

## Key Conventions

- **No `//nolint` directives.** Fix lint issues properly: use `_ =` for intentionally discarded errors, named returns (`retErr error`) for deferred cleanup in library code, `slog.Warn` for deferred cleanup in CLI code.
- **Interfaces at consumption point.** `audio.Generator` is in `internal/audio`, consumed by `internal/scream`. Implementations are in sub-packages.
- **Compile-time interface checks:** `var _ audio.Generator = (*Generator)(nil)` in each implementation file.
- **Sentinel errors per package** (e.g., `config.ErrInvalidBackend`, `discord.ErrNoVoiceChannel`), wrapped with `%w`.
- **Context propagation throughout.** All long-running operations accept `context.Context` and honor cancellation.
- **Table-driven tests** with standard `testing` package (no test frameworks). Mocks are hand-rolled locally in test files with `sync.Mutex`-protected call tracking — no mock libraries.

## Package Notes

- `internal/config` must **not** import `internal/audio` (decoupled). When adding a preset to `audio/presets.go`, you must also add it to the `knownPresets` slice in `config/validate.go`.
- `internal/app` is imported only by `cmd/` binaries — never by other `internal/` packages.
- Native audio layers use coprime constants from `audio.Coprime*` shared between native and ffmpeg backends.
