package native

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// testScreamParams returns a minimal valid ScreamParams for generator tests.
func testScreamParams() audio.ScreamParams {
	return audio.ScreamParams{
		Duration:   3 * time.Second,
		SampleRate: 48000,
		Channels:   2,
		Seed:       12345,
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

func TestNativeGenerator_CorrectByteCount(t *testing.T) {
	gen := NewNativeGenerator()
	params := testScreamParams()

	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// 3s * 48000 samples/s * 2 channels * 2 bytes/sample = 576000 bytes
	expected := 3 * 48000 * 2 * 2
	if len(data) != expected {
		t.Errorf("byte count = %d, want %d", len(data), expected)
	}
}

func TestNativeGenerator_NonSilent(t *testing.T) {
	gen := NewNativeGenerator()
	params := testScreamParams()

	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Check that at least some bytes are non-zero
	hasNonZero := false
	for _, b := range data {
		if b != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("output is all zeros; expected non-silent audio")
	}
}

func TestNativeGenerator_Deterministic(t *testing.T) {
	gen := NewNativeGenerator()
	params := testScreamParams()

	reader1, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() #1 error = %v", err)
	}
	data1, err := io.ReadAll(reader1)
	if err != nil {
		t.Fatalf("ReadAll() #1 error = %v", err)
	}

	reader2, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() #2 error = %v", err)
	}
	data2, err := io.ReadAll(reader2)
	if err != nil {
		t.Fatalf("ReadAll() #2 error = %v", err)
	}

	if !bytes.Equal(data1, data2) {
		t.Error("same params produced different output; expected deterministic generation")
	}
}

func TestNativeGenerator_DifferentSeeds(t *testing.T) {
	gen := NewNativeGenerator()

	params1 := testScreamParams()
	params1.Seed = 111

	params2 := testScreamParams()
	params2.Seed = 222

	reader1, err := gen.Generate(params1)
	if err != nil {
		t.Fatalf("Generate() #1 error = %v", err)
	}
	data1, err := io.ReadAll(reader1)
	if err != nil {
		t.Fatalf("ReadAll() #1 error = %v", err)
	}

	reader2, err := gen.Generate(params2)
	if err != nil {
		t.Fatalf("Generate() #2 error = %v", err)
	}
	data2, err := io.ReadAll(reader2)
	if err != nil {
		t.Fatalf("ReadAll() #2 error = %v", err)
	}

	if bytes.Equal(data1, data2) {
		t.Error("different seeds produced identical output; expected different audio")
	}
}

func TestNativeGenerator_AllPresets(t *testing.T) {
	gen := NewNativeGenerator()

	for _, name := range audio.AllPresets() {
		t.Run(string(name), func(t *testing.T) {
			params, ok := audio.GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false", name)
			}

			reader, err := gen.Generate(params)
			if err != nil {
				t.Fatalf("Generate(%q) error = %v", name, err)
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}

			if len(data) == 0 {
				t.Error("Generate() produced empty output")
			}

			// Verify byte count matches expected for preset's parameters
			expectedBytes := int(params.Duration.Seconds()) * params.SampleRate * params.Channels * 2
			// Use approximate check since Duration may not be exact seconds
			totalSamples := int(params.Duration.Seconds() * float64(params.SampleRate))
			expectedBytesExact := totalSamples * params.Channels * 2
			if len(data) != expectedBytesExact {
				t.Errorf("byte count = %d, want %d (approx %d)", len(data), expectedBytesExact, expectedBytes)
			}
		})
	}
}

func TestNativeGenerator_InvalidParams(t *testing.T) {
	gen := NewNativeGenerator()

	params := testScreamParams()
	params.Duration = 0

	_, err := gen.Generate(params)
	if err == nil {
		t.Error("Generate() with Duration=0 should return error, got nil")
	}
}

func TestNativeGenerator_MonoOutput(t *testing.T) {
	gen := NewNativeGenerator()

	params := testScreamParams()
	params.Channels = 1

	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Mono: 3s * 48000 * 1ch * 2bytes = 288000 bytes (half of stereo)
	expected := 3 * 48000 * 1 * 2
	if len(data) != expected {
		t.Errorf("mono byte count = %d, want %d", len(data), expected)
	}
}

func TestNativeGenerator_S16LERange(t *testing.T) {
	gen := NewNativeGenerator()
	params := testScreamParams()

	reader, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Parse as little-endian int16 samples and verify range
	if len(data)%2 != 0 {
		t.Fatalf("data length %d is not even; cannot parse as int16", len(data))
	}

	buf := bytes.NewReader(data)
	numSamples := len(data) / 2
	for i := 0; i < numSamples; i++ {
		var sample int16
		if err := binary.Read(buf, binary.LittleEndian, &sample); err != nil {
			t.Fatalf("failed to read sample %d: %v", i, err)
		}
		// int16 range is [-32768, 32767] by definition, so this verifies
		// the data is valid s16le encoding. We also check no extreme clipping
		// (not ALL samples at min/max).
	}

	// Additional check: not all samples should be at the extremes
	buf = bytes.NewReader(data)
	extremeCount := 0
	for i := 0; i < numSamples; i++ {
		var sample int16
		if err := binary.Read(buf, binary.LittleEndian, &sample); err != nil {
			t.Fatalf("failed to read sample %d: %v", i, err)
		}
		if sample == -32768 || sample == 32767 {
			extremeCount++
		}
	}
	if extremeCount == numSamples {
		t.Error("all samples are at int16 extremes; indicates encoding error")
	}
}

// TestNativeGenerator_ImplementsInterface verifies NativeGenerator satisfies AudioGenerator.
func TestNativeGenerator_ImplementsInterface(t *testing.T) {
	var _ audio.AudioGenerator = NewNativeGenerator()
}

// --- Benchmarks ---

func BenchmarkNativeGenerator_Classic(b *testing.B) {
	gen := NewNativeGenerator()
	params, _ := audio.GetPreset(audio.PresetClassic)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := gen.Generate(params)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			b.Fatalf("io.Copy failed: %v", err)
		}
	}
}
