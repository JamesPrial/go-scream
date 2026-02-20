// Package encoding — OGG/Opus file encoder.
package encoding

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

// OGGEncoder encodes raw s16le PCM audio into an OGG/Opus container.
type OGGEncoder struct {
	opus   OpusFrameEncoder
	logger *slog.Logger
}

// NewOGGEncoder returns an OGGEncoder backed by a default GopusFrameEncoder.
func NewOGGEncoder(logger *slog.Logger) *OGGEncoder {
	return &OGGEncoder{opus: NewGopusFrameEncoder(logger), logger: logger}
}

// NewOGGEncoderWithOpus returns an OGGEncoder that uses the provided OpusFrameEncoder.
// It panics if opus is nil.
func NewOGGEncoderWithOpus(opus OpusFrameEncoder, logger *slog.Logger) *OGGEncoder {
	if opus == nil {
		panic("encoding: opus encoder must not be nil")
	}
	return &OGGEncoder{opus: opus, logger: logger}
}

// Encode reads s16le PCM data from src, encodes it as Opus frames, and writes
// an OGG container to dst. Returns errors wrapping ErrOGGWrite on writer
// failures, or the underlying opus error (ErrInvalidSampleRate, ErrInvalidChannels,
// ErrOpusEncode) on encoding failures.
func (e *OGGEncoder) Encode(dst io.Writer, src io.Reader, sampleRate, channels int) (retErr error) {
	e.logger.Debug("writing OGG container", "sample_rate", sampleRate, "channels", channels)

	frameCh, errCh := e.opus.EncodeFrames(src, sampleRate, channels)

	oggWriter, err := oggwriter.NewWith(dst, uint32(sampleRate), uint16(channels))
	if err != nil {
		// Drain channels to avoid goroutine leak.
		for range frameCh {
		}
		<-errCh
		return fmt.Errorf("%w: creating OGG writer: %w", ErrOGGWrite, err)
	}
	defer func() {
		if cerr := oggWriter.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("%w: closing OGG writer: %w", ErrOGGWrite, cerr)
		}
	}()

	var seqNum uint16
	var timestamp uint32
	var oggWriteErr error

	for frame := range frameCh {
		seqNum++
		timestamp += uint32(OpusFrameSamples)

		pkt := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				PayloadType:    oggPayloadType,
				SequenceNumber: seqNum,
				Timestamp:      timestamp,
				SSRC:           oggSSRC,
			},
			Payload: frame,
		}

		if writeErr := oggWriter.WriteRTP(pkt); writeErr != nil {
			// Record the first OGG write error but continue draining frames
			// to avoid blocking the opus goroutine.
			if oggWriteErr == nil {
				oggWriteErr = writeErr
			}
		}
	}

	e.logger.Debug("OGG encoding complete", "frames", seqNum)

	// Collect the opus encoding error.
	opusErr := <-errCh

	// OGG write errors take priority in reporting.
	if oggWriteErr != nil {
		return fmt.Errorf("%w: %w", ErrOGGWrite, oggWriteErr)
	}

	// Propagate opus errors. Return them directly so that errors.Is can find
	// sentinel errors (ErrInvalidSampleRate, ErrInvalidChannels) that were
	// already wrapped by the opus encoder.
	if opusErr != nil {
		return opusErr
	}

	return nil
}
