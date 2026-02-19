package native

import (
	"math"
	"testing"

	"github.com/JamesPrial/go-scream/internal/audio"
)

const testSampleRate = 48000

// validLayerParams returns layer params suitable for each layer type.
func validPrimaryScreamParams() audio.LayerParams {
	return audio.LayerParams{
		Type:      audio.LayerPrimaryScream,
		BaseFreq:  500,
		FreqRange: 1500,
		JumpRate:  10,
		Amplitude: 0.4,
		Rise:      1.2,
		Seed:      4242,
	}
}

func validHarmonicSweepParams() audio.LayerParams {
	return audio.LayerParams{
		Type:      audio.LayerHarmonicSweep,
		BaseFreq:  350,
		SweepRate: 500,
		FreqRange: 800,
		JumpRate:  6,
		Amplitude: 0.25,
		Seed:      3000,
	}
}

func validHighShriekParams() audio.LayerParams {
	return audio.LayerParams{
		Type:      audio.LayerHighShriek,
		BaseFreq:  1200,
		FreqRange: 1600,
		JumpRate:  20,
		Amplitude: 0.25,
		Rise:      2.5,
		Seed:      7000,
	}
}

func validNoiseBurstParams() audio.LayerParams {
	return audio.LayerParams{
		Type:      audio.LayerNoiseBurst,
		Amplitude: 0.18,
		Seed:      4000,
	}
}

func validNoiseParams() audio.NoiseParams {
	return audio.NoiseParams{
		BurstRate: 8,
		Threshold: 0.7,
		BurstAmp:  0.18,
		FloorAmp:  0.1,
		BurstSeed: 4000,
	}
}

// --- PrimaryScreamLayer Tests ---

func TestPrimaryScreamLayer_NonZeroOutput(t *testing.T) {
	layer := NewPrimaryScreamLayer(validPrimaryScreamParams(), testSampleRate)

	hasNonZero := false
	for i := 0; i < 1000; i++ {
		t := float64(i) / float64(testSampleRate)
		s := layer.Sample(t)
		if s != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("PrimaryScreamLayer produced all zero samples for first 1000 samples")
	}
}

func TestPrimaryScreamLayer_AmplitudeBounds(t *testing.T) {
	p := validPrimaryScreamParams()
	layer := NewPrimaryScreamLayer(p, testSampleRate)

	// For t in [0, 3], envelope = amp * (1 + rise*t)
	// Max envelope at t=3: 0.4 * (1 + 1.2*3) = 0.4 * 4.6 = 1.84
	// Oscillator output is in [-1, 1], so max absolute value = 1.84
	maxEnvelope := p.Amplitude * (1 + p.Rise*3.0)
	tolerance := 0.01 // small tolerance for floating point

	numSamples := 3 * testSampleRate
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(testSampleRate)
		s := layer.Sample(tSec)
		if math.Abs(s) > maxEnvelope+tolerance {
			t.Fatalf("sample %d (t=%.4f): |Sample()| = %f, exceeds max envelope %f",
				i, tSec, math.Abs(s), maxEnvelope)
		}
	}
}

// --- HarmonicSweepLayer Tests ---

func TestHarmonicSweepLayer_NonZeroOutput(t *testing.T) {
	layer := NewHarmonicSweepLayer(validHarmonicSweepParams(), testSampleRate)

	hasNonZero := false
	for i := 0; i < 1000; i++ {
		tSec := float64(i) / float64(testSampleRate)
		s := layer.Sample(tSec)
		if s != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("HarmonicSweepLayer produced all zero samples for first 1000 samples")
	}
}

// --- HighShriekLayer Tests ---

func TestHighShriekLayer_NonZeroOutput(t *testing.T) {
	layer := NewHighShriekLayer(validHighShriekParams(), testSampleRate)

	hasNonZero := false
	for i := 0; i < 1000; i++ {
		tSec := float64(i) / float64(testSampleRate)
		s := layer.Sample(tSec)
		if s != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("HighShriekLayer produced all zero samples for first 1000 samples")
	}
}

func TestHighShriekLayer_EnvelopeRises(t *testing.T) {
	layer := NewHighShriekLayer(validHighShriekParams(), testSampleRate)

	// Collect average absolute amplitude over two different time windows.
	// Early window: t in [0.0, 0.5)
	// Late window: t in [2.0, 2.5)
	// The envelope rises with time, so the late window should have higher average.
	calcAvgAbs := func(startSec, endSec float64) float64 {
		startSample := int(startSec * float64(testSampleRate))
		endSample := int(endSec * float64(testSampleRate))
		var sum float64
		count := 0

		// We need a fresh layer for each window since the oscillator is stateful.
		// Instead, we'll generate all samples up to the end and measure the windows.
		l := NewHighShriekLayer(validHighShriekParams(), testSampleRate)
		for i := 0; i < endSample; i++ {
			tSec := float64(i) / float64(testSampleRate)
			s := l.Sample(tSec)
			if i >= startSample {
				sum += math.Abs(s)
				count++
			}
		}
		return sum / float64(count)
	}

	// Suppress unused variable warning for the first layer created above
	_ = layer

	earlyAvg := calcAvgAbs(0.0, 0.5)
	lateAvg := calcAvgAbs(2.0, 2.5)

	if lateAvg <= earlyAvg {
		t.Errorf("envelope should rise: early avg abs = %f, late avg abs = %f", earlyAvg, lateAvg)
	}
}

// --- NoiseBurstLayer Tests ---

func TestNoiseBurstLayer_HasSilentAndActiveSegments(t *testing.T) {
	np := validNoiseParams()
	lp := validNoiseBurstParams()
	layer := NewNoiseBurstLayer(lp, np, testSampleRate)

	hasZero := false
	hasNonZero := false

	// Sample over 3 seconds
	numSamples := 3 * testSampleRate
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(testSampleRate)
		s := layer.Sample(tSec)
		if s == 0 {
			hasZero = true
		} else {
			hasNonZero = true
		}
		if hasZero && hasNonZero {
			break
		}
	}

	if !hasZero {
		t.Error("NoiseBurstLayer had no silent samples; expected gated silence")
	}
	if !hasNonZero {
		t.Error("NoiseBurstLayer had no active samples; expected noise bursts")
	}
}

// --- BackgroundNoiseLayer Tests ---

func TestBackgroundNoiseLayer_ContinuousOutput(t *testing.T) {
	np := validNoiseParams()
	layer := NewBackgroundNoiseLayer(np)

	// Background noise should be (almost) always non-zero.
	// With pseudo-random values, exact zero is extremely unlikely.
	zeroCount := 0
	numSamples := testSampleRate // 1 second
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(testSampleRate)
		s := layer.Sample(tSec)
		if s == 0 {
			zeroCount++
		}
	}

	// Allow a tiny fraction to be zero (extremely unlikely for noise)
	maxZeros := numSamples / 100 // less than 1%
	if zeroCount > maxZeros {
		t.Errorf("BackgroundNoiseLayer had %d zero samples out of %d; expected continuous noise", zeroCount, numSamples)
	}
}

// --- LayerMixer Tests ---

// mockLayer is a test double that returns a fixed value.
type mockLayer struct {
	value float64
}

func (m *mockLayer) Sample(_ float64) float64 {
	return m.value
}

func TestLayerMixer_SumsLayers(t *testing.T) {
	l1 := &mockLayer{value: 0.3}
	l2 := &mockLayer{value: 0.2}
	l3 := &mockLayer{value: 0.1}

	mixer := NewLayerMixer(l1, l2, l3)
	got := mixer.Sample(0)
	want := 0.6

	if math.Abs(got-want) > 1e-10 {
		t.Errorf("LayerMixer.Sample() = %f, want %f", got, want)
	}
}

func TestLayerMixer_ClampsOutput(t *testing.T) {
	l1 := &mockLayer{value: 0.7}
	l2 := &mockLayer{value: 0.5}

	mixer := NewLayerMixer(l1, l2)
	got := mixer.Sample(0)

	// 0.7 + 0.5 = 1.2, should be clamped to 1.0
	if got != 1.0 {
		t.Errorf("LayerMixer.Sample() = %f, want 1.0 (clamped)", got)
	}
}

func TestLayerMixer_ClampsNegative(t *testing.T) {
	l1 := &mockLayer{value: -0.7}
	l2 := &mockLayer{value: -0.5}

	mixer := NewLayerMixer(l1, l2)
	got := mixer.Sample(0)

	// -0.7 + -0.5 = -1.2, should be clamped to -1.0
	if got != -1.0 {
		t.Errorf("LayerMixer.Sample() = %f, want -1.0 (clamped)", got)
	}
}

func TestLayerMixer_ZeroLayers(t *testing.T) {
	mixer := NewLayerMixer()
	got := mixer.Sample(0)

	if got != 0 {
		t.Errorf("LayerMixer.Sample() with zero layers = %f, want 0", got)
	}
}

// --- Benchmarks ---

func BenchmarkPrimaryScreamLayer(b *testing.B) {
	layer := NewPrimaryScreamLayer(validPrimaryScreamParams(), testSampleRate)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layer.Sample(float64(i) / float64(testSampleRate))
	}
}

func BenchmarkLayerMixer(b *testing.B) {
	layers := []Layer{
		NewPrimaryScreamLayer(validPrimaryScreamParams(), testSampleRate),
		NewHarmonicSweepLayer(validHarmonicSweepParams(), testSampleRate),
		NewHighShriekLayer(validHighShriekParams(), testSampleRate),
		NewNoiseBurstLayer(validNoiseBurstParams(), validNoiseParams(), testSampleRate),
		NewBackgroundNoiseLayer(validNoiseParams()),
	}
	mixer := NewLayerMixer(layers...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mixer.Sample(float64(i) / float64(testSampleRate))
	}
}
