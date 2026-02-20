package native

import (
	"math/rand"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// backgroundNoiseSeedXOR is XORed into the burst seed to produce a distinct
// seed for the background noise RNG, decorrelating it from the burst layer.
const backgroundNoiseSeedXOR int64 = 0x5a5a5a5a5a5a5a5a

// layer generates audio samples for a single synthesis layer.
type layer interface {
	Sample(t float64) float64
}

// sweepJumpLayer generates a scream tone with frequency jumps, parameterised
// by a coprime constant for deterministic stepping. It is used for both the
// primary scream and high-shriek synthesis layers.
type sweepJumpLayer struct {
	osc       *oscillator
	seed      int64
	base      float64
	freqRange float64
	jump      float64
	amp       float64
	rise      float64
	coprime   int64
	curStep   int64
	curFreq   float64
}

// newSweepJumpLayer constructs a sweepJumpLayer with the given params, sample
// rate, and coprime constant. It is the shared implementation for both the
// primary scream and high-shriek layer constructors.
func newSweepJumpLayer(p audio.LayerParams, sampleRate int, coprime int64) *sweepJumpLayer {
	return &sweepJumpLayer{
		osc:       newOscillator(sampleRate),
		seed:      p.Seed,
		base:      p.BaseFreq,
		freqRange: p.FreqRange,
		jump:      p.JumpRate,
		amp:       p.Amplitude,
		rise:      p.Rise,
		coprime:   coprime,
		curStep:   -1,
	}
}

// newPrimaryScreamLayer creates a primary scream layer from params.
func newPrimaryScreamLayer(p audio.LayerParams, sampleRate int) *sweepJumpLayer {
	return newSweepJumpLayer(p, sampleRate, audio.CoprimePrimaryScream)
}

// Sample returns the audio sample at time t for the sweep-jump layer.
// The frequency jumps at discrete steps determined by the layer seed and coprime.
func (l *sweepJumpLayer) Sample(t float64) float64 {
	step := int64(t * l.jump)
	if step != l.curStep {
		l.curStep = step
		l.curFreq = l.base + l.freqRange*seededRandom(l.seed, step, l.coprime)
	}
	envelope := l.amp * (1 + l.rise*t)
	return envelope * l.osc.sin(l.curFreq)
}

// harmonicSweepLayer generates a harmonic tone with linear frequency sweep plus jumps.
type harmonicSweepLayer struct {
	osc       *oscillator
	seed      int64
	base      float64
	sweep     float64
	freqRange float64
	jump      float64
	amp       float64
	curStep   int64
	curFreq   float64
}

// newHarmonicSweepLayer creates a harmonic sweep layer from params.
func newHarmonicSweepLayer(p audio.LayerParams, sampleRate int) *harmonicSweepLayer {
	return &harmonicSweepLayer{
		osc:       newOscillator(sampleRate),
		seed:      p.Seed,
		base:      p.BaseFreq,
		sweep:     p.SweepRate,
		freqRange: p.FreqRange,
		jump:      p.JumpRate,
		amp:       p.Amplitude,
		curStep:   -1,
	}
}

// Sample returns the audio sample at time t for the harmonic sweep layer.
// The frequency sweeps linearly over time with discrete jumps.
func (l *harmonicSweepLayer) Sample(t float64) float64 {
	step := int64(t * l.jump)
	if step != l.curStep {
		l.curStep = step
		l.curFreq = l.freqRange * seededRandom(l.seed, step, audio.CoprimeHarmonicSweep)
	}
	freq := l.base + l.sweep*t + l.curFreq
	return l.amp * l.osc.sin(freq)
}

// newHighShriekLayer creates a high shriek layer from params.
// It returns a *sweepJumpLayer configured with CoprimeHighShriek.
func newHighShriekLayer(p audio.LayerParams, sampleRate int) *sweepJumpLayer {
	return newSweepJumpLayer(p, sampleRate, audio.CoprimeHighShriek)
}

// noiseBurstLayer generates gated noise bursts.
type noiseBurstLayer struct {
	burstSeed int64
	noiseRng  *rand.Rand
	burstRate float64
	threshold float64
	amp       float64
	curStep   int64
	curGate   float64
}

// newNoiseBurstLayer creates a noise burst layer from params.
func newNoiseBurstLayer(p audio.LayerParams, noise audio.NoiseParams) *noiseBurstLayer {
	return &noiseBurstLayer{
		burstSeed: noise.BurstSeed,
		noiseRng:  rand.New(rand.NewSource(noise.BurstSeed)),
		burstRate: noise.BurstRate,
		threshold: noise.Threshold,
		amp:       noise.BurstAmp,
		curStep:   -1,
	}
}

// Sample returns the audio sample at time t for the noise burst layer.
// The gate opens at discrete burst steps; when open, white noise is output.
func (l *noiseBurstLayer) Sample(t float64) float64 {
	step := int64(t * l.burstRate)
	if step != l.curStep {
		l.curStep = step
		l.curGate = seededRandom(l.burstSeed, step, audio.CoprimeNoiseBurst)
	}
	if l.curGate <= l.threshold {
		return 0
	}
	// White noise when gate is open; use stateful RNG for continuous noise.
	noise := 2*l.noiseRng.Float64() - 1
	return l.amp * noise
}

// backgroundNoiseLayer generates constant low-level background noise.
type backgroundNoiseLayer struct {
	noiseRng *rand.Rand
	amp      float64
}

// newBackgroundNoiseLayer creates a background noise layer from params.
func newBackgroundNoiseLayer(noise audio.NoiseParams) *backgroundNoiseLayer {
	return &backgroundNoiseLayer{
		noiseRng: rand.New(rand.NewSource(noise.BurstSeed ^ backgroundNoiseSeedXOR)),
		amp:      noise.FloorAmp,
	}
}

// Sample returns the audio sample at time t for the background noise layer.
// Produces continuous low-level white noise at the configured amplitude.
func (l *backgroundNoiseLayer) Sample(_ float64) float64 {
	noise := 2*l.noiseRng.Float64() - 1
	return l.amp * noise
}

// layerMixer mixes multiple layers together, clamping to [-1, 1].
type layerMixer struct {
	layers []layer
}

// newLayerMixer creates a mixer with the given layers.
func newLayerMixer(layers ...layer) *layerMixer {
	return &layerMixer{layers: layers}
}

// Sample returns the sum of all layer samples at time t, clamped to [-1, 1].
func (m *layerMixer) Sample(t float64) float64 {
	var sum float64
	for _, l := range m.layers {
		sum += l.Sample(t)
	}
	return clamp(sum, -1, 1)
}

// splitmix64 is a stateless bijective hash function used for deterministic
// pseudo-random number generation. It returns a float64 in [0, 1).
func splitmix64(seed int64) float64 {
	s := uint64(seed)
	s = (s ^ (s >> 30)) * 0xbf58476d1ce4e5b9
	s = (s ^ (s >> 27)) * 0x94d049bb133111eb
	s = s ^ (s >> 31)
	return float64(s>>11) / float64(1<<53)
}

// seededRandom returns a deterministic value in [0, 1) for the given
// (layerSeed, step, coprime) triple. The result is the same for any given
// input regardless of call order — it does not advance any shared RNG state.
func seededRandom(layerSeed, step, coprime int64) float64 {
	h := layerSeed ^ (step * coprime)
	return splitmix64(h)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
