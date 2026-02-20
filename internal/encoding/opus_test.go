package encoding

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"testing"
)

// ---------------------------------------------------------------------------
// Compile-time interface check
// ---------------------------------------------------------------------------

func TestGopusFrameEncoder_ImplementsInterface(t *testing.T) {
	var _ OpusFrameEncoder = (*GopusFrameEncoder)(nil)
}

// ---------------------------------------------------------------------------
// CGO skip helper
// ---------------------------------------------------------------------------

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func skipIfNoOpus(t *testing.T) {
	t.Helper()
	// Try creating an encoder; if gopus/libopus isn't available, skip.
	defer func() {
		if r := recover(); r != nil {
			t.Skip("opus not available: ", r)
		}
	}()
	enc := NewGopusFrameEncoder(discardLogger)
	if enc == nil {
		t.Skip("opus not available: NewGopusFrameEncoder returned nil")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// drainFrames reads all frames from the channel and returns them, plus any error.
func drainFrames(t *testing.T, frameCh <-chan []byte, errCh <-chan error) ([][]byte, error) {
	t.Helper()
	var frames [][]byte
	for f := range frameCh {
		frames = append(frames, f)
	}
	// Read the error (should always be sent before errCh is closed or has capacity 1).
	// The goroutine may still be finishing, so we do a blocking read here rather
	// than a select/default, to avoid a race where the error hasn't been sent yet.
	encErr := <-errCh
	return frames, encErr
}

// makeSilentPCM creates silent (zero-valued) PCM data.
// numBytes is the total byte count. Must be even for s16le.
func makeSilentPCM(numBytes int) []byte {
	return make([]byte, numBytes)
}

// pcmBytesForDuration calculates the number of PCM bytes for a given duration
// in seconds at the specified sample rate and channel count (s16le format).
func pcmBytesForDuration(seconds float64, sampleRate, channels int) int {
	totalSamples := int(seconds * float64(sampleRate))
	return totalSamples * channels * 2
}

// ---------------------------------------------------------------------------
// Frame count tests
// ---------------------------------------------------------------------------

func TestGopusFrameEncoder_FrameCount_3s(t *testing.T) {
	skipIfNoOpus(t)

	// 3 seconds of stereo 48kHz = 48000 * 3 * 2 channels * 2 bytes = 576000 bytes
	// Frames per second = 48000 / 960 = 50
	// 3 seconds = 150 frames
	pcmSize := pcmBytesForDuration(3.0, 48000, 2)
	if pcmSize != 576000 {
		t.Fatalf("expected 576000 bytes, got %d", pcmSize)
	}

	pcm := makeSilentPCM(pcmSize)
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)

	frames, err := drainFrames(t, frameCh, errCh)
	if err != nil {
		t.Fatalf("EncodeFrames() error = %v", err)
	}

	if len(frames) != 150 {
		t.Errorf("frame count = %d, want 150", len(frames))
	}
}

func TestGopusFrameEncoder_FrameCount_1Frame(t *testing.T) {
	skipIfNoOpus(t)

	// Exactly 1 frame of stereo 48kHz:
	// 960 samples * 2 channels * 2 bytes = 3840 bytes
	pcm := makeSilentPCM(3840)
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)

	frames, err := drainFrames(t, frameCh, errCh)
	if err != nil {
		t.Fatalf("EncodeFrames() error = %v", err)
	}

	if len(frames) != 1 {
		t.Errorf("frame count = %d, want 1", len(frames))
	}
}

func TestGopusFrameEncoder_PartialFrame(t *testing.T) {
	skipIfNoOpus(t)

	// 1.5 frames of stereo 48kHz:
	// 1.5 * 960 = 1440 samples * 2 channels * 2 bytes = 5760 bytes
	// Should yield 2 frames (the partial frame is zero-padded)
	pcm := makeSilentPCM(5760)
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)

	frames, err := drainFrames(t, frameCh, errCh)
	if err != nil {
		t.Fatalf("EncodeFrames() error = %v", err)
	}

	if len(frames) != 2 {
		t.Errorf("frame count = %d, want 2 (partial frame should be zero-padded)", len(frames))
	}
}

func TestGopusFrameEncoder_SingleSample(t *testing.T) {
	skipIfNoOpus(t)

	// 1 stereo sample = 4 bytes. Should produce 1 frame (heavily zero-padded).
	pcm := makeSilentPCM(4)
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)

	frames, err := drainFrames(t, frameCh, errCh)
	if err != nil {
		t.Fatalf("EncodeFrames() error = %v", err)
	}

	if len(frames) != 1 {
		t.Errorf("frame count = %d, want 1", len(frames))
	}
}

func TestGopusFrameEncoder_EmptyInput(t *testing.T) {
	skipIfNoOpus(t)

	pcm := makeSilentPCM(0)
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)

	frames, err := drainFrames(t, frameCh, errCh)
	if err != nil {
		t.Fatalf("EncodeFrames() error = %v, want nil", err)
	}

	if len(frames) != 0 {
		t.Errorf("frame count = %d, want 0 for empty input", len(frames))
	}
}

func TestGopusFrameEncoder_FrameSizeBounds(t *testing.T) {
	skipIfNoOpus(t)

	// 0.5 seconds of stereo 48kHz = 25 frames
	pcm := makeSilentPCM(pcmBytesForDuration(0.5, 48000, 2))
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)

	frames, err := drainFrames(t, frameCh, errCh)
	if err != nil {
		t.Fatalf("EncodeFrames() error = %v", err)
	}

	for i, f := range frames {
		if len(f) == 0 {
			t.Errorf("frame[%d] is empty", i)
		}
		if len(f) > MaxOpusFrameBytes {
			t.Errorf("frame[%d] size = %d, exceeds MaxOpusFrameBytes (%d)", i, len(f), MaxOpusFrameBytes)
		}
	}
}

func TestGopusFrameEncoder_MonoEncoding(t *testing.T) {
	skipIfNoOpus(t)

	// 1 frame of mono 48kHz:
	// 960 samples * 1 channel * 2 bytes = 1920 bytes
	pcm := makeSilentPCM(1920)
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 1)

	frames, err := drainFrames(t, frameCh, errCh)
	if err != nil {
		t.Fatalf("EncodeFrames() error = %v", err)
	}

	if len(frames) != 1 {
		t.Errorf("frame count = %d, want 1", len(frames))
	}

	if len(frames) > 0 && len(frames[0]) == 0 {
		t.Error("mono frame is empty")
	}
}

// ---------------------------------------------------------------------------
// Validation error tests
// ---------------------------------------------------------------------------

func TestGopusFrameEncoder_InvalidSampleRate(t *testing.T) {
	skipIfNoOpus(t)

	tests := []struct {
		name       string
		sampleRate int
	}{
		{"zero", 0},
		{"22050 (unsupported by opus)", 22050},
		{"negative", -48000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewGopusFrameEncoder(discardLogger)
			pcm := makeSilentPCM(3840)
			frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), tt.sampleRate, 2)

			// Drain all frames and collect the error.
			_, encErr := drainFrames(t, frameCh, errCh)

			if encErr == nil {
				t.Fatal("EncodeFrames() with invalid sample rate should send error")
			}
			if !errors.Is(encErr, ErrInvalidSampleRate) {
				t.Errorf("error = %v, want error wrapping ErrInvalidSampleRate", encErr)
			}
		})
	}
}

func TestGopusFrameEncoder_InvalidChannels(t *testing.T) {
	skipIfNoOpus(t)

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
			enc := NewGopusFrameEncoder(discardLogger)
			pcm := makeSilentPCM(3840)
			frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, tt.channels)

			// Drain all frames and collect the error.
			_, encErr := drainFrames(t, frameCh, errCh)

			if encErr == nil {
				t.Fatal("EncodeFrames() with invalid channels should send error")
			}
			if !errors.Is(encErr, ErrInvalidChannels) {
				t.Errorf("error = %v, want error wrapping ErrInvalidChannels", encErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Channel lifecycle tests
// ---------------------------------------------------------------------------

func TestGopusFrameEncoder_ChannelsClosed(t *testing.T) {
	skipIfNoOpus(t)

	pcm := makeSilentPCM(3840) // 1 frame
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)

	// Drain all frames via range (blocks until frameCh is closed)
	for range frameCh {
	}

	// frameCh should now be closed. Verify by receiving (should return zero value, false).
	_, open := <-frameCh
	if open {
		t.Error("frame channel should be closed after encoding completes")
	}

	// errCh should have exactly one value (possibly nil).
	<-errCh
}

func TestGopusFrameEncoder_ChannelsClosed_OnError(t *testing.T) {
	skipIfNoOpus(t)

	// Use invalid params to trigger an error path
	pcm := makeSilentPCM(3840)
	enc := NewGopusFrameEncoder(discardLogger)
	frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 0, 2) // invalid sample rate

	// Frame channel should eventually close even on error
	for range frameCh {
	}

	// Verify frame channel is closed
	_, open := <-frameCh
	if open {
		t.Error("frame channel should be closed even when encoding fails")
	}

	// Error channel should have the error
	err := <-errCh
	if err == nil {
		t.Error("expected error for invalid sample rate, got nil")
	}
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewGopusFrameEncoder_NotNil(t *testing.T) {
	skipIfNoOpus(t)
	enc := NewGopusFrameEncoder(discardLogger)
	if enc == nil {
		t.Fatal("NewGopusFrameEncoder() returned nil")
	}
}

func TestNewGopusFrameEncoderWithBitrate_NotNil(t *testing.T) {
	skipIfNoOpus(t)
	enc := NewGopusFrameEncoderWithBitrate(128000, discardLogger)
	if enc == nil {
		t.Fatal("NewGopusFrameEncoderWithBitrate(128000) returned nil")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkGopusFrameEncoder_1s_Stereo48k(b *testing.B) {
	// Skip if opus is unavailable
	func() {
		defer func() {
			if r := recover(); r != nil {
				b.Skip("opus not available")
			}
		}()
		enc := NewGopusFrameEncoder(discardLogger)
		if enc == nil {
			b.Skip("opus not available")
		}
	}()

	pcm := makeSilentPCM(pcmBytesForDuration(1.0, 48000, 2))
	enc := NewGopusFrameEncoder(discardLogger)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		frameCh, errCh := enc.EncodeFrames(bytes.NewReader(pcm), 48000, 2)
		for range frameCh {
		}
		<-errCh
	}
}
