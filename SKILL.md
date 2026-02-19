---
name: scream
description: "Join a Discord voice channel and play a scream sound. Use when a user says 'scream', 'scream in voice', 'join voice and scream', 'AAAA', 'go scream', or asks you to make noise in a voice channel."
metadata:
  openclaw:
    emoji: "😱"
    requires:
      bins:
        - skill
      config:
        - channels.discord.token
---

# Scream

A Discord voice bot skill that generates unique synthetic screams.

## Usage

```
skill <guildId> [channelId]
```

## Arguments

- `guildId` (required): The Discord guild (server) ID
- `channelId` (optional): The voice channel ID. If omitted, auto-detects the first populated voice channel.

## Configuration

Token is resolved in order:
1. `DISCORD_TOKEN` environment variable
2. `~/.openclaw/openclaw.json` → `.channels.discord.token`

## Audio Parameters

Override via environment variables:
- `SCREAM_PRESET` — Preset name (classic, whisper, death-metal, glitch, banshee, robot)
- `SCREAM_DURATION` — Duration (e.g., "3s", "500ms")
- `SCREAM_VOLUME` — Volume 0.0–1.0
- `SCREAM_BACKEND` — Audio backend (native or ffmpeg)
