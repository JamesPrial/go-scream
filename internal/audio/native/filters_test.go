package native

import (
	"math"
	"testing"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// --- HighpassFilter Tests ---

func TestHighpassFilter_RemovesDC(t *testing.T) {
	hp := newHighpassFilter(100, 48000)

	// Feed constant 1.0 (DC) for 1000 samples. Output should decay toward 0.
	var last float64
	for i := 0; i < 1000; i++ {
		last = hp.Process(1.0)
	}

	if math.Abs(last) > 0.1 {
		t.Errorf("after 1000 samples of DC input, output = %f, want near 0", last)
	}
}

func TestHighpassFilter_PassesHighFreq(t *testing.T) {
	hp := newHighpassFilter(100, 48000)

	// Alternating +1/-1 represents a high-frequency signal (Nyquist/2 = 24kHz).
	// Warm up the filter first.
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			hp.Process(1.0)
		} else {
			hp.Process(-1.0)
		}
	}

	// Now measure output magnitude
	var maxOut float64
	for i := 0; i < 100; i++ {
		var in float64
		if i%2 == 0 {
			in = 1.0
		} else {
			in = -1.0
		}
		out := hp.Process(in)
		if math.Abs(out) > maxOut {
			maxOut = math.Abs(out)
		}
	}

	// High frequency should pass through with magnitude near 1.0
	if maxOut < 0.8 {
		t.Errorf("highpass output magnitude for high freq = %f, want >= 0.8", maxOut)
	}
}

// --- LowpassFilter Tests ---

func TestLowpassFilter_PassesDC(t *testing.T) {
	lp := newLowpassFilter(8000, 48000)

	// Constant 1.0 (DC) should converge to 1.0 after warmup.
	var last float64
	for i := 0; i < 1000; i++ {
		last = lp.Process(1.0)
	}

	if math.Abs(last-1.0) > 0.05 {
		t.Errorf("after 1000 samples of DC input, output = %f, want near 1.0", last)
	}
}

func TestLowpassFilter_AttenuatesHighFreq(t *testing.T) {
	lp := newLowpassFilter(1000, 48000)

	// Warm up with alternating signal
	for i := 0; i < 200; i++ {
		if i%2 == 0 {
			lp.Process(1.0)
		} else {
			lp.Process(-1.0)
		}
	}

	// Measure output magnitude for alternating signal (Nyquist freq)
	var maxOut float64
	for i := 0; i < 100; i++ {
		var in float64
		if i%2 == 0 {
			in = 1.0
		} else {
			in = -1.0
		}
		out := lp.Process(in)
		if math.Abs(out) > maxOut {
			maxOut = math.Abs(out)
		}
	}

	// Output should be significantly attenuated
	if maxOut >= 0.5 {
		t.Errorf("lowpass output magnitude for high freq = %f, want < 0.5", maxOut)
	}
}

// --- Bitcrusher Tests ---

func TestBitcrusher_FullMix(t *testing.T) {
	bc := newBitcrusher(4, 1.0)

	// With 4 bits, signal is quantized to 2^4 = 16 levels.
	// A value like 0.123456 should snap to the nearest quantization step.
	tests := []struct {
		name  string
		input float64
	}{
		{"positive", 0.123456},
		{"negative", -0.654321},
		{"zero", 0.0},
		{"one", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := bc.Process(tt.input)
			// With 4 bits, quantization step = 2 / 16 = 0.125
			// Output should be a multiple of 1/8 (approximately)
			step := 2.0 / math.Pow(2, 4)
			// The quantized value should be within one step of the input
			if math.Abs(out-tt.input) > step+1e-10 {
				t.Errorf("Bitcrusher(%f) = %f, differs by more than one quantization step (%f)",
					tt.input, out, step)
			}
		})
	}
}

func TestBitcrusher_ZeroMix(t *testing.T) {
	bc := newBitcrusher(4, 0.0)

	// With mix=0, output should equal input exactly (dry signal only)
	tests := []float64{0.0, 0.5, -0.5, 0.999, -0.999}
	for _, in := range tests {
		out := bc.Process(in)
		if out != in {
			t.Errorf("Bitcrusher(mix=0).Process(%f) = %f, want %f", in, out, in)
		}
	}
}

func TestBitcrusher_Blend(t *testing.T) {
	// mix=0.5 should blend 50% clean with 50% crushed
	bc := newBitcrusher(4, 0.5)

	in := 0.3
	out := bc.Process(in)

	// Output should be between the clean signal and the fully crushed signal
	bcFull := newBitcrusher(4, 1.0)
	crushed := bcFull.Process(in)

	expected := 0.5*in + 0.5*crushed
	if math.Abs(out-expected) > 1e-10 {
		t.Errorf("Bitcrusher(mix=0.5).Process(%f) = %f, want %f (blend of %f and %f)",
			in, out, expected, in, crushed)
	}
}

// --- Compressor Tests ---

func TestCompressor_BelowThreshold(t *testing.T) {
	// Threshold at -20dB ~ amplitude 0.1. Signal at 0.01 is well below.
	comp := newCompressor(8, -20, 5, 50, 48000)

	// Feed quiet signal and let compressor settle
	for i := 0; i < 1000; i++ {
		comp.Process(0.01)
	}

	// After settling, quiet signal should pass approximately unchanged
	out := comp.Process(0.01)
	if math.Abs(out-0.01) > 0.005 {
		t.Errorf("Compressor below threshold: Process(0.01) = %f, want near 0.01", out)
	}
}

func TestCompressor_AboveThreshold(t *testing.T) {
	// Threshold at -6dB ~ amplitude 0.5. Signal at 0.9 is above.
	comp := newCompressor(8, -6, 5, 50, 48000)

	// Feed loud signal to let compressor engage
	var out float64
	for i := 0; i < 5000; i++ {
		out = comp.Process(0.9)
	}

	// Compressed output should be less than input
	if out >= 0.9 {
		t.Errorf("Compressor above threshold: Process(0.9) = %f, want < 0.9", out)
	}
	// But still positive (signal not destroyed)
	if out <= 0 {
		t.Errorf("Compressor above threshold: Process(0.9) = %f, want > 0", out)
	}
}

func TestCompressor_PreservesSign(t *testing.T) {
	comp := newCompressor(8, -20, 5, 50, 48000)

	// Feed negative signal
	for i := 0; i < 1000; i++ {
		comp.Process(-0.8)
	}

	out := comp.Process(-0.8)
	if out >= 0 {
		t.Errorf("Compressor should preserve sign: Process(-0.8) = %f, want < 0", out)
	}
}

// --- VolumeBoost Tests ---

func TestVolumeBoost_ZeroDB(t *testing.T) {
	vb := newVolumeBoost(0)

	tests := []float64{0.0, 0.5, -0.5, 1.0, -1.0}
	for _, in := range tests {
		out := vb.Process(in)
		if math.Abs(out-in) > 1e-10 {
			t.Errorf("VolumeBoost(0dB).Process(%f) = %f, want %f", in, out, in)
		}
	}
}

func TestVolumeBoost_6dB(t *testing.T) {
	vb := newVolumeBoost(6)

	in := 0.5
	out := vb.Process(in)

	// 6dB ~ 10^(6/20) ~ 1.995 multiplier, so 0.5 * ~2 = ~1.0
	expected := in * math.Pow(10, 6.0/20.0)
	tolerance := 0.05
	if math.Abs(out-expected) > tolerance {
		t.Errorf("VolumeBoost(6dB).Process(%f) = %f, want approximately %f", in, out, expected)
	}
}

func TestVolumeBoost_NegativeDB(t *testing.T) {
	vb := newVolumeBoost(-6)

	in := 1.0
	out := vb.Process(in)

	// -6dB ~ 0.501 multiplier
	if out >= in {
		t.Errorf("VolumeBoost(-6dB).Process(%f) = %f, want < %f (attenuation)", in, out, in)
	}
	if out <= 0 {
		t.Errorf("VolumeBoost(-6dB).Process(%f) = %f, want > 0", in, out)
	}
}

// --- Limiter Tests ---

func TestLimiter_WithinRange(t *testing.T) {
	lim := newLimiter(0.95)

	out := lim.Process(0.5)
	if out != 0.5 {
		t.Errorf("Limiter(0.95).Process(0.5) = %f, want 0.5", out)
	}
}

func TestLimiter_ClipsPositive(t *testing.T) {
	lim := newLimiter(0.95)

	out := lim.Process(1.5)
	if out != 0.95 {
		t.Errorf("Limiter(0.95).Process(1.5) = %f, want 0.95", out)
	}
}

func TestLimiter_ClipsNegative(t *testing.T) {
	lim := newLimiter(0.95)

	out := lim.Process(-1.5)
	if out != -0.95 {
		t.Errorf("Limiter(0.95).Process(-1.5) = %f, want -0.95", out)
	}
}

// --- FilterChain Tests ---

func TestFilterChain_OrderMatters(t *testing.T) {
	// VolumeBoost(12dB) then Limiter(0.95) vs Limiter(0.95) then VolumeBoost(12dB)
	// should produce different results for a signal that gets boosted above the limit.

	// Chain 1: Boost then Limit
	chain1 := newFilterChain(newVolumeBoost(12), newLimiter(0.95))
	out1 := chain1.Process(0.5)

	// Chain 2: Limit then Boost
	chain2 := newFilterChain(newLimiter(0.95), newVolumeBoost(12))
	out2 := chain2.Process(0.5)

	// In chain1, 0.5 gets boosted to ~2.0, then limited to 0.95
	// In chain2, 0.5 passes limiter unchanged, then boosted to ~2.0
	if out1 == out2 {
		t.Errorf("filter chain order should matter: boost+limit = %f, limit+boost = %f", out1, out2)
	}

	// Chain1 output should be clamped
	if out1 > 0.95+1e-10 {
		t.Errorf("boost+limit output = %f, want <= 0.95", out1)
	}

	// Chain2 output should be boosted above limit
	if out2 <= 0.95 {
		t.Errorf("limit+boost output = %f, want > 0.95 (boosted after limit)", out2)
	}
}

func TestFilterChainFromParams_ClassicPreset(t *testing.T) {
	// Get classic preset's filter params
	classic, ok := audio.GetPreset(audio.PresetClassic)
	if !ok {
		t.Fatal("classic preset not found")
	}

	chain := newFilterChainFromParams(classic.Filter, classic.SampleRate)

	// Feed a sine-like signal through the chain. All output should be bounded
	// by the limiter level.
	osc := newOscillator(classic.SampleRate)
	for i := 0; i < classic.SampleRate; i++ {
		in := osc.sin(500) // 500Hz sine
		out := chain.Process(in)

		if math.Abs(out) > classic.Filter.LimiterLevel+1e-10 {
			t.Fatalf("sample %d: |output| = %f exceeds limiter level %f",
				i, math.Abs(out), classic.Filter.LimiterLevel)
		}
	}
}

// --- Filter Interface Compliance ---

func TestHighpassFilter_ImplementsFilter(t *testing.T) {
	var _ filter = newHighpassFilter(100, 48000)
}

func TestLowpassFilter_ImplementsFilter(t *testing.T) {
	var _ filter = newLowpassFilter(8000, 48000)
}

func TestBitcrusher_ImplementsFilter(t *testing.T) {
	var _ filter = newBitcrusher(8, 0.5)
}

func TestCompressor_ImplementsFilter(t *testing.T) {
	var _ filter = newCompressor(8, -20, 5, 50, 48000)
}

func TestVolumeBoost_ImplementsFilter(t *testing.T) {
	var _ filter = newVolumeBoost(6)
}

func TestLimiter_ImplementsFilter(t *testing.T) {
	var _ filter = newLimiter(0.95)
}

func TestFilterChain_ImplementsFilter(t *testing.T) {
	var _ filter = newFilterChain()
}

// --- Benchmarks ---

func BenchmarkFilterChain_Classic(b *testing.B) {
	classic, ok := audio.GetPreset(audio.PresetClassic)
	if !ok {
		b.Fatal("classic preset not found")
	}
	chain := newFilterChainFromParams(classic.Filter, classic.SampleRate)
	osc := newOscillator(classic.SampleRate)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain.Process(osc.sin(500))
	}
}
