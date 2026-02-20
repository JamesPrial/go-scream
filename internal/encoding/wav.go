// Package encoding — WAV file encoder.
package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
)

// wavFileHeader is the 44-byte RIFF/WAVE header for a PCM WAV file.
type wavFileHeader struct {
	ChunkID       [4]byte // "RIFF"
	ChunkSize     uint32  // dataSize + 36
	Format        [4]byte // "WAVE"
	Subchunk1ID   [4]byte // "fmt "
	Subchunk1Size uint32  // 16 (PCM)
	AudioFormat   uint16  // 1 (PCM)
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32  // SampleRate * NumChannels * 2
	BlockAlign    uint16  // NumChannels * 2
	BitsPerSample uint16  // 16
	Subchunk2ID   [4]byte // "data"
	Subchunk2Size uint32  // len(pcmData)
}

// Compile-time check that WAVEncoder implements FileEncoder.
var _ FileEncoder = (*WAVEncoder)(nil)

// WAVEncoder encodes raw s16le PCM audio into a WAV file.
type WAVEncoder struct {
	logger *slog.Logger
}

// NewWAVEncoder returns a new WAVEncoder.
func NewWAVEncoder(logger *slog.Logger) *WAVEncoder {
	return &WAVEncoder{logger: logger}
}

// Encode reads s16le PCM data from src and writes a WAV file to dst.
// sampleRate must be positive. channels must be 1 or 2.
// Returns errors wrapping ErrInvalidSampleRate, ErrInvalidChannels, or ErrWAVWrite.
func (e *WAVEncoder) Encode(dst io.Writer, src io.Reader, sampleRate, channels int) error {
	if sampleRate <= 0 {
		return fmt.Errorf("%w: got %d", ErrInvalidSampleRate, sampleRate)
	}
	if channels != 1 && channels != 2 {
		return fmt.Errorf("%w: got %d", ErrInvalidChannels, channels)
	}

	pcmData, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("%w: reading PCM data: %w", ErrWAVWrite, err)
	}

	e.logger.Debug("writing WAV file", "sample_rate", sampleRate, "channels", channels, "data_bytes", len(pcmData))

	dataSize := uint32(len(pcmData))

	hdr := wavFileHeader{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize:     dataSize + 36,
		Format:        [4]byte{'W', 'A', 'V', 'E'},
		Subchunk1ID:   [4]byte{'f', 'm', 't', ' '},
		Subchunk1Size: 16,
		AudioFormat:   1,
		NumChannels:   uint16(channels),
		SampleRate:    uint32(sampleRate),
		ByteRate:      uint32(sampleRate * channels * 2),
		BlockAlign:    uint16(channels * 2),
		BitsPerSample: 16,
		Subchunk2ID:   [4]byte{'d', 'a', 't', 'a'},
		Subchunk2Size: dataSize,
	}

	if err := binary.Write(dst, binary.LittleEndian, &hdr); err != nil {
		return fmt.Errorf("%w: writing header: %w", ErrWAVWrite, err)
	}

	if len(pcmData) > 0 {
		if _, err := dst.Write(pcmData); err != nil {
			return fmt.Errorf("%w: writing PCM data: %w", ErrWAVWrite, err)
		}
	}

	return nil
}
