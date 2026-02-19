package ffmpeg

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// BuildArgs builds the complete FFmpeg CLI argument list from ScreamParams.
// The output is raw s16le PCM written to stdout (pipe:1).
func BuildArgs(params audio.ScreamParams) []string {
	sampleRate := strconv.Itoa(params.SampleRate)
	channels := strconv.Itoa(params.Channels)
	durationSec := params.Duration.Seconds()
	durationStr := strconv.FormatFloat(durationSec, 'f', -1, 64)

	aevalsrcExpr := buildAevalsrcExpr(params)
	aevalsrcArg := fmt.Sprintf("aevalsrc='%s':s=%s:d=%s", aevalsrcExpr, sampleRate, durationStr)

	filterChain := buildFilterChain(params.Filter)

	return []string{
		"-nostdin",
		"-v", "quiet",
		"-f", "lavfi",
		"-i", aevalsrcArg,
		"-af", filterChain,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ac", channels,
		"-ar", sampleRate,
		"pipe:1",
	}
}

// buildAevalsrcExpr builds the aevalsrc expression by summing all active layers.
func buildAevalsrcExpr(params audio.ScreamParams) string {
	parts := make([]string, 0, len(params.Layers))
	for i, layer := range params.Layers {
		expr := layerExpr(layer, params.Noise, params.Seed, i)
		parts = append(parts, expr)
	}
	return strings.Join(parts, "+")
}

// layerExpr builds the FFmpeg aevalsrc expression for a single synthesis layer.
// Zero-amplitude layers return "0".
func layerExpr(layer audio.LayerParams, noise audio.NoiseParams, globalSeed int64, index int) string {
	seed := deriveSeed(globalSeed, layer.Seed, index)
	seedStr := strconv.FormatInt(seed, 10)
	sampleRate := "48000" // aevalsrc uses its own sample rate; use a constant for the random seeding

	switch layer.Type {
	case audio.LayerPrimaryScream:
		if layer.Amplitude == 0 {
			return "0"
		}
		amp := fmtFloat(layer.Amplitude)
		rise := fmtFloat(layer.Rise)
		baseFreq := fmtFloat(layer.BaseFreq)
		freqRange := fmtFloat(layer.FreqRange)
		jumpRate := fmtFloat(layer.JumpRate)
		return fmt.Sprintf(
			"%s*(1+%s*t)*sin(2*PI*t*(%s+%s*random(floor(t*%s)*137+%s)))",
			amp, rise, baseFreq, freqRange, jumpRate, seedStr,
		)

	case audio.LayerHarmonicSweep:
		if layer.Amplitude == 0 {
			return "0"
		}
		amp := fmtFloat(layer.Amplitude)
		baseFreq := fmtFloat(layer.BaseFreq)
		sweepRate := fmtFloat(layer.SweepRate)
		freqRange := fmtFloat(layer.FreqRange)
		jumpRate := fmtFloat(layer.JumpRate)
		return fmt.Sprintf(
			"%s*sin(2*PI*t*(%s+%s*t+%s*random(floor(t*%s)*251+%s)))",
			amp, baseFreq, sweepRate, freqRange, jumpRate, seedStr,
		)

	case audio.LayerHighShriek:
		if layer.Amplitude == 0 {
			return "0"
		}
		amp := fmtFloat(layer.Amplitude)
		rise := fmtFloat(layer.Rise)
		baseFreq := fmtFloat(layer.BaseFreq)
		freqRange := fmtFloat(layer.FreqRange)
		jumpRate := fmtFloat(layer.JumpRate)
		return fmt.Sprintf(
			"%s*(1+%s*t)*sin(2*PI*t*(%s+%s*random(floor(t*%s)*89+%s)))",
			amp, rise, baseFreq, freqRange, jumpRate, seedStr,
		)

	case audio.LayerNoiseBurst:
		burstAmp := noise.BurstAmp
		if burstAmp == 0 {
			return "0"
		}
		burstSeed := deriveSeed(globalSeed, noise.BurstSeed, index)
		burstSeedStr := strconv.FormatInt(burstSeed, 10)
		burstAmpStr := fmtFloat(burstAmp)
		burstRateStr := fmtFloat(noise.BurstRate)
		thresholdStr := fmtFloat(noise.Threshold)
		return fmt.Sprintf(
			"%s*gt(random(floor(t*%s)*173+%s),%s)*(2*random(t*%s)-1)",
			burstAmpStr, burstRateStr, burstSeedStr, thresholdStr, sampleRate,
		)

	case audio.LayerBackgroundNoise:
		floorAmp := noise.FloorAmp
		if floorAmp == 0 {
			return "0"
		}
		floorAmpStr := fmtFloat(floorAmp)
		return fmt.Sprintf(
			"%s*(2*random(t*%s+7777)-1)",
			floorAmpStr, sampleRate,
		)

	default:
		return "0"
	}
}

// buildFilterChain builds the FFmpeg -af filter chain string from FilterParams.
// Filters are applied in order: highpass, lowpass, acrusher, acompressor, volume, alimiter.
func buildFilterChain(filter audio.FilterParams) string {
	highpass := fmt.Sprintf("highpass=f=%s", fmtFloat(filter.HighpassCutoff))
	lowpass := fmt.Sprintf("lowpass=f=%s", fmtFloat(filter.LowpassCutoff))
	acrusher := fmt.Sprintf("acrusher=bits=%d:mix=%s:mode=log:aa=1",
		filter.CrusherBits,
		fmtFloat(filter.CrusherMix),
	)
	acompressor := fmt.Sprintf("acompressor=ratio=%s:attack=%s:release=%s:threshold=%sdB",
		fmtFloat(filter.CompRatio),
		fmtFloat(filter.CompAttack),
		fmtFloat(filter.CompRelease),
		fmtFloat(filter.CompThreshold),
	)
	volume := fmt.Sprintf("volume=%sdB", fmtFloat(filter.VolumeBoostDB))
	alimiter := fmt.Sprintf("alimiter=limit=%s:attack=1:release=10", fmtFloat(filter.LimiterLevel))

	return strings.Join([]string{highpass, lowpass, acrusher, acompressor, volume, alimiter}, ",")
}

// fmtFloat formats a float64 with 6 decimal places for use in FFmpeg expressions.
func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}

// deriveSeed derives a deterministic non-negative per-layer seed from the global seed,
// per-layer seed, and layer index.
func deriveSeed(globalSeed, layerSeed int64, index int) int64 {
	result := globalSeed*1000003 ^ layerSeed ^ (int64(index) * 7919)
	if result == math.MinInt64 {
		return 0
	}
	if result < 0 {
		return -result
	}
	return result
}
