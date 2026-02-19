package audio

import (
	"errors"
	"testing"
	"time"
)

func TestRandomize_ProducesValidParams(t *testing.T) {
	p := Randomize(12345)
	if err := p.Validate(); err != nil {
		t.Fatalf("Randomize(12345) produced invalid params: %v", err)
	}

	// Duration should be between 2.5s and 4.0s
	if p.Duration < time.Duration(2.5*float64(time.Second)) || p.Duration > 4*time.Second {
		t.Errorf("Duration %v outside expected range [2.5s, 4s]", p.Duration)
	}
	if p.SampleRate != 48000 {
		t.Errorf("SampleRate = %d, want 48000", p.SampleRate)
	}
	if p.Channels != 2 {
		t.Errorf("Channels = %d, want 2", p.Channels)
	}
	if p.Seed != 12345 {
		t.Errorf("Seed = %d, want 12345", p.Seed)
	}

	// Check all layer amplitudes are in [0, 1]
	for i, l := range p.Layers {
		if l.Amplitude < 0 || l.Amplitude > 1 {
			t.Errorf("Layer[%d].Amplitude = %f, want in [0, 1]", i, l.Amplitude)
		}
	}

	// Check layer types are assigned correctly
	expectedTypes := [5]LayerType{
		LayerPrimaryScream,
		LayerHarmonicSweep,
		LayerHighShriek,
		LayerNoiseBurst,
		LayerBackgroundNoise,
	}
	for i, l := range p.Layers {
		if l.Type != expectedTypes[i] {
			t.Errorf("Layer[%d].Type = %d, want %d", i, l.Type, expectedTypes[i])
		}
	}

	// Check filter params ranges
	if p.Filter.HighpassCutoff < 80 || p.Filter.HighpassCutoff > 200 {
		t.Errorf("HighpassCutoff = %f, want in [80, 200]", p.Filter.HighpassCutoff)
	}
	if p.Filter.LowpassCutoff < 6000 || p.Filter.LowpassCutoff > 12000 {
		t.Errorf("LowpassCutoff = %f, want in [6000, 12000]", p.Filter.LowpassCutoff)
	}
	if p.Filter.CrusherBits < 6 || p.Filter.CrusherBits > 12 {
		t.Errorf("CrusherBits = %d, want in [6, 12]", p.Filter.CrusherBits)
	}
	if p.Filter.CrusherMix < 0.3 || p.Filter.CrusherMix > 0.7 {
		t.Errorf("CrusherMix = %f, want in [0.3, 0.7]", p.Filter.CrusherMix)
	}
	if p.Filter.LimiterLevel != 0.95 {
		t.Errorf("LimiterLevel = %f, want 0.95", p.Filter.LimiterLevel)
	}
	if p.Filter.VolumeBoostDB < 6 || p.Filter.VolumeBoostDB > 12 {
		t.Errorf("VolumeBoostDB = %f, want in [6, 12]", p.Filter.VolumeBoostDB)
	}

	// Check noise params ranges
	if p.Noise.BurstRate < 3 || p.Noise.BurstRate > 12 {
		t.Errorf("Noise.BurstRate = %f, want in [3, 12]", p.Noise.BurstRate)
	}
	if p.Noise.Threshold < 0.5 || p.Noise.Threshold > 0.85 {
		t.Errorf("Noise.Threshold = %f, want in [0.5, 0.85]", p.Noise.Threshold)
	}
}

func TestRandomize_Deterministic(t *testing.T) {
	p1 := Randomize(42)
	p2 := Randomize(42)

	if p1.Duration != p2.Duration {
		t.Errorf("Duration mismatch: %v vs %v", p1.Duration, p2.Duration)
	}
	if p1.Seed != p2.Seed {
		t.Errorf("Seed mismatch: %d vs %d", p1.Seed, p2.Seed)
	}
	for i := range p1.Layers {
		if p1.Layers[i] != p2.Layers[i] {
			t.Errorf("Layer[%d] mismatch: %+v vs %+v", i, p1.Layers[i], p2.Layers[i])
		}
	}
	if p1.Noise != p2.Noise {
		t.Errorf("Noise mismatch: %+v vs %+v", p1.Noise, p2.Noise)
	}
	if p1.Filter != p2.Filter {
		t.Errorf("Filter mismatch: %+v vs %+v", p1.Filter, p2.Filter)
	}
}

func TestRandomize_DifferentSeeds(t *testing.T) {
	p1 := Randomize(1)
	p2 := Randomize(2)

	// With different seeds, at least some params should differ.
	// Duration is the most likely to differ since it is the first random draw.
	if p1.Duration == p2.Duration &&
		p1.Layers[0].BaseFreq == p2.Layers[0].BaseFreq &&
		p1.Filter.HighpassCutoff == p2.Filter.HighpassCutoff {
		t.Error("different seeds produced identical params; expected divergence in at least one field")
	}
}

func TestRandomize_ZeroSeed(t *testing.T) {
	p := Randomize(0)

	// Seed=0 should auto-generate, so stored seed should not be 0
	if p.Seed == 0 {
		t.Error("Randomize(0) should auto-generate a non-zero seed, but Seed is 0")
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("Randomize(0) produced invalid params: %v", err)
	}
}

func TestValidate_ValidParams(t *testing.T) {
	// Use each preset as a known-valid set of params
	for _, name := range AllPresets() {
		p, ok := GetPreset(name)
		if !ok {
			t.Fatalf("GetPreset(%q) returned false", name)
		}
		if err := p.Validate(); err != nil {
			t.Errorf("preset %q failed validation: %v", name, err)
		}
	}
}

func TestValidate_InvalidDuration(t *testing.T) {
	p := validBaseParams()
	p.Duration = 0

	err := p.Validate()
	if !errors.Is(err, ErrInvalidDuration) {
		t.Errorf("Validate() = %v, want ErrInvalidDuration", err)
	}
}

func TestValidate_InvalidSampleRate(t *testing.T) {
	p := validBaseParams()
	p.SampleRate = 0

	err := p.Validate()
	if !errors.Is(err, ErrInvalidSampleRate) {
		t.Errorf("Validate() = %v, want ErrInvalidSampleRate", err)
	}
}

func TestValidate_InvalidChannels(t *testing.T) {
	p := validBaseParams()
	p.Channels = 3

	err := p.Validate()
	if !errors.Is(err, ErrInvalidChannels) {
		t.Errorf("Validate() = %v, want ErrInvalidChannels", err)
	}
}

func TestValidate_InvalidAmplitude(t *testing.T) {
	p := validBaseParams()
	p.Layers[0].Amplitude = 1.5

	err := p.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want LayerValidationError wrapping ErrInvalidAmplitude")
	}

	var layerErr *LayerValidationError
	if !errors.As(err, &layerErr) {
		t.Fatalf("Validate() error type = %T, want *LayerValidationError", err)
	}
	if layerErr.Layer != 0 {
		t.Errorf("LayerValidationError.Layer = %d, want 0", layerErr.Layer)
	}
	if !errors.Is(err, ErrInvalidAmplitude) {
		t.Errorf("Validate() error does not wrap ErrInvalidAmplitude: %v", err)
	}
}

func TestValidate_InvalidLimiterLevel(t *testing.T) {
	tests := []struct {
		name  string
		level float64
	}{
		{"zero", 0},
		{"negative", -0.5},
		{"above_one", 1.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validBaseParams()
			p.Filter.LimiterLevel = tt.level

			err := p.Validate()
			if !errors.Is(err, ErrInvalidLimiterLevel) {
				t.Errorf("Validate() with LimiterLevel=%f: got %v, want ErrInvalidLimiterLevel", tt.level, err)
			}
		})
	}
}

// validBaseParams returns a minimal valid ScreamParams for mutation in tests.
func validBaseParams() ScreamParams {
	return ScreamParams{
		Duration:   3 * time.Second,
		SampleRate: 48000,
		Channels:   2,
		Layers: [5]LayerParams{
			{Type: LayerPrimaryScream, Amplitude: 0.4},
			{Type: LayerHarmonicSweep, Amplitude: 0.25},
			{Type: LayerHighShriek, Amplitude: 0.25},
			{Type: LayerNoiseBurst, Amplitude: 0.18},
			{Type: LayerBackgroundNoise, Amplitude: 0.1},
		},
		Filter: FilterParams{
			HighpassCutoff: 120,
			LowpassCutoff:  8000,
			CrusherBits:    8,
			CrusherMix:     0.5,
			CompRatio:      8,
			CompThreshold:  -20,
			CompAttack:     5,
			CompRelease:    50,
			VolumeBoostDB:  9,
			LimiterLevel:   0.95,
		},
	}
}
