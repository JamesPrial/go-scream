package audio

import (
	"math/rand"
	"time"
)

// ScreamParams holds all parameters for generating a scream.
type ScreamParams struct {
	Duration   time.Duration
	SampleRate int
	Channels   int
	Seed       int64
	Layers     [5]LayerParams
	Noise      NoiseParams
	Filter     FilterParams
}

// LayerType identifies the synthesis method for a layer.
type LayerType int

const (
	LayerPrimaryScream LayerType = iota
	LayerHarmonicSweep
	LayerHighShriek
	LayerNoiseBurst
	LayerBackgroundNoise
)

// Coprime constants used by both native and ffmpeg backends for
// deterministic frequency stepping in audio layer generation.
const (
	CoprimePrimaryScream int64 = 137
	CoprimeHarmonicSweep int64 = 251
	CoprimeHighShriek    int64 = 89
	CoprimeNoiseBurst    int64 = 173
)

// LayerParams holds parameters for a single synthesis layer.
type LayerParams struct {
	Type      LayerType
	BaseFreq  float64 // Base frequency in Hz
	FreqRange float64 // Frequency jump range in Hz
	SweepRate float64 // Linear frequency sweep rate (Hz/s), used by harmonic sweep
	JumpRate  float64 // How often frequency jumps (Hz)
	Amplitude float64 // Layer amplitude [0, 1]
	Rise      float64 // Exponential amplitude rise over time
	Seed      int64   // RNG seed for this layer
}

// NoiseParams holds parameters for the noise layers.
type NoiseParams struct {
	BurstRate float64 // Burst frequency for gated noise (Hz)
	Threshold float64 // Gate threshold [0, 1]
	BurstAmp  float64 // Amplitude for noise bursts
	FloorAmp  float64 // Amplitude for background noise floor
	BurstSeed int64   // RNG seed for burst gating
}

// FilterParams holds post-processing filter parameters.
type FilterParams struct {
	HighpassCutoff float64 // High-pass filter cutoff (Hz)
	LowpassCutoff  float64 // Low-pass filter cutoff (Hz)
	CrusherBits    int     // Bit depth for bitcrusher (6-12)
	CrusherMix     float64 // Mix of crushed vs clean signal [0, 1]
	CompRatio      float64 // Compressor ratio
	CompThreshold  float64 // Compressor threshold in dB
	CompAttack     float64 // Compressor attack in ms
	CompRelease    float64 // Compressor release in ms
	VolumeBoostDB  float64 // Volume boost in dB
	LimiterLevel   float64 // Hard limiter level [0, 1]
}

// Randomize fills ScreamParams with random values matching the original bot's ranges.
// If seed is 0, a random seed is chosen.
func Randomize(seed int64) ScreamParams {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(seed))
	rf := func(min, max float64) float64 { return min + r.Float64()*(max-min) }
	ri := func(min, max int64) int64 { return min + r.Int63n(max-min+1) }

	dur := rf(2.5, 4.0)

	return ScreamParams{
		Duration:   time.Duration(dur * float64(time.Second)),
		SampleRate: 48000,
		Channels:   2,
		Seed:       seed,
		Layers: [5]LayerParams{
			{
				Type:      LayerPrimaryScream,
				BaseFreq:  rf(300, 700),
				FreqRange: rf(800, 2500),
				JumpRate:  rf(5, 15),
				Amplitude: rf(0.3, 0.5),
				Rise:      rf(0.5, 2.0),
				Seed:      ri(1, 9999),
			},
			{
				Type:      LayerHarmonicSweep,
				BaseFreq:  rf(200, 500),
				SweepRate: rf(300, 900),
				FreqRange: rf(400, 1200),
				JumpRate:  rf(3, 10),
				Amplitude: rf(0.15, 0.3),
				Seed:      ri(1, 9999),
			},
			{
				Type:      LayerHighShriek,
				BaseFreq:  rf(900, 1800),
				FreqRange: rf(800, 2400),
				JumpRate:  rf(10, 25),
				Amplitude: rf(0.15, 0.3),
				Rise:      rf(1.0, 3.0),
				Seed:      ri(1, 9999),
			},
			{
				Type:      LayerNoiseBurst,
				Amplitude: rf(0.1, 0.25),
				Seed:      ri(1, 9999),
			},
			{
				Type:      LayerBackgroundNoise,
				Amplitude: rf(0.05, 0.15),
			},
		},
		Noise: NoiseParams{
			BurstRate: rf(3, 12),
			Threshold: rf(0.5, 0.85),
			BurstAmp:  rf(0.1, 0.25),
			FloorAmp:  rf(0.05, 0.15),
			BurstSeed: ri(1, 9999),
		},
		Filter: FilterParams{
			HighpassCutoff: rf(80, 200),
			LowpassCutoff:  rf(6000, 12000),
			CrusherBits:    int(ri(6, 12)),
			CrusherMix:     rf(0.3, 0.7),
			CompRatio:      rf(4, 12),
			CompThreshold:  -20,
			CompAttack:     5,
			CompRelease:    50,
			VolumeBoostDB:  rf(6, 12),
			LimiterLevel:   0.95,
		},
	}
}

// Validate checks that all parameters are within valid ranges.
func (p ScreamParams) Validate() error {
	if p.Duration <= 0 {
		return ErrInvalidDuration
	}
	if p.SampleRate <= 0 {
		return ErrInvalidSampleRate
	}
	if p.Channels != 1 && p.Channels != 2 {
		return ErrInvalidChannels
	}
	for i, l := range p.Layers {
		if l.Amplitude < 0 || l.Amplitude > 1 {
			return &LayerValidationError{Layer: i, Err: ErrInvalidAmplitude}
		}
	}
	if p.Filter.HighpassCutoff < 0 {
		return ErrInvalidFilterCutoff
	}
	if p.Filter.LowpassCutoff < 0 {
		return ErrInvalidFilterCutoff
	}
	if p.Filter.CrusherBits < 1 || p.Filter.CrusherBits > 16 {
		return ErrInvalidCrusherBits
	}
	if p.Filter.LimiterLevel <= 0 || p.Filter.LimiterLevel > 1 {
		return ErrInvalidLimiterLevel
	}
	return nil
}
