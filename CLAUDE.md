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
go test -run TestGenerator_Classic ./internal/audio/native/  # Run a single test
go test -bench=. -benchmem ./...        # Run benchmarks
go vet ./...                            # Static analysis
staticcheck ./...                       # Extended linting (if installed)
```

**CGO required:** `layeh.com/gopus` links against libopus. Ensure libopus-dev (or equivalent) is installed.

**FFmpeg tests:** ~18 tests are skipped when `ffmpeg` is not on `$PATH`. The native backend requires no external tools.

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

**Dependency injection:** All components wired via constructor injection in `internal/app/wire.go`. `scream.NewServiceWithDeps` accepts interfaces, making the service fully testable with mocks.

**Config cascade:** Defaults → YAML file → env vars (`SCREAM_*`, `DISCORD_TOKEN`) → CLI flags.

## Key Conventions

- **No `//nolint` directives.** Fix lint issues properly: use `_ =` for intentionally discarded errors, named returns (`retErr error`) for deferred cleanup in library code, stderr warnings for deferred cleanup in CLI code.
- **Interfaces at consumption point.** `audio.Generator` is in `internal/audio`, consumed by `internal/scream`. Implementations are in sub-packages.
- **Compile-time interface checks:** `var _ audio.Generator = (*Generator)(nil)` in each implementation file.
- **Sentinel errors per package** (e.g., `config.ErrInvalidBackend`, `discord.ErrNoVoiceChannel`), wrapped with `%w`.
- **Context propagation throughout.** All long-running operations accept `context.Context` and honor cancellation.
- **Table-driven tests** with standard `testing` package (no test frameworks).

## Package Notes

- `internal/config` must **not** import `internal/audio` (decoupled; preset names are maintained as a local list with a sync comment).
- `internal/app` is imported only by `cmd/` binaries — never by other `internal/` packages.
- Native audio layers use coprime constants from `audio.Coprime*` shared between native and ffmpeg backends.
