package encoding

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// ---------------------------------------------------------------------------
// Compile-time interface check
// ---------------------------------------------------------------------------

func TestWAVEncoder_ImplementsFileEncoder(t *testing.T) {
	var _ FileEncoder = (*WAVEncoder)(nil)
}

// ---------------------------------------------------------------------------
// WAV header parsing helpers
// ---------------------------------------------------------------------------

type wavHeader struct {
	ChunkID       [4]byte
	ChunkSize     uint32
	Format        [4]byte
	Subchunk1ID   [4]byte
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte
	Subchunk2Size uint32
}

func parseWAVHeader(t *testing.T, data []byte) wavHeader {
	t.Helper()
	if len(data) < 44 {
		t.Fatalf("WAV data too short: %d bytes, need at least 44", len(data))
	}
	var h wavHeader
	r := bytes.NewReader(data[:44])
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		t.Fatalf("failed to parse WAV header: %v", err)
	}
	return h
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// makePCM creates a PCM byte slice of the given number of samples for the given
// number of channels. Each sample per channel is 2 bytes (s16le).
// Values are sequential so they can be verified on round-trip.
func makePCM(totalSamples, channels int) []byte {
	numBytes := totalSamples * channels * 2
	pcm := make([]byte, numBytes)
	for i := 0; i < numBytes/2; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(i%65536))
	}
	return pcm
}

// encodeWAV is a helper that calls WAVEncoder.Encode and returns the output.
func encodeWAV(t *testing.T, pcm []byte, sampleRate, channels int) []byte {
	t.Helper()
	enc := NewWAVEncoder()
	var buf bytes.Buffer
	err := enc.Encode(&buf, bytes.NewReader(pcm), sampleRate, channels)
	if err != nil {
		t.Fatalf("WAVEncoder.Encode() unexpected error: %v", err)
	}
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestWAVEncoder_HeaderByteLayout(t *testing.T) {
	// 10 stereo samples = 10 * 2 * 2 = 40 bytes of PCM data
	pcm := makePCM(10, 2)
	data := encodeWAV(t, pcm, 48000, 2)

	h := parseWAVHeader(t, data)

	// Check RIFF chunk
	if string(h.ChunkID[:]) != "RIFF" {
		t.Errorf("ChunkID = %q, want %q", h.ChunkID, "RIFF")
	}
	expectedChunkSize := uint32(len(pcm) + 36)
	if h.ChunkSize != expectedChunkSize {
		t.Errorf("ChunkSize = %d, want %d", h.ChunkSize, expectedChunkSize)
	}

	// Check WAVE format
	if string(h.Format[:]) != "WAVE" {
		t.Errorf("Format = %q, want %q", h.Format, "WAVE")
	}

	// Check fmt subchunk
	if string(h.Subchunk1ID[:]) != "fmt " {
		t.Errorf("Subchunk1ID = %q, want %q", h.Subchunk1ID, "fmt ")
	}
	if h.Subchunk1Size != 16 {
		t.Errorf("Subchunk1Size = %d, want 16", h.Subchunk1Size)
	}
	if h.AudioFormat != 1 {
		t.Errorf("AudioFormat = %d, want 1 (PCM)", h.AudioFormat)
	}
	if h.NumChannels != 2 {
		t.Errorf("NumChannels = %d, want 2", h.NumChannels)
	}
	if h.SampleRate != 48000 {
		t.Errorf("SampleRate = %d, want 48000", h.SampleRate)
	}
	if h.ByteRate != 48000*2*2 {
		t.Errorf("ByteRate = %d, want %d", h.ByteRate, 48000*2*2)
	}
	if h.BlockAlign != 4 {
		t.Errorf("BlockAlign = %d, want 4", h.BlockAlign)
	}
	if h.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %d, want 16", h.BitsPerSample)
	}

	// Check data subchunk
	if string(h.Subchunk2ID[:]) != "data" {
		t.Errorf("Subchunk2ID = %q, want %q", h.Subchunk2ID, "data")
	}
	if h.Subchunk2Size != uint32(len(pcm)) {
		t.Errorf("Subchunk2Size = %d, want %d", h.Subchunk2Size, len(pcm))
	}

	// Verify total header is exactly 44 bytes
	if len(data) != 44+len(pcm) {
		t.Errorf("total output = %d bytes, want %d", len(data), 44+len(pcm))
	}
}

func TestWAVEncoder_OutputSize(t *testing.T) {
	tests := []struct {
		name       string
		samples    int
		channels   int
		sampleRate int
	}{
		{"1 sample stereo", 1, 2, 48000},
		{"100 samples stereo", 100, 2, 48000},
		{"48000 samples mono", 48000, 1, 44100},
		{"1000 samples stereo 44100", 1000, 2, 44100},
		{"0 samples", 0, 2, 48000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pcm := makePCM(tt.samples, tt.channels)
			data := encodeWAV(t, pcm, tt.sampleRate, tt.channels)
			expected := 44 + len(pcm)
			if len(data) != expected {
				t.Errorf("output size = %d, want %d (44 header + %d data)", len(data), expected, len(pcm))
			}
		})
	}
}

func TestWAVEncoder_Stereo48kHz(t *testing.T) {
	pcm := makePCM(4800, 2) // 100ms of audio
	data := encodeWAV(t, pcm, 48000, 2)
	h := parseWAVHeader(t, data)

	if h.NumChannels != 2 {
		t.Errorf("NumChannels = %d, want 2", h.NumChannels)
	}
	if h.SampleRate != 48000 {
		t.Errorf("SampleRate = %d, want 48000", h.SampleRate)
	}
	// ByteRate = SampleRate * Channels * BitsPerSample/8 = 48000 * 2 * 2 = 192000
	if h.ByteRate != 192000 {
		t.Errorf("ByteRate = %d, want 192000", h.ByteRate)
	}
	// BlockAlign = Channels * BitsPerSample/8 = 2 * 2 = 4
	if h.BlockAlign != 4 {
		t.Errorf("BlockAlign = %d, want 4", h.BlockAlign)
	}
	if h.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %d, want 16", h.BitsPerSample)
	}
	if h.AudioFormat != 1 {
		t.Errorf("AudioFormat = %d, want 1 (PCM)", h.AudioFormat)
	}
}

func TestWAVEncoder_Mono44100(t *testing.T) {
	pcm := makePCM(4410, 1) // 100ms of mono 44100Hz
	data := encodeWAV(t, pcm, 44100, 1)
	h := parseWAVHeader(t, data)

	if h.NumChannels != 1 {
		t.Errorf("NumChannels = %d, want 1", h.NumChannels)
	}
	if h.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", h.SampleRate)
	}
	// ByteRate = 44100 * 1 * 2 = 88200
	if h.ByteRate != 88200 {
		t.Errorf("ByteRate = %d, want 88200", h.ByteRate)
	}
	// BlockAlign = 1 * 2 = 2
	if h.BlockAlign != 2 {
		t.Errorf("BlockAlign = %d, want 2", h.BlockAlign)
	}
	if h.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %d, want 16", h.BitsPerSample)
	}

	dataSize := uint32(4410 * 1 * 2)
	if h.Subchunk2Size != dataSize {
		t.Errorf("Subchunk2Size = %d, want %d", h.Subchunk2Size, dataSize)
	}
	if h.ChunkSize != dataSize+36 {
		t.Errorf("ChunkSize = %d, want %d", h.ChunkSize, dataSize+36)
	}
}

func TestWAVEncoder_EmptyInput(t *testing.T) {
	data := encodeWAV(t, []byte{}, 48000, 2)

	if len(data) != 44 {
		t.Fatalf("output size = %d, want 44", len(data))
	}

	h := parseWAVHeader(t, data)
	if h.Subchunk2Size != 0 {
		t.Errorf("Subchunk2Size = %d, want 0", h.Subchunk2Size)
	}
	if h.ChunkSize != 36 {
		t.Errorf("ChunkSize = %d, want 36", h.ChunkSize)
	}
}

func TestWAVEncoder_SingleSample(t *testing.T) {
	// Single sample, stereo = 1 * 2 channels * 2 bytes = 4 bytes PCM
	pcm := []byte{0x01, 0x00, 0x02, 0x00}
	data := encodeWAV(t, pcm, 48000, 2)

	if len(data) != 48 {
		t.Errorf("output size = %d, want 48 (44 header + 4 data)", len(data))
	}

	h := parseWAVHeader(t, data)
	if h.Subchunk2Size != 4 {
		t.Errorf("Subchunk2Size = %d, want 4", h.Subchunk2Size)
	}
}

func TestWAVEncoder_PCMDataPreserved(t *testing.T) {
	// Create recognizable PCM data pattern
	pcm := make([]byte, 100)
	for i := range pcm {
		pcm[i] = byte(i)
	}

	data := encodeWAV(t, pcm, 48000, 2)

	if len(data) < 44+len(pcm) {
		t.Fatalf("output too short: %d bytes", len(data))
	}

	audioData := data[44:]
	if !bytes.Equal(audioData, pcm) {
		t.Errorf("PCM data after header does not match input")
		// Show first mismatch
		for i := range pcm {
			if i >= len(audioData) {
				t.Errorf("  output truncated at byte %d", i)
				break
			}
			if audioData[i] != pcm[i] {
				t.Errorf("  first mismatch at byte %d: got 0x%02X, want 0x%02X", i, audioData[i], pcm[i])
				break
			}
		}
	}
}

func TestWAVEncoder_InvalidSampleRate(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate int
	}{
		{"zero", 0},
		{"negative", -1},
		{"large negative", -48000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewWAVEncoder()
			var buf bytes.Buffer
			err := enc.Encode(&buf, bytes.NewReader([]byte{0, 0}), tt.sampleRate, 2)
			if err == nil {
				t.Fatal("Encode() with invalid sample rate should return error")
			}
			if !errors.Is(err, ErrInvalidSampleRate) {
				t.Errorf("Encode() error = %v, want error wrapping ErrInvalidSampleRate", err)
			}
		})
	}
}

func TestWAVEncoder_InvalidChannels(t *testing.T) {
	tests := []struct {
		name     string
		channels int
	}{
		{"zero", 0},
		{"negative", -1},
		{"three", 3},
		{"large", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewWAVEncoder()
			var buf bytes.Buffer
			err := enc.Encode(&buf, bytes.NewReader([]byte{0, 0}), 48000, tt.channels)
			if err == nil {
				t.Fatal("Encode() with invalid channels should return error")
			}
			if !errors.Is(err, ErrInvalidChannels) {
				t.Errorf("Encode() error = %v, want error wrapping ErrInvalidChannels", err)
			}
		})
	}
}

func TestWAVEncoder_WriterError(t *testing.T) {
	enc := NewWAVEncoder()
	pcm := makePCM(100, 2)
	w := &failWriter{err: io.ErrClosedPipe}
	err := enc.Encode(w, bytes.NewReader(pcm), 48000, 2)
	if err == nil {
		t.Fatal("Encode() with failing writer should return error")
	}
	if !errors.Is(err, ErrWAVWrite) {
		t.Errorf("Encode() error = %v, want error wrapping ErrWAVWrite", err)
	}
}

// ---------------------------------------------------------------------------
// Table-driven header fields test
// ---------------------------------------------------------------------------

func TestWAVEncoder_HeaderFields_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate int
		channels   int
		pcmLen     int // PCM data length in bytes
	}{
		{"48kHz stereo 1s", 48000, 2, 48000 * 2 * 2},
		{"44100Hz mono 0.5s", 44100, 1, 44100 * 1 * 2 / 2},
		{"48kHz mono 20ms", 48000, 1, 960 * 1 * 2},
		{"22050Hz stereo 1s", 22050, 2, 22050 * 2 * 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pcm := make([]byte, tt.pcmLen)
			data := encodeWAV(t, pcm, tt.sampleRate, tt.channels)
			h := parseWAVHeader(t, data)

			if h.SampleRate != uint32(tt.sampleRate) {
				t.Errorf("SampleRate = %d, want %d", h.SampleRate, tt.sampleRate)
			}
			if h.NumChannels != uint16(tt.channels) {
				t.Errorf("NumChannels = %d, want %d", h.NumChannels, tt.channels)
			}

			expectedByteRate := uint32(tt.sampleRate * tt.channels * 2)
			if h.ByteRate != expectedByteRate {
				t.Errorf("ByteRate = %d, want %d", h.ByteRate, expectedByteRate)
			}

			expectedBlockAlign := uint16(tt.channels * 2)
			if h.BlockAlign != expectedBlockAlign {
				t.Errorf("BlockAlign = %d, want %d", h.BlockAlign, expectedBlockAlign)
			}

			if h.Subchunk2Size != uint32(tt.pcmLen) {
				t.Errorf("Subchunk2Size = %d, want %d", h.Subchunk2Size, tt.pcmLen)
			}

			if h.ChunkSize != uint32(tt.pcmLen)+36 {
				t.Errorf("ChunkSize = %d, want %d", h.ChunkSize, uint32(tt.pcmLen)+36)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkWAVEncoder_1s_Stereo48k(b *testing.B) {
	pcm := makePCM(48000, 2)
	enc := NewWAVEncoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		buf.Grow(44 + len(pcm))
		_ = enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)
	}
}

func BenchmarkWAVEncoder_3s_Stereo48k(b *testing.B) {
	pcm := makePCM(48000*3, 2)
	enc := NewWAVEncoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		buf.Grow(44 + len(pcm))
		_ = enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// failWriter is an io.Writer that always returns an error.
type failWriter struct {
	err error
}

func (w *failWriter) Write(p []byte) (int, error) {
	return 0, w.err
}
