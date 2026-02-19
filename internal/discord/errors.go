// Package discord provides Discord voice channel integration for audio playback.
package discord

import "errors"

// Sentinel errors returned by discord package functions.
var (
	ErrVoiceJoinFailed    = errors.New("discord: failed to join voice channel")
	ErrSpeakingFailed     = errors.New("discord: failed to set speaking state")
	ErrNoPopulatedChannel = errors.New("discord: no populated voice channel found")
	ErrGuildStateFailed   = errors.New("discord: failed to retrieve guild state")
	ErrEmptyGuildID       = errors.New("discord: guild ID must not be empty")
	ErrEmptyChannelID     = errors.New("discord: channel ID must not be empty")
	ErrNilFrameChannel    = errors.New("discord: frame channel must not be nil")
)
