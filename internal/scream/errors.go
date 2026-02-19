// Package scream provides the service orchestrator for the go-scream bot.
// It wires together audio generation, encoding, and Discord playback.
package scream

import "errors"

// Sentinel errors returned by the scream service.
var (
	// ErrNoPlayer is returned when Play is called without a configured Discord player.
	ErrNoPlayer = errors.New("scream: discord player not configured (missing token?)")

	// ErrUnknownPreset is returned when the configured preset name does not exist.
	ErrUnknownPreset = errors.New("scream: unknown preset name")

	// ErrGenerateFailed is returned when audio generation fails.
	ErrGenerateFailed = errors.New("scream: audio generation failed")

	// ErrEncodeFailed is returned when audio encoding fails.
	ErrEncodeFailed = errors.New("scream: encoding failed")

	// ErrPlayFailed is returned when Discord voice playback fails.
	ErrPlayFailed = errors.New("scream: playback failed")
)
