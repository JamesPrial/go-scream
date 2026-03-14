package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// silenceFrame is the Opus silence frame sent after audio playback ends.
var silenceFrame = []byte{0xF8, 0xFF, 0xFE}

// silenceFrameCount is the number of silence frames sent after playback.
const silenceFrameCount = 5

// VoicePlayer sends Opus-encoded audio frames to a Discord voice channel.
type VoicePlayer interface {
	Play(ctx context.Context, guildID, channelID string, frames <-chan []byte) error
}

// Player implements VoicePlayer using a Session.
type Player struct {
	session Session
	logger  *slog.Logger
}

// Compile-time interface check.
var _ VoicePlayer = (*Player)(nil)

// NewPlayer returns a VoicePlayer using the provided Session and logger.
func NewPlayer(session Session, logger *slog.Logger) *Player {
	return &Player{session: session, logger: logger}
}

// Play joins the specified voice channel, streams all frames from the frames
// channel to Discord, then sends silence frames and disconnects. It returns
// an error if validation fails, the context is already cancelled before
// joining, joining fails, or setting the speaking state fails. If the context
// is cancelled during playback, silence frames are sent and ctx.Err() is
// returned. If the underlying voice channel is lost due to an encryption
// failure (Discord close codes 4016/4017), ErrEncryptionFailed is returned.
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

	p.logger.Debug("joining voice channel", "guild", guildID, "channel", channelID)

	// Join the voice channel.
	vc, err := p.session.ChannelVoiceJoin(ctx, guildID, channelID, false, true)
	if err != nil {
		if isEncryptionError(err) {
			return fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
		}
		return fmt.Errorf("%w: %w", ErrVoiceJoinFailed, err)
	}
	defer func() {
		// Use context.Background() because the caller's ctx may be cancelled.
		if derr := vc.Disconnect(context.Background()); derr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to disconnect from voice: %w", derr)
		}
	}()

	// Signal that we are speaking.
	if err := vc.Speaking(true); err != nil {
		return fmt.Errorf("%w: %w", ErrSpeakingFailed, err)
	}

	p.logger.Debug("voice channel joined, sending frames")

	opusSend := vc.OpusSendChannel()
	start := time.Now()
	var frameCount int

	// Frame loop with double-select pattern for graceful context cancellation.
loop:
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("playback cancelled", "frames_sent", frameCount, "elapsed", time.Since(start))
			cancelPlayback(p.logger, vc, opusSend)
			return ctx.Err()
		case frame, ok := <-frames:
			if !ok {
				break loop
			}
			frameCount++
			select {
			case opusSend <- frame:
			case <-ctx.Done():
				p.logger.Info("playback cancelled", "frames_sent", frameCount, "elapsed", time.Since(start))
				cancelPlayback(p.logger, vc, opusSend)
				return ctx.Err()
			}
		}
	}

	p.logger.Info("playback complete", "frames_sent", frameCount, "elapsed", time.Since(start))

	// Normal completion: send silence and stop speaking.
	sendSilence(context.Background(), opusSend)

	if err := vc.Speaking(false); err != nil {
		return fmt.Errorf("%w: %w", ErrSpeakingFailed, err)
	}

	return nil
}

// isEncryptionError reports whether err indicates a Discord DAVE E2EE failure.
// It returns true when the error message contains the Discord close codes
// 4016 (failed to decrypt) or 4017 (failed to encrypt).
func isEncryptionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "4016") || strings.Contains(msg, "4017")
}

// cancelPlayback sends silence frames with a timeout and stops speaking.
// Used when context cancellation interrupts playback. Silence is best-effort.
func cancelPlayback(logger *slog.Logger, vc VoiceConn, opusSend chan<- []byte) {
	logger.Debug("playback cancelled, sending silence")
	silenceCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	sendSilence(silenceCtx, opusSend)
	cancel()
	_ = vc.Speaking(false)
}

// sendSilence sends silenceFrameCount copies of silenceFrame to opusSend.
// It returns early if ctx is cancelled, making it safe to call on a full channel.
func sendSilence(ctx context.Context, opusSend chan<- []byte) {
	for range silenceFrameCount {
		select {
		case opusSend <- silenceFrame:
		case <-ctx.Done():
			return
		}
	}
}
