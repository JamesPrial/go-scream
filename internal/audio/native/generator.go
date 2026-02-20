// Package native provides pure Go audio synthesis and processing.
package native

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// Compile-time check that Generator implements audio.Generator.
var _ audio.Generator = (*Generator)(nil)

// Prime multipliers used to decorrelate per-layer seeds from the global seed.
// Each constant is a distinct prime, ensuring the XOR mixes are independent.
const (
	seedMixLayer0 int64 = 1000003
	seedMixLayer1 int64 = 1000033
	seedMixLayer2 int64 = 1000037
	seedMixLayer3 int64 = 1000039
	seedMixNoise  int64 = 1000081
)

// Generator implements audio.Generator using pure Go synthesis.
// It produces s16le PCM audio with a configurable sample rate and channel count.
type Generator struct {
	logger *slog.Logger
}

// NewGenerator creates a new Generator using the provided logger.
func NewGenerator(logger *slog.Logger) *Generator {
	return &Generator{logger: logger}
}

// Generate produces PCM audio data in s16le format (little-endian signed 16-bit).
// The output byte count is: totalSamples * channels * 2, where
// totalSamples = int(duration.Seconds() * float64(sampleRate)).
// Returns an error if params fail validation.
func (g *Generator) Generate(params audio.ScreamParams) (io.Reader, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	g.logger.Debug("generating PCM audio", "duration", params.Duration, "sample_rate", params.SampleRate, "channels", params.Channels)

	sampleRate := params.SampleRate
	totalSamples := int(params.Duration.Seconds() * float64(sampleRate))
	channels := params.Channels

	// Build the 5 synthesis layers from params.
	layers := buildLayers(params, sampleRate)

	// Create the mixer from all layers.
	mixer := newLayerMixer(layers...)

	// Create the filter chain from params.
	chain := newFilterChainFromParams(params.Filter, sampleRate)

	// Allocate output buffer: totalSamples * channels * 2 bytes per sample.
	out := make([]byte, totalSamples*channels*2)
	pos := 0

	for i := 0; i < totalSamples; i++ {
		t := float64(i) / float64(sampleRate)

		// Mix all layers at time t.
		raw := mixer.Sample(t)

		// Apply the filter chain.
		filtered := chain.Process(raw)

		// Convert to int16 by scaling and clamping.
		scaled := filtered * 32767.0
		clamped := math.Max(-32768, math.Min(32767, scaled))
		s16 := int16(math.Round(clamped))

		// Encode the sample as little-endian int16 for each channel.
		lo := byte(uint16(s16))
		hi := byte(uint16(s16) >> 8)
		for range channels {
			out[pos] = lo
			out[pos+1] = hi
			pos += 2
		}
	}

	g.logger.Debug("PCM generation complete", "bytes", len(out))
	return bytes.NewReader(out), nil
}

// buildLayers creates all 5 synthesis layers from ScreamParams.
// The global params.Seed is mixed into each layer's seed so that different
// top-level seeds produce different audio even when LayerParams seeds are identical.
func buildLayers(params audio.ScreamParams, sampleRate int) []layer {
	lp := params.Layers
	noise := params.Noise
	globalSeed := params.Seed

	// Derive per-layer seeds that incorporate the global seed.
	// Using XOR with prime multiples of globalSeed ensures decorrelation.
	p0 := lp[0]
	p0.Seed = lp[0].Seed ^ (globalSeed * seedMixLayer0)

	p1 := lp[1]
	p1.Seed = lp[1].Seed ^ (globalSeed * seedMixLayer1)

	p2 := lp[2]
	p2.Seed = lp[2].Seed ^ (globalSeed * seedMixLayer2)

	p3 := lp[3]
	p3.Seed = lp[3].Seed ^ (globalSeed * seedMixLayer3)

	noiseWithSeed := noise
	noiseWithSeed.BurstSeed = noise.BurstSeed ^ (globalSeed * seedMixNoise)

	layers := []layer{
		newPrimaryScreamLayer(p0, sampleRate),
		newHarmonicSweepLayer(p1, sampleRate),
		newHighShriekLayer(p2, sampleRate),
		newNoiseBurstLayer(p3, noiseWithSeed),
		newBackgroundNoiseLayer(noiseWithSeed),
	}

	return layers
}
