package ffmpeg

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// classicParams returns the "classic" preset params for deterministic testing.
func classicParams() audio.ScreamParams {
	return audio.ScreamParams{
		Duration:   3 * time.Second,
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

// --- BuildArgs tests ---

func Test_BuildArgs_ContainsLavfiInput(t *testing.T) {
	args := buildArgs(classicParams())
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-f lavfi") {
		t.Errorf("buildArgs() should contain '-f lavfi', got args: %v", args)
	}
}

func Test_BuildArgs_ContainsAevalsrc(t *testing.T) {
	args := buildArgs(classicParams())

	found := false
	for _, arg := range args {
		if strings.Contains(arg, "aevalsrc=") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("buildArgs() should contain an arg with 'aevalsrc=', got: %v", args)
	}
}

func Test_BuildArgs_ContainsAudioFilter(t *testing.T) {
	args := buildArgs(classicParams())

	afIndex := -1
	for i, arg := range args {
		if arg == "-af" {
			afIndex = i
			break
		}
	}
	if afIndex == -1 {
		t.Fatalf("buildArgs() should contain '-af', got: %v", args)
	}
	if afIndex >= len(args)-1 {
		t.Fatalf("buildArgs() has '-af' as last arg with no filter value")
	}
	filterVal := args[afIndex+1]
	if filterVal == "" {
		t.Error("buildArgs() '-af' value should not be empty")
	}
}

func Test_BuildArgs_ContainsOutputFormat(t *testing.T) {
	args := buildArgs(classicParams())
	joined := strings.Join(args, " ")

	checks := []string{
		"-f s16le",
		"-acodec pcm_s16le",
	}
	for _, check := range checks {
		if !strings.Contains(joined, check) {
			t.Errorf("buildArgs() should contain '%s', got: %s", check, joined)
		}
	}
}

func Test_BuildArgs_ContainsChannels(t *testing.T) {
	params := classicParams()
	args := buildArgs(params)
	joined := strings.Join(args, " ")

	expected := fmt.Sprintf("-ac %d", params.Channels)
	if !strings.Contains(joined, expected) {
		t.Errorf("buildArgs() should contain '%s', got: %s", expected, joined)
	}
}

func Test_BuildArgs_ContainsSampleRate(t *testing.T) {
	params := classicParams()
	args := buildArgs(params)
	joined := strings.Join(args, " ")

	expected := fmt.Sprintf("-ar %d", params.SampleRate)
	if !strings.Contains(joined, expected) {
		t.Errorf("buildArgs() should contain '%s', got: %s", expected, joined)
	}
}

func Test_BuildArgs_LastArgIsPipe(t *testing.T) {
	args := buildArgs(classicParams())

	if len(args) == 0 {
		t.Fatal("buildArgs() returned empty args")
	}
	lastArg := args[len(args)-1]
	if lastArg != "pipe:1" {
		t.Errorf("buildArgs() last arg = %q, want %q", lastArg, "pipe:1")
	}
}

func Test_BuildArgs_ContainsDuration(t *testing.T) {
	params := classicParams()
	args := buildArgs(params)

	// Duration should appear somewhere in the aevalsrc arg (as d= or duration=)
	// or as a separate -t argument.
	joined := strings.Join(args, " ")
	durationSeconds := params.Duration.Seconds()
	durationStr := fmt.Sprintf("%g", durationSeconds)

	// Check that the duration value appears somewhere in the args.
	// It could be in the aevalsrc expression as :d=3 or as a -t argument.
	hasDuration := false
	for _, arg := range args {
		if strings.Contains(arg, "aevalsrc=") {
			// Check for duration in aevalsrc (e.g., :d=3)
			if strings.Contains(arg, fmt.Sprintf("d=%s", durationStr)) ||
				strings.Contains(arg, fmt.Sprintf("d=%d", int(durationSeconds))) ||
				strings.Contains(arg, fmt.Sprintf("duration=%s", durationStr)) ||
				strings.Contains(arg, fmt.Sprintf("duration=%d", int(durationSeconds))) {
				hasDuration = true
				break
			}
		}
	}
	if !hasDuration {
		// Check for -t flag instead
		if strings.Contains(joined, "-t") {
			hasDuration = true
		}
	}
	if !hasDuration {
		t.Errorf("buildArgs() should contain duration (%v) in aevalsrc or as -t flag, got: %s", params.Duration, joined)
	}
}

func Test_BuildArgs_MonoParams(t *testing.T) {
	params := classicParams()
	params.Channels = 1
	args := buildArgs(params)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-ac 1") {
		t.Errorf("buildArgs() with mono should contain '-ac 1', got: %s", joined)
	}
}

func Test_BuildArgs_DifferentSampleRate(t *testing.T) {
	params := classicParams()
	params.SampleRate = 44100
	args := buildArgs(params)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-ar 44100") {
		t.Errorf("buildArgs() with 44100 should contain '-ar 44100', got: %s", joined)
	}
}

// --- buildAevalsrcExpr tests ---

func Test_buildAevalsrcExpr_ContainsSin(t *testing.T) {
	expr := buildAevalsrcExpr(classicParams())

	if !strings.Contains(expr, "sin") {
		t.Errorf("buildAevalsrcExpr() should contain 'sin' for tonal layers, got: %s", expr)
	}
}

func Test_buildAevalsrcExpr_ContainsRandom(t *testing.T) {
	expr := buildAevalsrcExpr(classicParams())

	if !strings.Contains(expr, "random(") {
		t.Errorf("buildAevalsrcExpr() should contain 'random(' for noise layers, got: %s", expr)
	}
}

func Test_buildAevalsrcExpr_ContainsPI(t *testing.T) {
	expr := buildAevalsrcExpr(classicParams())

	if !strings.Contains(expr, "PI") {
		t.Errorf("buildAevalsrcExpr() should contain 'PI' for sine wave computation, got: %s", expr)
	}
}

func Test_buildAevalsrcExpr_NonEmptyForAllPresets(t *testing.T) {
	presets := []struct {
		name string
		seed int64
	}{
		{"seed_1", 1},
		{"seed_42", 42},
		{"seed_100", 100},
		{"seed_9999", 9999},
		{"seed_12345", 12345},
		{"seed_99999", 99999},
	}

	for _, tt := range presets {
		t.Run(tt.name, func(t *testing.T) {
			params := audio.Randomize(tt.seed)
			expr := buildAevalsrcExpr(params)
			if expr == "" {
				t.Errorf("buildAevalsrcExpr() returned empty string for Randomize(%d)", tt.seed)
			}
		})
	}
}

func Test_buildAevalsrcExpr_ZeroAmplitudeLayer(t *testing.T) {
	params := classicParams()
	// Set all layer amplitudes to zero
	for i := range params.Layers {
		params.Layers[i].Amplitude = 0
	}
	params.Noise.BurstAmp = 0
	params.Noise.FloorAmp = 0

	expr := buildAevalsrcExpr(params)
	// Should still produce a valid (non-empty) expression even with zero amplitudes
	if expr == "" {
		t.Error("buildAevalsrcExpr() should produce valid output even with zero amplitude layers")
	}
}

// --- buildFilterChain tests ---

func Test_buildFilterChain_ContainsHighpass(t *testing.T) {
	filter := audio.FilterParams{
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
	}
	chain := buildFilterChain(filter)

	if !strings.Contains(chain, "highpass=f=120") {
		t.Errorf("buildFilterChain() should contain 'highpass=f=120', got: %s", chain)
	}
}

func Test_buildFilterChain_ContainsLowpass(t *testing.T) {
	filter := audio.FilterParams{
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
	}
	chain := buildFilterChain(filter)

	if !strings.Contains(chain, "lowpass=f=8000") {
		t.Errorf("buildFilterChain() should contain 'lowpass=f=8000', got: %s", chain)
	}
}

func Test_buildFilterChain_ContainsAcrusher(t *testing.T) {
	filter := audio.FilterParams{
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
	}
	chain := buildFilterChain(filter)

	if !strings.Contains(chain, "acrusher") {
		t.Errorf("buildFilterChain() should contain 'acrusher', got: %s", chain)
	}
	// Should contain bits=8 somewhere in the acrusher section
	if !strings.Contains(chain, "bits=8") {
		t.Errorf("buildFilterChain() should contain 'bits=8' for CrusherBits=8, got: %s", chain)
	}
}

func Test_buildFilterChain_ContainsAcompressor(t *testing.T) {
	filter := audio.FilterParams{
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
	}
	chain := buildFilterChain(filter)

	if !strings.Contains(chain, "acompressor") {
		t.Errorf("buildFilterChain() should contain 'acompressor', got: %s", chain)
	}
	if !strings.Contains(chain, "ratio=8") {
		t.Errorf("buildFilterChain() should contain compressor ratio, got: %s", chain)
	}
	if !strings.Contains(chain, "threshold=") {
		t.Errorf("buildFilterChain() should contain compressor threshold, got: %s", chain)
	}
}

func Test_buildFilterChain_ContainsVolume(t *testing.T) {
	filter := audio.FilterParams{
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
	}
	chain := buildFilterChain(filter)

	if !strings.Contains(chain, "volume=") {
		t.Errorf("buildFilterChain() should contain 'volume=', got: %s", chain)
	}
	// Should contain dB value
	if !strings.Contains(chain, "dB") && !strings.Contains(chain, "9") {
		t.Errorf("buildFilterChain() volume should reference dB value, got: %s", chain)
	}
}

func Test_buildFilterChain_ContainsAlimiter(t *testing.T) {
	filter := audio.FilterParams{
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
	}
	chain := buildFilterChain(filter)

	if !strings.Contains(chain, "alimiter") {
		t.Errorf("buildFilterChain() should contain 'alimiter', got: %s", chain)
	}
	if !strings.Contains(chain, "limit=") {
		t.Errorf("buildFilterChain() should contain 'limit=' for limiter level, got: %s", chain)
	}
}

func Test_buildFilterChain_FilterOrder(t *testing.T) {
	filter := audio.FilterParams{
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
	}
	chain := buildFilterChain(filter)

	// Verify ordering: highpass < lowpass < acrusher < acompressor < volume < alimiter
	positions := map[string]int{
		"highpass":    strings.Index(chain, "highpass"),
		"lowpass":     strings.Index(chain, "lowpass"),
		"acrusher":    strings.Index(chain, "acrusher"),
		"acompressor": strings.Index(chain, "acompressor"),
		"volume":      strings.Index(chain, "volume"),
		"alimiter":    strings.Index(chain, "alimiter"),
	}

	for name, pos := range positions {
		if pos == -1 {
			t.Fatalf("buildFilterChain() missing filter '%s' in chain: %s", name, chain)
		}
	}

	orderedFilters := []string{"highpass", "lowpass", "acrusher", "acompressor", "volume", "alimiter"}
	for i := 0; i < len(orderedFilters)-1; i++ {
		curr := orderedFilters[i]
		next := orderedFilters[i+1]
		if positions[curr] >= positions[next] {
			t.Errorf("buildFilterChain() filter order violation: '%s' (pos %d) should come before '%s' (pos %d) in chain: %s",
				curr, positions[curr], next, positions[next], chain)
		}
	}
}

// --- layerExpr tests ---

func Test_layerExpr_PrimaryScream(t *testing.T) {
	layer := audio.LayerParams{
		Type:      audio.LayerPrimaryScream,
		BaseFreq:  500,
		FreqRange: 1500,
		JumpRate:  10,
		Amplitude: 0.4,
		Rise:      1.2,
		Seed:      4242,
	}
	noise := audio.NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000}

	expr := layerExpr(layer, noise, 42, 0)

	if !strings.Contains(expr, "sin") {
		t.Errorf("layerExpr(PrimaryScream) should contain 'sin', got: %s", expr)
	}
	if !strings.Contains(expr, "500") {
		t.Errorf("layerExpr(PrimaryScream) should contain base freq '500', got: %s", expr)
	}
}

func Test_layerExpr_HarmonicSweep(t *testing.T) {
	layer := audio.LayerParams{
		Type:      audio.LayerHarmonicSweep,
		BaseFreq:  350,
		SweepRate: 500,
		FreqRange: 800,
		JumpRate:  6,
		Amplitude: 0.25,
		Seed:      3000,
	}
	noise := audio.NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000}

	expr := layerExpr(layer, noise, 42, 1)

	if !strings.Contains(expr, "sin") {
		t.Errorf("layerExpr(HarmonicSweep) should contain 'sin', got: %s", expr)
	}
	if !strings.Contains(expr, "500") {
		t.Errorf("layerExpr(HarmonicSweep) should reference sweep rate '500', got: %s", expr)
	}
}

func Test_layerExpr_HighShriek(t *testing.T) {
	layer := audio.LayerParams{
		Type:      audio.LayerHighShriek,
		BaseFreq:  1200,
		FreqRange: 1600,
		JumpRate:  20,
		Amplitude: 0.25,
		Rise:      2.5,
		Seed:      7000,
	}
	noise := audio.NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000}

	expr := layerExpr(layer, noise, 42, 2)

	if !strings.Contains(expr, "sin") {
		t.Errorf("layerExpr(HighShriek) should contain 'sin', got: %s", expr)
	}
	if !strings.Contains(expr, "1200") {
		t.Errorf("layerExpr(HighShriek) should contain base freq '1200', got: %s", expr)
	}
}

func Test_layerExpr_NoiseBurst(t *testing.T) {
	layer := audio.LayerParams{
		Type:      audio.LayerNoiseBurst,
		Amplitude: 0.18,
		Seed:      4000,
	}
	noise := audio.NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000}

	expr := layerExpr(layer, noise, 42, 3)

	if !strings.Contains(expr, "random") {
		t.Errorf("layerExpr(NoiseBurst) should contain 'random', got: %s", expr)
	}
	// Should reference the threshold for gating
	if !strings.Contains(expr, "0.7") {
		t.Errorf("layerExpr(NoiseBurst) should reference threshold '0.7', got: %s", expr)
	}
}

func Test_layerExpr_BackgroundNoise(t *testing.T) {
	layer := audio.LayerParams{
		Type:      audio.LayerBackgroundNoise,
		Amplitude: 0.1,
	}
	noise := audio.NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000}

	expr := layerExpr(layer, noise, 42, 4)

	if !strings.Contains(expr, "random") {
		t.Errorf("layerExpr(BackgroundNoise) should contain 'random', got: %s", expr)
	}
	// Should reference floor amplitude
	if !strings.Contains(expr, "0.1") {
		t.Errorf("layerExpr(BackgroundNoise) should reference floor amp '0.1', got: %s", expr)
	}
}

func Test_layerExpr_ZeroAmplitude(t *testing.T) {
	layer := audio.LayerParams{
		Type:      audio.LayerPrimaryScream,
		BaseFreq:  500,
		FreqRange: 1500,
		JumpRate:  10,
		Amplitude: 0,
		Rise:      1.2,
		Seed:      4242,
	}
	noise := audio.NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000}

	expr := layerExpr(layer, noise, 42, 0)

	// Zero amplitude should return "0" or empty string to indicate silence
	if expr != "0" && expr != "" {
		// If it produces something else, it should at least be valid and multiply by 0
		if !strings.Contains(expr, "0") {
			t.Errorf("layerExpr() with zero amplitude should produce '0', empty, or contain zero multiplier, got: %s", expr)
		}
	}
}

// --- fmtFloat tests ---

func Test_fmtFloat_Cases(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{"integer_value", 42.0},
		{"fractional_value", 3.14159},
		{"small_value", 0.001},
		{"negative_value", -5.5},
		{"zero", 0.0},
		{"large_value", 12000.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fmtFloat(tt.input)
			if result == "" {
				t.Errorf("fmtFloat(%v) returned empty string", tt.input)
			}
		})
	}
}

func Test_fmtFloat_ConsistentPrecision(t *testing.T) {
	// All outputs should have the same formatting approach (6 decimal places)
	result := fmtFloat(1.0)

	// Check that it produces a consistent format - should contain a decimal point
	if !strings.Contains(result, ".") && !strings.Contains(result, "e") {
		t.Errorf("fmtFloat(1.0) should contain decimal point for consistent precision, got: %s", result)
	}

	// Should have 6 decimal places
	parts := strings.SplitN(result, ".", 2)
	if len(parts) == 2 {
		decimals := parts[1]
		if len(decimals) != 6 {
			t.Errorf("fmtFloat(1.0) should have 6 decimal places, got %d in result: %s", len(decimals), result)
		}
	}
}

func Test_fmtFloat_NegativeValue(t *testing.T) {
	result := fmtFloat(-5.5)
	if !strings.HasPrefix(result, "-") {
		t.Errorf("fmtFloat(-5.5) should start with '-', got: %s", result)
	}
}

// --- deriveSeed tests ---

func Test_deriveSeed_DifferentIndexes(t *testing.T) {
	seeds := make(map[int64]bool)
	for i := 0; i < 5; i++ {
		seed := deriveSeed(42, 100, i)
		if seeds[seed] {
			t.Errorf("deriveSeed(42, 100, %d) produced duplicate seed %d", i, seed)
		}
		seeds[seed] = true
	}
}

func Test_deriveSeed_DifferentGlobalSeeds(t *testing.T) {
	s1 := deriveSeed(1, 100, 0)
	s2 := deriveSeed(2, 100, 0)

	if s1 == s2 {
		t.Errorf("deriveSeed with different global seeds produced same result: %d", s1)
	}
}

func Test_deriveSeed_Deterministic(t *testing.T) {
	s1 := deriveSeed(42, 100, 3)
	s2 := deriveSeed(42, 100, 3)

	if s1 != s2 {
		t.Errorf("deriveSeed() not deterministic: %d != %d", s1, s2)
	}
}

func Test_deriveSeed_NonNegative(t *testing.T) {
	// Test with a variety of inputs including negative seeds
	testCases := []struct {
		name       string
		globalSeed int64
		layerSeed  int64
		index      int
	}{
		{"positive_seeds", 42, 100, 0},
		{"zero_global", 0, 100, 0},
		{"zero_layer", 42, 0, 0},
		{"large_seeds", 999999, 888888, 4},
		{"negative_global", -1, 100, 0},
		{"negative_layer", 42, -1, 0},
		{"both_negative", -100, -200, 3},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveSeed(tt.globalSeed, tt.layerSeed, tt.index)
			if result < 0 {
				t.Errorf("deriveSeed(%d, %d, %d) = %d, want non-negative",
					tt.globalSeed, tt.layerSeed, tt.index, result)
			}
		})
	}
}

func Test_deriveSeed_DifferentLayerSeeds(t *testing.T) {
	s1 := deriveSeed(42, 100, 0)
	s2 := deriveSeed(42, 200, 0)

	if s1 == s2 {
		t.Errorf("deriveSeed with different layer seeds produced same result: %d", s1)
	}
}

// --- BuildArgs comprehensive table test ---

func Test_BuildArgs_AllPresets(t *testing.T) {
	presets := audio.AllPresets()
	for _, name := range presets {
		t.Run(string(name), func(t *testing.T) {
			params, ok := audio.GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false", name)
			}
			args := buildArgs(params)
			if len(args) == 0 {
				t.Errorf("buildArgs() for preset %q returned empty args", name)
			}

			joined := strings.Join(args, " ")
			requiredFragments := []string{
				"-f lavfi",
				"aevalsrc=",
				"-af",
				"-f s16le",
				"-acodec pcm_s16le",
				"pipe:1",
			}
			for _, frag := range requiredFragments {
				if !strings.Contains(joined, frag) {
					t.Errorf("buildArgs() for preset %q missing '%s' in: %s", name, frag, joined)
				}
			}
		})
	}
}

func Test_BuildArgs_WithRandomizedParams(t *testing.T) {
	seeds := []int64{1, 42, 100, 9999, 12345}
	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			params := audio.Randomize(seed)
			args := buildArgs(params)
			if len(args) == 0 {
				t.Errorf("buildArgs() for Randomize(%d) returned empty args", seed)
			}
			if args[len(args)-1] != "pipe:1" {
				t.Errorf("buildArgs() last arg = %q, want 'pipe:1'", args[len(args)-1])
			}
		})
	}
}

// --- Benchmarks ---

func BenchmarkBuildArgs(b *testing.B) {
	params := classicParams()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildArgs(params)
	}
}

func BenchmarkBuildAevalsrcExpr(b *testing.B) {
	params := classicParams()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildAevalsrcExpr(params)
	}
}

func BenchmarkBuildFilterChain(b *testing.B) {
	filter := classicParams().Filter
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildFilterChain(filter)
	}
}
