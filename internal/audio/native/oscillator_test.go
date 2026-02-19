package native

import (
	"math"
	"testing"
)

func TestOscillator_Sin_FrequencyAccuracy(t *testing.T) {
	const (
		sampleRate = 48000
		freq       = 440.0
		numSamples = sampleRate // 1 second of audio
	)
	osc := NewOscillator(sampleRate)

	// Generate samples and count zero crossings.
	// A 440Hz sine wave has 880 zero crossings per second (2 per cycle).
	var prev float64
	zeroCrossings := 0
	for i := 0; i < numSamples; i++ {
		s := osc.Sin(freq)
		if i > 0 && prev*s < 0 {
			zeroCrossings++
		}
		prev = s
	}

	// Expected: ~880 zero crossings. Allow +/- 5 for edge effects.
	expected := 880
	tolerance := 5
	if zeroCrossings < expected-tolerance || zeroCrossings > expected+tolerance {
		t.Errorf("zero crossings = %d, want ~%d (tolerance %d)", zeroCrossings, expected, tolerance)
	}
}

func TestOscillator_Sin_AmplitudeBounds(t *testing.T) {
	const (
		sampleRate = 48000
		freq       = 440.0
		numSamples = 48000
	)
	osc := NewOscillator(sampleRate)

	for i := 0; i < numSamples; i++ {
		s := osc.Sin(freq)
		if s < -1.0 || s > 1.0 {
			t.Fatalf("sample %d: Sin() = %f, want in [-1, 1]", i, s)
		}
	}
}

func TestOscillator_Sin_PhaseContinuity(t *testing.T) {
	const (
		sampleRate = 48000
		freq       = 440.0
		numSamples = 480000 // 10 seconds - many phase wraps
	)
	osc := NewOscillator(sampleRate)

	for i := 0; i < numSamples; i++ {
		osc.Sin(freq)
		phase := osc.Phase()
		if phase < 0 || phase >= 1.0 {
			t.Fatalf("sample %d: Phase() = %f, want in [0, 1)", i, phase)
		}
	}
}

func TestOscillator_Saw_AmplitudeBounds(t *testing.T) {
	const (
		sampleRate = 48000
		freq       = 440.0
		numSamples = 48000
	)
	osc := NewOscillator(sampleRate)

	for i := 0; i < numSamples; i++ {
		s := osc.Saw(freq)
		// Saw maps [0,1) to [-1,1)
		if s < -1.0 || s >= 1.0 {
			t.Fatalf("sample %d: Saw() = %f, want in [-1, 1)", i, s)
		}
	}
}

func TestOscillator_Saw_FrequencyAccuracy(t *testing.T) {
	const (
		sampleRate = 48000
		freq       = 440.0
		numSamples = sampleRate // 1 second
	)
	osc := NewOscillator(sampleRate)

	// Count complete cycles via positive-to-negative transitions.
	// A sawtooth wave at 440Hz has 440 such transitions per second.
	var prev float64
	transitions := 0
	for i := 0; i < numSamples; i++ {
		s := osc.Saw(freq)
		// Sawtooth goes from -1 up to ~1, then jumps back to -1.
		// A positive-to-negative transition occurs when it wraps.
		if i > 0 && prev > 0 && s < 0 {
			transitions++
		}
		prev = s
	}

	expected := 440
	tolerance := 5
	if transitions < expected-tolerance || transitions > expected+tolerance {
		t.Errorf("positive-to-negative transitions = %d, want ~%d (tolerance %d)", transitions, expected, tolerance)
	}
}

func TestOscillator_Reset(t *testing.T) {
	osc := NewOscillator(48000)

	// Advance the oscillator
	for i := 0; i < 1000; i++ {
		osc.Sin(440)
	}

	if osc.Phase() == 0 {
		t.Fatal("Phase should be non-zero after generating samples")
	}

	osc.Reset()

	phase := osc.Phase()
	if phase != 0 {
		t.Errorf("after Reset(), Phase() = %f, want 0", phase)
	}
}

// BenchmarkOscillator_Sin benchmarks sine generation at audio rate.
func BenchmarkOscillator_Sin(b *testing.B) {
	osc := NewOscillator(48000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		osc.Sin(440)
	}
}

// BenchmarkOscillator_Saw benchmarks sawtooth generation at audio rate.
func BenchmarkOscillator_Saw(b *testing.B) {
	osc := NewOscillator(48000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		osc.Saw(440)
	}
}

// TestOscillator_Sin_KnownValues tests specific known values of sine at key phases.
func TestOscillator_Sin_KnownValues(t *testing.T) {
	// At sampleRate=4, freq=1, each sample advances phase by 0.25.
	// Phase 0 -> sin(0) = 0
	// Phase 0.25 -> sin(pi/2) = 1
	// Phase 0.5 -> sin(pi) = 0
	// Phase 0.75 -> sin(3pi/2) = -1
	osc := NewOscillator(4)

	s0 := osc.Sin(1)
	if math.Abs(s0) > 1e-10 {
		t.Errorf("sample 0: got %f, want 0", s0)
	}

	s1 := osc.Sin(1)
	if math.Abs(s1-1.0) > 1e-10 {
		t.Errorf("sample 1: got %f, want 1", s1)
	}

	s2 := osc.Sin(1)
	if math.Abs(s2) > 1e-10 {
		t.Errorf("sample 2: got %f, want 0", s2)
	}

	s3 := osc.Sin(1)
	if math.Abs(s3-(-1.0)) > 1e-10 {
		t.Errorf("sample 3: got %f, want -1", s3)
	}
}
