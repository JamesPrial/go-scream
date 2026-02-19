package ffmpeg

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"testing"
	"time"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// skipIfNoFFmpeg skips the test if ffmpeg is not available on PATH.
func skipIfNoFFmpeg(t *testing.T) {
	t.Helper()
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available")
	}
}

// testParams returns a valid ScreamParams suitable for fast integration tests.
// Uses classic-style parameters with 1s duration for speed.
func testParams() audio.ScreamParams {
	return audio.ScreamParams{
		Duration:   1 * time.Second,
		SampleRate: 48000,
		Channels:   2,
		Seed:       42,
		Layers: [5]audio.LayerParams{
			{Type: audio.LayerPrimaryScream, BaseFreq: 500, FreqRange: 1500, JumpRate: 10, Amplitude: 0.4, Rise: 1.2, Seed: 4242},
			{Type: audio.LayerHarmonicSweep, BaseFreq: 350, SweepRate: 500, FreqRange: 800, JumpRate: 6, Amplitude: 0.25, Seed: 3000},
			{Type: audio.LayerHighShriek, BaseFreq: 1200, FreqRange: 1600, JumpRate: 20, Amplitude: 0.25, Rise: 2.5, Seed: 7000},
			{Type: audio.LayerNoiseBurst, Amplitude: 0.18, Seed: 4000},
			{Type: audio.LayerBackgroundNoise, Amplitude: 0.1},
		},
		Noise: audio.NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000},
		Filter: audio.FilterParams{
			HighpassCutoff: 120, LowpassCutoff: 8000,
			CrusherBits: 8, CrusherMix: 0.5,
			CompRatio: 8, CompThreshold: -20, CompAttack: 5, CompRelease: 50,
			VolumeBoostDB: 9, LimiterLevel: 0.95,
		},
	}
}

// --- Compile-time interface check ---

var _ audio.AudioGenerator = (*FFmpegGenerator)(nil)

// --- Constructor tests ---

func TestNewFFmpegGenerator_Success(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v, want nil", err)
	}
	if gen == nil {
		t.Fatal("NewFFmpegGenerator() returned nil generator with nil error")
	}
}

func TestNewFFmpegGeneratorWithPath_NotNil(t *testing.T) {
	gen := NewFFmpegGeneratorWithPath("/usr/bin/ffmpeg")
	if gen == nil {
		t.Fatal("NewFFmpegGeneratorWithPath() returned nil")
	}
}

func TestNewFFmpegGenerator_NoFFmpegOnPath(t *testing.T) {
	// This test verifies the error case when ffmpeg is not found.
	// We cannot easily manipulate PATH in the test, so we rely on
	// the explicit path constructor for that case. This test documents
	// the expected behavior: when exec.LookPath fails,
	// NewFFmpegGenerator returns ErrFFmpegNotFound.

	// We just verify the sentinel error exists and is usable.
	if ErrFFmpegNotFound == nil {
		t.Fatal("ErrFFmpegNotFound should not be nil")
	}
	if ErrFFmpegFailed == nil {
		t.Fatal("ErrFFmpegFailed should not be nil")
	}
}

// --- Generate output correctness tests ---

func TestFFmpegGenerator_CorrectByteCount(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// 1 second * 48000 Hz * 2 channels * 2 bytes (s16le) = 192000 bytes
	expectedMin := 192000
	if len(data) < expectedMin {
		t.Errorf("Generate() produced %d bytes, want >= %d (1s stereo 48kHz s16le)", len(data), expectedMin)
	}
}

func TestFFmpegGenerator_NonSilent(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	hasNonZero := false
	for _, b := range data {
		if b != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("Generate() produced all-zero (silent) output, expected non-silent audio data")
	}
}

func TestFFmpegGenerator_AllPresets(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	// Use different seeds to create varied params
	seeds := []int64{1, 42, 100, 9999, 12345}
	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			params := audio.Randomize(seed)
			// Override duration to 1s for speed
			params.Duration = 1 * time.Second

			reader, err := gen.Generate(params)
			if err != nil {
				t.Fatalf("Generate() with seed %d error = %v", seed, err)
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}

			if len(data) == 0 {
				t.Errorf("Generate() with seed %d produced empty output", seed)
			}
		})
	}
}

func TestFFmpegGenerator_AllNamedPresets(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	for _, name := range audio.AllPresets() {
		t.Run(string(name), func(t *testing.T) {
			params, ok := audio.GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false", name)
			}
			// Override duration to 1s for speed
			params.Duration = 1 * time.Second

			reader, err := gen.Generate(params)
			if err != nil {
				t.Fatalf("Generate() for preset %q error = %v", name, err)
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}

			if len(data) == 0 {
				t.Errorf("Generate() for preset %q produced empty output", name)
			}
		})
	}
}

// --- Generate error condition tests ---

func TestFFmpegGenerator_InvalidDuration(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Duration = 0

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with Duration=0 should return error")
	}
	if !errors.Is(err, audio.ErrInvalidDuration) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidDuration)", err)
	}
}

func TestFFmpegGenerator_NegativeDuration(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Duration = -1 * time.Second

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with negative Duration should return error")
	}
	if !errors.Is(err, audio.ErrInvalidDuration) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidDuration)", err)
	}
}

func TestFFmpegGenerator_InvalidSampleRate(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.SampleRate = 0

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with SampleRate=0 should return error")
	}
	if !errors.Is(err, audio.ErrInvalidSampleRate) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidSampleRate)", err)
	}
}

func TestFFmpegGenerator_NegativeSampleRate(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.SampleRate = -44100

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with negative SampleRate should return error")
	}
	if !errors.Is(err, audio.ErrInvalidSampleRate) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidSampleRate)", err)
	}
}

func TestFFmpegGenerator_InvalidChannels(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Channels = 3

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with Channels=3 should return error")
	}
	if !errors.Is(err, audio.ErrInvalidChannels) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidChannels)", err)
	}
}

func TestFFmpegGenerator_ZeroChannels(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Channels = 0

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with Channels=0 should return error")
	}
	if !errors.Is(err, audio.ErrInvalidChannels) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidChannels)", err)
	}
}

func TestFFmpegGenerator_BadBinaryPath(t *testing.T) {
	gen := NewFFmpegGeneratorWithPath("/nonexistent/ffmpeg")

	params := testParams()
	_, err := gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with bad binary path should return error")
	}
	if !errors.Is(err, ErrFFmpegFailed) {
		t.Errorf("Generate() error = %v, want errors.Is(err, ErrFFmpegFailed)", err)
	}
}

func TestFFmpegGenerator_InvalidAmplitude(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Layers[0].Amplitude = 1.5

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with invalid amplitude should return error")
	}
	if !errors.Is(err, audio.ErrInvalidAmplitude) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidAmplitude)", err)
	}
}

func TestFFmpegGenerator_InvalidCrusherBits(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Filter.CrusherBits = 0

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with CrusherBits=0 should return error")
	}
	if !errors.Is(err, audio.ErrInvalidCrusherBits) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidCrusherBits)", err)
	}
}

func TestFFmpegGenerator_InvalidLimiterLevel(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Filter.LimiterLevel = 0

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("Generate() with LimiterLevel=0 should return error")
	}
	if !errors.Is(err, audio.ErrInvalidLimiterLevel) {
		t.Errorf("Generate() error = %v, want errors.Is(err, audio.ErrInvalidLimiterLevel)", err)
	}
}

// --- Output format tests ---

func TestFFmpegGenerator_EvenByteCount(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(data)%2 != 0 {
		t.Errorf("Generate() produced %d bytes, expected even count for s16le format", len(data))
	}
}

func TestFFmpegGenerator_StereoAligned(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Channels = 2
	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// For stereo s16le, each frame is 4 bytes (2 bytes * 2 channels)
	if len(data)%4 != 0 {
		t.Errorf("Generate() stereo output %d bytes is not aligned to 4-byte frames", len(data))
	}
}

func TestFFmpegGenerator_MonoOutput(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()
	params.Channels = 1
	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// 1 second * 48000 Hz * 1 channel * 2 bytes = 96000 bytes minimum
	expectedMin := 96000
	if len(data) < expectedMin {
		t.Errorf("Generate() mono produced %d bytes, want >= %d", len(data), expectedMin)
	}
}

// --- Determinism tests ---

func TestFFmpegGenerator_DeterministicOutput(t *testing.T) {
	skipIfNoFFmpeg(t)

	gen, err := NewFFmpegGenerator()
	if err != nil {
		t.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()

	reader1, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() first call error = %v", err)
	}
	data1, err := io.ReadAll(reader1)
	if err != nil {
		t.Fatalf("ReadAll() first call error = %v", err)
	}

	reader2, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() second call error = %v", err)
	}
	data2, err := io.ReadAll(reader2)
	if err != nil {
		t.Fatalf("ReadAll() second call error = %v", err)
	}

	if len(data1) != len(data2) {
		t.Errorf("Determinism check: first call %d bytes, second call %d bytes", len(data1), len(data2))
	}

	// Compare content (FFmpeg with same aevalsrc seed should produce identical output)
	mismatch := 0
	minLen := len(data1)
	if len(data2) < minLen {
		minLen = len(data2)
	}
	for i := 0; i < minLen; i++ {
		if data1[i] != data2[i] {
			mismatch++
		}
	}
	if mismatch > 0 {
		t.Errorf("Determinism check: %d byte mismatches out of %d bytes", mismatch, minLen)
	}
}

// --- Benchmarks ---

func BenchmarkFFmpegGenerator_1s(b *testing.B) {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		b.Skip("ffmpeg not available")
	}

	gen, err := NewFFmpegGenerator()
	if err != nil {
		b.Fatalf("NewFFmpegGenerator() error = %v", err)
	}

	params := testParams()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := gen.Generate(params)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
		_, err = io.ReadAll(reader)
		if err != nil {
			b.Fatalf("ReadAll() error = %v", err)
		}
	}
}
