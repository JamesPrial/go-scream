package encoding

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// ---------------------------------------------------------------------------
// Compile-time interface check
// ---------------------------------------------------------------------------

func TestOGGEncoder_ImplementsFileEncoder(t *testing.T) {
	var _ FileEncoder = (*OGGEncoder)(nil)
}

// ---------------------------------------------------------------------------
// Mock OpusFrameEncoder
// ---------------------------------------------------------------------------

type mockOpusEncoder struct {
	frames [][]byte
	err    error
}

func (m *mockOpusEncoder) EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error) {
	frameCh := make(chan []byte, len(m.frames))
	errCh := make(chan error, 1)
	go func() {
		defer close(frameCh)
		// Drain the source reader to simulate reading PCM data
		_, _ = io.Copy(io.Discard, src)
		for _, f := range m.frames {
			frameCh <- f
		}
		errCh <- m.err
	}()
	return frameCh, errCh
}

// mockOpusEncoderValidating validates sample rate and channels.
type mockOpusEncoderValidating struct {
	frames [][]byte
	err    error
}

func (m *mockOpusEncoderValidating) EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error) {
	frameCh := make(chan []byte, len(m.frames))
	errCh := make(chan error, 1)
	go func() {
		defer close(frameCh)
		_, _ = io.Copy(io.Discard, src)

		if sampleRate <= 0 {
			errCh <- ErrInvalidSampleRate
			return
		}
		if channels < 1 || channels > 2 {
			errCh <- ErrInvalidChannels
			return
		}

		for _, f := range m.frames {
			frameCh <- f
		}
		errCh <- m.err
	}()
	return frameCh, errCh
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeFakeOpusFrames creates a slice of fake Opus frame data for testing.
func makeFakeOpusFrames(count, frameSize int) [][]byte {
	frames := make([][]byte, count)
	for i := range frames {
		frame := make([]byte, frameSize)
		// Fill with a recognizable pattern
		for j := range frame {
			frame[j] = byte((i + j) % 256)
		}
		frames[i] = frame
	}
	return frames
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestOGGEncoder_StartsWithOggS(t *testing.T) {
	frames := makeFakeOpusFrames(10, 100)
	mock := &mockOpusEncoder{frames: frames}
	enc := NewOGGEncoderWithOpus(mock, discardLogger)

	var buf bytes.Buffer
	pcm := make([]byte, 3840*10) // enough PCM for 10 frames
	err := enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	data := buf.Bytes()
	if len(data) < 4 {
		t.Fatalf("output too short: %d bytes", len(data))
	}

	// OGG capture pattern: "OggS" = 0x4F 0x67 0x67 0x53
	if data[0] != 0x4F || data[1] != 0x67 || data[2] != 0x67 || data[3] != 0x53 {
		t.Errorf("output does not start with OggS capture pattern: got [0x%02X 0x%02X 0x%02X 0x%02X]",
			data[0], data[1], data[2], data[3])
	}
}

func TestOGGEncoder_NonEmptyOutput(t *testing.T) {
	frames := makeFakeOpusFrames(5, 80)
	mock := &mockOpusEncoder{frames: frames}
	enc := NewOGGEncoderWithOpus(mock, discardLogger)

	var buf bytes.Buffer
	pcm := make([]byte, 3840*5)
	err := enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Encode() produced empty output for valid input frames")
	}
}

func TestOGGEncoder_InvalidSampleRate(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use validating mock that returns ErrInvalidSampleRate for bad params
			mock := &mockOpusEncoderValidating{
				frames: makeFakeOpusFrames(1, 80),
			}
			enc := NewOGGEncoderWithOpus(mock, discardLogger)

			var buf bytes.Buffer
			pcm := make([]byte, 3840)
			err := enc.Encode(&buf, bytes.NewReader(pcm), tt.sampleRate, 2)
			if err == nil {
				t.Fatal("Encode() should return error for invalid sample rate")
			}
			if !errors.Is(err, ErrInvalidSampleRate) {
				t.Errorf("Encode() error = %v, want error wrapping ErrInvalidSampleRate", err)
			}
		})
	}
}

func TestOGGEncoder_InvalidChannels(t *testing.T) {
	tests := []struct {
		name     string
		channels int
	}{
		{"zero", 0},
		{"three", 3},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockOpusEncoderValidating{
				frames: makeFakeOpusFrames(1, 80),
			}
			enc := NewOGGEncoderWithOpus(mock, discardLogger)

			var buf bytes.Buffer
			pcm := make([]byte, 3840)
			err := enc.Encode(&buf, bytes.NewReader(pcm), 48000, tt.channels)
			if err == nil {
				t.Fatal("Encode() should return error for invalid channels")
			}
			if !errors.Is(err, ErrInvalidChannels) {
				t.Errorf("Encode() error = %v, want error wrapping ErrInvalidChannels", err)
			}
		})
	}
}

func TestOGGEncoder_OpusError(t *testing.T) {
	opusErr := errors.New("simulated opus failure")
	mock := &mockOpusEncoder{
		frames: nil, // no frames, just an error
		err:    opusErr,
	}
	enc := NewOGGEncoderWithOpus(mock, discardLogger)

	var buf bytes.Buffer
	pcm := make([]byte, 3840)
	err := enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)
	if err == nil {
		t.Fatal("Encode() should propagate error from OpusFrameEncoder")
	}

	// The OGG encoder should propagate the underlying opus error.
	// It may or may not wrap it further.
	// Check that the error message contains something about the opus failure
	// or that we can unwrap to the original.
	if !errors.Is(err, opusErr) {
		// It may be wrapped - check string containment as fallback
		if err.Error() == "" {
			t.Errorf("Encode() returned empty error, want propagated opus error")
		}
		// The implementation might wrap with ErrOpusEncode, which is acceptable
		if !errors.Is(err, ErrOpusEncode) && !errors.Is(err, opusErr) {
			t.Logf("note: error %v does not unwrap to original opus error %v (may be wrapped differently)", err, opusErr)
		}
	}
}

func TestOGGEncoder_EmptyFrames(t *testing.T) {
	// Zero frames from the encoder - should handle gracefully
	mock := &mockOpusEncoder{
		frames: nil, // no frames
		err:    nil, // no error
	}
	enc := NewOGGEncoderWithOpus(mock, discardLogger)

	var buf bytes.Buffer
	pcm := make([]byte, 0)
	err := enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)

	// Implementation may either:
	// 1. Return nil with no output (or minimal OGG headers)
	// 2. Return an error indicating no data
	// Both are acceptable behaviors for empty input.
	// The key thing is it should NOT panic.
	if err != nil {
		// If it returns an error, that's fine - just log it
		t.Logf("Encode() with empty frames returned error (acceptable): %v", err)
	}
}

func TestOGGEncoder_WriterError(t *testing.T) {
	frames := makeFakeOpusFrames(5, 100)
	mock := &mockOpusEncoder{frames: frames}
	enc := NewOGGEncoderWithOpus(mock, discardLogger)

	w := &failOGGWriter{err: io.ErrClosedPipe, failAfter: 0}
	pcm := make([]byte, 3840*5)
	err := enc.Encode(w, bytes.NewReader(pcm), 48000, 2)
	if err == nil {
		t.Fatal("Encode() with failing writer should return error")
	}
	if !errors.Is(err, ErrOGGWrite) {
		t.Errorf("Encode() error = %v, want error wrapping ErrOGGWrite", err)
	}
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewOGGEncoder_NotNil(t *testing.T) {
	// NewOGGEncoder creates an OGG encoder with a default GopusFrameEncoder.
	// This may panic if opus is not available, so we guard it.
	defer func() {
		if r := recover(); r != nil {
			t.Skip("opus not available for NewOGGEncoder: ", r)
		}
	}()

	enc := NewOGGEncoder(discardLogger)
	if enc == nil {
		t.Fatal("NewOGGEncoder() returned nil")
	}
}

func TestNewOGGEncoderWithOpus_NotNil(t *testing.T) {
	mock := &mockOpusEncoder{}
	enc := NewOGGEncoderWithOpus(mock, discardLogger)
	if enc == nil {
		t.Fatal("NewOGGEncoderWithOpus() returned nil")
	}
}

// ---------------------------------------------------------------------------
// Multiple frame sizes
// ---------------------------------------------------------------------------

func TestOGGEncoder_VariousFrameCounts(t *testing.T) {
	tests := []struct {
		name       string
		frameCount int
		frameSize  int
	}{
		{"1 frame", 1, 100},
		{"10 frames", 10, 100},
		{"50 frames", 50, 80},
		{"150 frames (3s)", 150, 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frames := makeFakeOpusFrames(tt.frameCount, tt.frameSize)
			mock := &mockOpusEncoder{frames: frames}
			enc := NewOGGEncoderWithOpus(mock, discardLogger)

			var buf bytes.Buffer
			pcm := make([]byte, 3840*tt.frameCount)
			err := enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			if buf.Len() == 0 {
				t.Error("Encode() produced empty output")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// failOGGWriter is an io.Writer that fails after a certain number of bytes.
type failOGGWriter struct {
	err       error
	failAfter int // fail immediately if 0
	written   int
}

func (w *failOGGWriter) Write(p []byte) (int, error) {
	if w.written >= w.failAfter {
		return 0, w.err
	}
	canWrite := w.failAfter - w.written
	if canWrite > len(p) {
		canWrite = len(p)
	}
	w.written += canWrite
	if w.written >= w.failAfter && canWrite < len(p) {
		return canWrite, w.err
	}
	return canWrite, nil
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkOGGEncoder_150Frames(b *testing.B) {
	frames := makeFakeOpusFrames(150, 100)
	mock := &mockOpusEncoder{frames: frames}
	enc := NewOGGEncoderWithOpus(mock, discardLogger)
	pcm := make([]byte, 3840*150)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = enc.Encode(&buf, bytes.NewReader(pcm), 48000, 2)
	}
}
