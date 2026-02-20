// Package native provides pure Go audio synthesis and processing.
package native

import (
	"math"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// filter processes individual audio samples.
type filter interface {
	Process(sample float64) float64
}

// highpassFilter implements a first-order IIR high-pass filter.
// It removes low-frequency content (including DC offset) from the signal.
type highpassFilter struct {
	alpha   float64
	prevIn  float64
	prevOut float64
}

// newHighpassFilter creates a high-pass filter with the given cutoff frequency and sample rate.
// alpha = RC / (RC + dt) where RC = 1/(2*pi*cutoff) and dt = 1/sampleRate.
func newHighpassFilter(cutoff float64, sampleRate int) *highpassFilter {
	dt := 1.0 / float64(sampleRate)
	rc := 1.0 / (2 * math.Pi * cutoff)
	alpha := rc / (rc + dt)
	return &highpassFilter{alpha: alpha}
}

// Process applies the high-pass filter to a single sample.
func (f *highpassFilter) Process(sample float64) float64 {
	out := f.alpha * (f.prevOut + sample - f.prevIn)
	f.prevIn = sample
	f.prevOut = out
	return out
}

// lowpassFilter implements a first-order IIR low-pass filter.
// It removes high-frequency content from the signal.
type lowpassFilter struct {
	alpha float64
	prev  float64
}

// newLowpassFilter creates a low-pass filter with the given cutoff frequency and sample rate.
// alpha = dt / (RC + dt) where RC = 1/(2*pi*cutoff) and dt = 1/sampleRate.
func newLowpassFilter(cutoff float64, sampleRate int) *lowpassFilter {
	dt := 1.0 / float64(sampleRate)
	rc := 1.0 / (2 * math.Pi * cutoff)
	alpha := dt / (rc + dt)
	return &lowpassFilter{alpha: alpha}
}

// Process applies the low-pass filter to a single sample.
func (f *lowpassFilter) Process(sample float64) float64 {
	f.prev = f.prev + f.alpha*(sample-f.prev)
	return f.prev
}

// bitcrusher reduces the bit depth of a signal, creating a lo-fi effect.
// It blends the quantized signal with the original clean signal.
type bitcrusher struct {
	levels float64
	mix    float64
}

// newBitcrusher creates a bitcrusher with the given bit depth and wet/dry mix.
// bits controls the quantization resolution (levels = 2^bits).
// mix controls the blend: 1.0 is fully crushed, 0.0 is fully clean.
func newBitcrusher(bits int, mix float64) *bitcrusher {
	return &bitcrusher{
		levels: math.Pow(2, float64(bits)),
		mix:    mix,
	}
}

// Process applies the bitcrusher to a single sample.
// Quantization: crushed = floor(sample * levels) / levels.
// Output: mix*crushed + (1-mix)*clean.
func (f *bitcrusher) Process(sample float64) float64 {
	crushed := math.Floor(sample*f.levels) / f.levels
	return f.mix*crushed + (1-f.mix)*sample
}

// compressor implements dynamic range compression.
// It tracks the signal envelope and reduces gain when the signal exceeds the threshold.
type compressor struct {
	ratio       float64
	threshold   float64 // linear amplitude threshold
	attackCoef  float64
	releaseCoef float64
	envelope    float64
	ratioExp    float64
}

// newCompressor creates a compressor with the given parameters.
// ratio is the compression ratio (e.g., 8 means 8:1).
// thresholdDB is the threshold in dBFS (e.g., -20).
// attackMs and releaseMs control how fast the envelope responds in milliseconds.
// sampleRate is the audio sample rate in Hz.
func newCompressor(ratio, thresholdDB, attackMs, releaseMs float64, sampleRate int) *compressor {
	// Convert dB threshold to linear amplitude
	threshold := math.Pow(10, thresholdDB/20.0)
	// Time constants: coefficient = exp(-1 / (time_in_samples))
	attackSamples := (attackMs / 1000.0) * float64(sampleRate)
	releaseSamples := (releaseMs / 1000.0) * float64(sampleRate)
	attackCoef := math.Exp(-1.0 / attackSamples)
	releaseCoef := math.Exp(-1.0 / releaseSamples)
	return &compressor{
		ratio:       ratio,
		threshold:   threshold,
		attackCoef:  attackCoef,
		releaseCoef: releaseCoef,
		ratioExp:    1.0/ratio - 1.0,
	}
}

// Process applies the compressor to a single sample.
func (f *compressor) Process(sample float64) float64 {
	absIn := math.Abs(sample)

	// Envelope tracking with attack/release
	if absIn > f.envelope {
		f.envelope = f.attackCoef*f.envelope + (1-f.attackCoef)*absIn
	} else {
		f.envelope = f.releaseCoef*f.envelope + (1-f.releaseCoef)*absIn
	}

	// Compute gain reduction
	gain := 1.0
	if f.envelope > f.threshold {
		// How much above threshold (in linear)
		excess := f.envelope / f.threshold
		// Apply ratio: reduce the excess by the compression ratio
		gain = math.Exp(f.ratioExp * math.Log(excess))
	}

	return sample * gain
}

// volumeBoost applies a fixed linear gain derived from a dB value.
type volumeBoost struct {
	gain float64
}

// newVolumeBoost creates a volume boost filter with gain specified in dB.
// gain = 10^(dB/20). Negative dB values produce attenuation.
func newVolumeBoost(dB float64) *volumeBoost {
	return &volumeBoost{gain: math.Pow(10, dB/20.0)}
}

// Process applies the volume boost to a single sample.
func (f *volumeBoost) Process(sample float64) float64 {
	return sample * f.gain
}

// limiter implements a hard clipper that limits the signal to ±level.
type limiter struct {
	level float64
}

// newLimiter creates a hard limiter that clips samples to ±level.
func newLimiter(level float64) *limiter {
	return &limiter{level: level}
}

// Process clips the sample to ±level.
func (f *limiter) Process(sample float64) float64 {
	return clamp(sample, -f.level, f.level)
}

// filterChain applies multiple filters in sequence.
type filterChain struct {
	filters []filter
}

// newFilterChain creates a filter chain from the given filters.
// Filters are applied in the order they are provided.
func newFilterChain(filters ...filter) *filterChain {
	return &filterChain{filters: filters}
}

// Process applies all filters in the chain to a single sample.
func (f *filterChain) Process(sample float64) float64 {
	out := sample
	for _, flt := range f.filters {
		out = flt.Process(out)
	}
	return out
}

// newFilterChainFromParams builds the standard processing chain from FilterParams.
// The chain order is: highpass -> lowpass -> bitcrusher -> compressor -> volumeBoost -> limiter.
func newFilterChainFromParams(fp audio.FilterParams, sampleRate int) *filterChain {
	return newFilterChain(
		newHighpassFilter(fp.HighpassCutoff, sampleRate),
		newLowpassFilter(fp.LowpassCutoff, sampleRate),
		newBitcrusher(fp.CrusherBits, fp.CrusherMix),
		newCompressor(fp.CompRatio, fp.CompThreshold, fp.CompAttack, fp.CompRelease, sampleRate),
		newVolumeBoost(fp.VolumeBoostDB),
		newLimiter(fp.LimiterLevel),
	)
}
