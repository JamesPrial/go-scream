package discord

import (
	"context"
	"fmt"
	"time"
)

// SilenceFrame is the Opus silence frame sent after audio playback ends.
var SilenceFrame = []byte{0xF8, 0xFF, 0xFE}

// SilenceFrameCount is the number of silence frames sent after playback.
const SilenceFrameCount = 5

// VoicePlayer sends Opus-encoded audio frames to a Discord voice channel.
type VoicePlayer interface {
	Play(ctx context.Context, guildID, channelID string, frames <-chan []byte) error
}

// Player implements VoicePlayer using a Session.
type Player struct {
	session Session
}

// Compile-time interface check.
var _ VoicePlayer = (*Player)(nil)

// NewPlayer returns a VoicePlayer using the provided Session.
func NewPlayer(session Session) *Player {
	return &Player{session: session}
}

// Play joins the specified voice channel, streams all frames from the frames
// channel to Discord, then sends silence frames and disconnects. It returns
// an error if validation fails, the context is already cancelled before
// joining, joining fails, or setting the speaking state fails. If the context
// is cancelled during playback, silence frames are sent and ctx.Err() is
// returned.
func (p *Player) Play(ctx context.Context, guildID, channelID string, frames <-chan []byte) (retErr error) {
	// Validate inputs.
	if guildID == "" {
		return ErrEmptyGuildID
	}
	if channelID == "" {
		return ErrEmptyChannelID
	}
	if frames == nil {
		return ErrNilFrameChannel
	}

	// Check for pre-cancelled context before joining.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Join the voice channel.
	vc, err := p.session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrVoiceJoinFailed, err)
	}
	defer func() {
		if derr := vc.Disconnect(); derr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to disconnect from voice: %w", derr)
		}
	}()

	// Signal that we are speaking.
	if err := vc.Speaking(true); err != nil {
		return fmt.Errorf("%w: %w", ErrSpeakingFailed, err)
	}

	opusSend := vc.OpusSendChannel()

	// Frame loop with double-select pattern for graceful context cancellation.
loop:
	for {
		select {
		case <-ctx.Done():
			silenceCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			sendSilence(silenceCtx, opusSend)
			cancel()
			_ = vc.Speaking(false)
			return ctx.Err()
		case frame, ok := <-frames:
			if !ok {
				break loop
			}
			select {
			case opusSend <- frame:
			case <-ctx.Done():
				silenceCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				sendSilence(silenceCtx, opusSend)
				cancel()
				_ = vc.Speaking(false)
				return ctx.Err()
			}
		}
	}

	// Normal completion: send silence and stop speaking.
	sendSilence(context.Background(), opusSend)

	if err := vc.Speaking(false); err != nil {
		return fmt.Errorf("%w: %w", ErrSpeakingFailed, err)
	}

	return nil
}

// sendSilence sends SilenceFrameCount copies of SilenceFrame to opusSend.
// It returns early if ctx is cancelled, making it safe to call on a full channel.
func sendSilence(ctx context.Context, opusSend chan<- []byte) {
	for i := 0; i < SilenceFrameCount; i++ {
		select {
		case opusSend <- SilenceFrame:
		case <-ctx.Done():
			return
		}
	}
}
