// Package encoding — Opus frame encoder using layeh.com/gopus.
package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"

	"layeh.com/gopus"
)

// validOpusSampleRates holds the sample rates supported by the Opus codec.
var validOpusSampleRates = map[int]bool{
	8000:  true,
	12000: true,
	16000: true,
	24000: true,
	48000: true,
}

// Compile-time check that GopusFrameEncoder implements OpusFrameEncoder.
var _ OpusFrameEncoder = (*GopusFrameEncoder)(nil)

// GopusFrameEncoder encodes raw s16le PCM audio into Opus frames using
// the gopus binding for libopus.
type GopusFrameEncoder struct {
	bitrate int
	logger  *slog.Logger
}

// NewGopusFrameEncoder returns a GopusFrameEncoder using the default Opus bitrate.
func NewGopusFrameEncoder(logger *slog.Logger) *GopusFrameEncoder {
	return &GopusFrameEncoder{bitrate: OpusBitrate, logger: logger}
}

// NewGopusFrameEncoderWithBitrate returns a GopusFrameEncoder using the specified bitrate.
func NewGopusFrameEncoderWithBitrate(bitrate int, logger *slog.Logger) *GopusFrameEncoder {
	return &GopusFrameEncoder{bitrate: bitrate, logger: logger}
}

// sendValidationError closes frameCh, sends err on errCh, and closes errCh in a
// new goroutine. It is used to report validation failures before the main encode
// goroutine starts.
func sendValidationError(frameCh chan []byte, errCh chan error, err error) {
	go func() {
		close(frameCh)
		errCh <- err
		close(errCh)
	}()
}

// encodeFrame converts pcmBuf to int16 samples, encodes them with encoder, and
// sends the resulting Opus packet on frameCh. It returns the encode error, or
// nil on success. The caller is responsible for sending the error on errCh.
func encodeFrame(encoder *gopus.Encoder, pcmBuf []byte, samples []int16, frameCh chan<- []byte) error {
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcmBuf[i*2:]))
	}
	encoded, err := encoder.Encode(samples, OpusFrameSamples, MaxOpusFrameBytes)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrOpusEncode, err)
	}
	frameCh <- encoded
	return nil
}

// EncodeFrames reads s16le PCM data from src and encodes it as Opus frames.
// Each frame is sent on the returned frame channel. Any error (or nil on clean
// completion) is sent on the error channel. Both channels are closed when done.
//
// Valid sample rates for Opus: 8000, 12000, 16000, 24000, 48000.
// Valid channel counts: 1 or 2.
func (e *GopusFrameEncoder) EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error) {
	frameCh := make(chan []byte, 50)
	errCh := make(chan error, 1)

	if !validOpusSampleRates[sampleRate] {
		sendValidationError(frameCh, errCh, fmt.Errorf("%w: got %d, must be one of 8000,12000,16000,24000,48000", ErrInvalidSampleRate, sampleRate))
		return frameCh, errCh
	}
	if channels != 1 && channels != 2 {
		sendValidationError(frameCh, errCh, fmt.Errorf("%w: got %d", ErrInvalidChannels, channels))
		return frameCh, errCh
	}

	go func() {
		defer close(frameCh)
		defer close(errCh)

		e.logger.Debug("encoding opus frames", "sample_rate", sampleRate, "channels", channels, "bitrate", e.bitrate)

		encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
		if err != nil {
			errCh <- fmt.Errorf("%w: creating encoder: %w", ErrOpusEncode, err)
			return
		}

		encoder.SetBitrate(e.bitrate)

		// frameBytes is the number of PCM bytes per Opus frame.
		frameBytes := OpusFrameSamples * channels * 2
		// pcmBuf is zeroed at the start of each iteration to ensure correct
		// zero-padding of partial frames.
		pcmBuf := make([]byte, frameBytes)
		// samples is pre-allocated to avoid a heap allocation on every frame.
		samples := make([]int16, OpusFrameSamples*channels)

		var frameCount int

		for {
			// Zero the buffer before each read so that partial frames are
			// automatically zero-padded.
			clear(pcmBuf)

			_, readErr := io.ReadFull(src, pcmBuf)
			if readErr == io.EOF {
				// No bytes read; we are done.
				break
			}
			if readErr == io.ErrUnexpectedEOF {
				// Partial frame: data was read into pcmBuf[0:n]; the rest was
				// already zeroed above. Encode and then stop.
				if encErr := encodeFrame(encoder, pcmBuf, samples, frameCh); encErr != nil {
					errCh <- encErr
					return
				}
				frameCount++
				break
			}
			if readErr != nil {
				errCh <- fmt.Errorf("%w: reading PCM: %w", ErrOpusEncode, readErr)
				return
			}

			// Full frame read successfully.
			if encErr := encodeFrame(encoder, pcmBuf, samples, frameCh); encErr != nil {
				errCh <- encErr
				return
			}
			frameCount++
		}

		e.logger.Debug("opus encoding complete", "frames", frameCount)
		errCh <- nil
	}()

	return frameCh, errCh
}
