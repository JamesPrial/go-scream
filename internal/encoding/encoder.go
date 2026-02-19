// Package encoding provides audio encoding utilities for the go-scream project.
// It supports WAV and OGG/Opus output formats from raw PCM (s16le) input.
package encoding

import (
	"errors"
	"io"
)

// Opus encoding constants.
const (
	// OpusFrameSamples is the number of samples per channel per Opus frame (20ms at 48kHz).
	OpusFrameSamples = 960

	// MaxOpusFrameBytes is the maximum size in bytes of an encoded Opus frame.
	MaxOpusFrameBytes = 3840

	// OpusBitrate is the default Opus encoding bitrate in bits per second.
	OpusBitrate = 64000
)

// OGG/RTP constants used when writing Opus frames into an OGG container.
const (
	// oggPayloadType is the RTP payload type for Opus audio as registered
	// with IANA (dynamic payload type 111).
	oggPayloadType = 111

	// oggSSRC is the fixed RTP synchronisation source identifier used for
	// single-stream OGG output.
	oggSSRC = 1
)

// Sentinel errors for the encoding package.
var (
	// ErrInvalidSampleRate is returned when the sample rate is not a positive value
	// supported by the target encoder.
	ErrInvalidSampleRate = errors.New("encoding: sample rate must be positive")

	// ErrInvalidChannels is returned when the channel count is not 1 or 2.
	ErrInvalidChannels = errors.New("encoding: channels must be 1 or 2")

	// ErrOpusEncode is returned when Opus encoding fails.
	ErrOpusEncode = errors.New("encoding: opus encoding failed")

	// ErrWAVWrite is returned when writing WAV output fails.
	ErrWAVWrite = errors.New("encoding: WAV write failed")

	// ErrOGGWrite is returned when writing OGG output fails.
	ErrOGGWrite = errors.New("encoding: OGG write failed")
)

// OpusFrameEncoder encodes raw PCM audio into a stream of Opus frames.
type OpusFrameEncoder interface {
	// EncodeFrames reads s16le PCM data from src and sends encoded Opus frames
	// on the returned channel. Any error (including nil at completion) is sent
	// on the error channel. Both channels are closed after completion.
	EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error)
}

// FileEncoder encodes raw PCM audio into a container format written to dst.
type FileEncoder interface {
	// Encode reads s16le PCM data from src and writes the encoded output to dst.
	Encode(dst io.Writer, src io.Reader, sampleRate, channels int) error
}

