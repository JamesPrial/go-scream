# API Changes -- go-scream Refactor

**Date:** 2026-02-19
**Baseline Commit:** 008595f Initial commit: go-scream audio generation tool
**Mode:** Aggressive

---

## API Changes Table

### Stage 1: Remove `//nolint` directives

No API changes. Internal implementation only.

### Stage 2: Rename stuttered types

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| RENAMED | `internal/audio` | `AudioGenerator` | `Generator` | Yes |
| RENAMED | `internal/audio/native` | `NativeGenerator` | `Generator` | Yes |
| RENAMED | `internal/audio/native` | `NewNativeGenerator` | `NewGenerator` | Yes |
| RENAMED | `internal/audio/ffmpeg` | `FFmpegGenerator` | `Generator` | Yes |
| RENAMED | `internal/audio/ffmpeg` | `NewFFmpegGenerator` | `NewGenerator` | Yes |
| RENAMED | `internal/audio/ffmpeg` | `NewFFmpegGeneratorWithPath` | `NewGeneratorWithPath` | Yes |
| RENAMED | `internal/discord` | `DiscordGoSession` | `GoSession` | Yes |
| RENAMED | `internal/discord` | `DiscordGoVoiceConn` | `GoVoiceConn` | Yes |
| RENAMED | `internal/discord` | `DiscordPlayer` | `Player` | Yes |
| RENAMED | `internal/discord` | `NewDiscordPlayer` | `NewPlayer` | Yes |

### Stage 3: Remove `reflect` dependency and dead `closer` field

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| DELETED | `internal/scream` | `Service.closer` (unexported) | -- | No (unexported) |
| DELETED | `internal/scream` | `Service.Close()` | -- | Yes |

### Stage 4: Remove dead code and merge files

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| DELETED | `internal/encoding` | `pcmBytesToInt16` (unexported) | -- | No (unexported) |
| MOVED | `internal/scream` | `resolveParams` (in resolve.go) | `resolveParams` (in service.go) | No (unexported) |

### Stage 5: Consolidate native layer types and extract coprime constants

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| RENAMED | `internal/audio/native` | `PrimaryScreamLayer` | `SweepJumpLayer` | Yes |
| RENAMED | `internal/audio/native` | `HighShriekLayer` | `SweepJumpLayer` (same type) | Yes |
| NEW | `internal/audio` | -- | `CoprimePrimaryScream` (const) | N/A |
| NEW | `internal/audio` | -- | `CoprimeHarmonicSweep` (const) | N/A |
| NEW | `internal/audio` | -- | `CoprimeHighShriek` (const) | N/A |
| NEW | `internal/audio` | -- | `CoprimeNoiseBurst` (const) | N/A |
| SIGNATURE | `internal/audio/native` | `NewNoiseBurstLayer(p, noise, sampleRate)` | `NewNoiseBurstLayer(p, noise)` | Yes |

### Stage 6: Fix Config.Volume silent no-op

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| BEHAVIOR | `internal/scream` | `resolveParams` (ignores Volume) | `resolveParams` (applies Volume) | No (bug fix) |

### Stage 7: Break config->audio coupling

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| SIGNATURE | `internal/config` | `Validate(cfg)` | `Validate(cfg)` with preset injection | Yes |
| DELETED | `internal/config` | import `internal/audio` | -- | N/A (import removed) |

### Stage 8: Deduplicate service wiring

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| NEW | `internal/app` (NEW) | -- | `NewGenerator` | N/A |
| NEW | `internal/app` (NEW) | -- | `NewFileEncoder` | N/A |
| NEW | `internal/app` (NEW) | -- | `NewDiscordDeps` | N/A |
| SIGNATURE | `cmd/scream` | `newServiceFromConfig` | `newServiceFromConfig` (internal wiring replaced) | No (unexported) |

### Stage 9: Extract signal-context + closer defer pattern

| Change Type | Package | Old Symbol | New Symbol | Breaking? |
|-------------|---------|-----------|------------|-----------|
| NEW | `cmd/scream` | -- | `runWithService` (unexported helper) | No (unexported) |

---

## Symbols With No Changes

All significant exported symbols in `internal/audio` (params, presets, errors), `internal/encoding` (interfaces, encoders, errors), `internal/discord` (interfaces, VoiceState, FindPopulatedChannel, errors), `internal/scream` (Service struct minus closer/Close, NewServiceWithDeps, Play, Generate, ListPresets, errors), and `pkg/version` (all) remain unchanged. `internal/config` types remain but validation API changes (Stage 7).
