package audio

import "time"

// PresetName identifies a named scream preset.
type PresetName string

const (
	PresetClassic    PresetName = "classic"
	PresetWhisper    PresetName = "whisper"
	PresetDeathMetal PresetName = "death-metal"
	PresetGlitch     PresetName = "glitch"
	PresetBanshee    PresetName = "banshee"
	PresetRobot      PresetName = "robot"
)

// AllPresets returns a list of all available preset names.
func AllPresets() []PresetName {
	return []PresetName{
		PresetClassic,
		PresetWhisper,
		PresetDeathMetal,
		PresetGlitch,
		PresetBanshee,
		PresetRobot,
	}
}

// GetPreset returns the ScreamParams for a named preset.
func GetPreset(name PresetName) (ScreamParams, bool) {
	p, ok := presets[name]
	return p, ok
}

func classicFilter() FilterParams {
	filter := DefaultFilterParams()
	filter.HighpassCutoff = 120
	filter.LowpassCutoff = 8000
	filter.CrusherBits = 8
	filter.CrusherMix = 0.5
	filter.CompRatio = 8
	filter.VolumeBoostDB = 9
	return filter
}

func whisperFilter() FilterParams {
	filter := DefaultFilterParams()
	filter.HighpassCutoff = 200
	filter.LowpassCutoff = 6000
	filter.CrusherBits = 12
	filter.CrusherMix = 0.2
	filter.CompRatio = 4
	filter.VolumeBoostDB = 6
	return filter
}

func deathMetalFilter() FilterParams {
	filter := DefaultFilterParams()
	filter.HighpassCutoff = 80
	filter.LowpassCutoff = 12000
	filter.CrusherBits = 6
	filter.CrusherMix = 0.7
	filter.CompRatio = 12
	filter.VolumeBoostDB = 12
	return filter
}

func glitchFilter() FilterParams {
	filter := DefaultFilterParams()
	filter.HighpassCutoff = 100
	filter.LowpassCutoff = 10000
	filter.CrusherBits = 6
	filter.CrusherMix = 0.7
	filter.CompRatio = 6
	filter.VolumeBoostDB = 8
	return filter
}

func bansheeFilter() FilterParams {
	filter := DefaultFilterParams()
	filter.HighpassCutoff = 150
	filter.LowpassCutoff = 11000
	filter.CrusherBits = 10
	filter.CrusherMix = 0.4
	filter.CompRatio = 6
	filter.VolumeBoostDB = 10
	return filter
}

func robotFilter() FilterParams {
	filter := DefaultFilterParams()
	filter.HighpassCutoff = 100
	filter.LowpassCutoff = 7000
	filter.CrusherBits = 6
	filter.CrusherMix = 0.65
	filter.CompRatio = 10
	filter.VolumeBoostDB = 8
	return filter
}

var presets = map[PresetName]ScreamParams{
	PresetClassic: {
		Duration:   3 * time.Second,
		SampleRate: DefaultSampleRate,
		Channels:   DefaultChannels,
		Layers: [5]LayerParams{
			{Type: LayerPrimaryScream, BaseFreq: 500, FreqRange: 1500, JumpRate: 10, Amplitude: 0.4, Rise: 1.2, Seed: 4242},
			{Type: LayerHarmonicSweep, BaseFreq: 350, SweepRate: 500, FreqRange: 800, JumpRate: 6, Amplitude: 0.25, Seed: 3000},
			{Type: LayerHighShriek, BaseFreq: 1200, FreqRange: 1600, JumpRate: 20, Amplitude: 0.25, Rise: 2.5, Seed: 7000},
			{Type: LayerNoiseBurst, Amplitude: 0.18, Seed: 4000},
			{Type: LayerBackgroundNoise, Amplitude: 0.1},
		},
		Noise:  NoiseParams{BurstRate: 8, Threshold: 0.7, BurstAmp: 0.18, FloorAmp: 0.1, BurstSeed: 4000},
		Filter: classicFilter(),
	},
	PresetWhisper: {
		Duration:   2 * time.Second,
		SampleRate: DefaultSampleRate,
		Channels:   DefaultChannels,
		Layers: [5]LayerParams{
			{Type: LayerPrimaryScream, BaseFreq: 300, FreqRange: 500, JumpRate: 5, Amplitude: 0.15, Rise: 0.3, Seed: 1111},
			{Type: LayerHarmonicSweep, BaseFreq: 200, SweepRate: 150, FreqRange: 300, JumpRate: 3, Amplitude: 0.1, Seed: 2222},
			{Type: LayerHighShriek, BaseFreq: 900, FreqRange: 400, JumpRate: 8, Amplitude: 0.08, Rise: 0.5, Seed: 3333},
			{Type: LayerNoiseBurst, Amplitude: 0.05, Seed: 4444},
			{Type: LayerBackgroundNoise, Amplitude: 0.12},
		},
		Noise:  NoiseParams{BurstRate: 3, Threshold: 0.85, BurstAmp: 0.05, FloorAmp: 0.12, BurstSeed: 4444},
		Filter: whisperFilter(),
	},
	PresetDeathMetal: {
		Duration:   4 * time.Second,
		SampleRate: DefaultSampleRate,
		Channels:   DefaultChannels,
		Layers: [5]LayerParams{
			{Type: LayerPrimaryScream, BaseFreq: 150, FreqRange: 800, JumpRate: 15, Amplitude: 0.5, Rise: 2.0, Seed: 6660},
			{Type: LayerHarmonicSweep, BaseFreq: 100, SweepRate: 200, FreqRange: 600, JumpRate: 10, Amplitude: 0.3, Seed: 6661},
			{Type: LayerHighShriek, BaseFreq: 600, FreqRange: 2400, JumpRate: 25, Amplitude: 0.3, Rise: 3.0, Seed: 6662},
			{Type: LayerNoiseBurst, Amplitude: 0.25, Seed: 6663},
			{Type: LayerBackgroundNoise, Amplitude: 0.15},
		},
		Noise:  NoiseParams{BurstRate: 12, Threshold: 0.5, BurstAmp: 0.25, FloorAmp: 0.15, BurstSeed: 6663},
		Filter: deathMetalFilter(),
	},
	PresetGlitch: {
		Duration:   3 * time.Second,
		SampleRate: DefaultSampleRate,
		Channels:   DefaultChannels,
		Layers: [5]LayerParams{
			{Type: LayerPrimaryScream, BaseFreq: 700, FreqRange: 2500, JumpRate: 15, Amplitude: 0.35, Rise: 0.5, Seed: 1337},
			{Type: LayerHarmonicSweep, BaseFreq: 500, SweepRate: 900, FreqRange: 1200, JumpRate: 10, Amplitude: 0.2, Seed: 1338},
			{Type: LayerHighShriek, BaseFreq: 1800, FreqRange: 2400, JumpRate: 25, Amplitude: 0.2, Rise: 1.0, Seed: 1339},
			{Type: LayerNoiseBurst, Amplitude: 0.22, Seed: 1340},
			{Type: LayerBackgroundNoise, Amplitude: 0.05},
		},
		Noise:  NoiseParams{BurstRate: 12, Threshold: 0.5, BurstAmp: 0.22, FloorAmp: 0.05, BurstSeed: 1340},
		Filter: glitchFilter(),
	},
	PresetBanshee: {
		Duration:   4 * time.Second,
		SampleRate: DefaultSampleRate,
		Channels:   DefaultChannels,
		Layers: [5]LayerParams{
			{Type: LayerPrimaryScream, BaseFreq: 600, FreqRange: 2000, JumpRate: 8, Amplitude: 0.45, Rise: 2.0, Seed: 9001},
			{Type: LayerHarmonicSweep, BaseFreq: 400, SweepRate: 800, FreqRange: 1000, JumpRate: 5, Amplitude: 0.25, Seed: 9002},
			{Type: LayerHighShriek, BaseFreq: 1500, FreqRange: 2400, JumpRate: 12, Amplitude: 0.3, Rise: 3.0, Seed: 9003},
			{Type: LayerNoiseBurst, Amplitude: 0.1, Seed: 9004},
			{Type: LayerBackgroundNoise, Amplitude: 0.08},
		},
		Noise:  NoiseParams{BurstRate: 5, Threshold: 0.8, BurstAmp: 0.1, FloorAmp: 0.08, BurstSeed: 9004},
		Filter: bansheeFilter(),
	},
	PresetRobot: {
		Duration:   3 * time.Second,
		SampleRate: DefaultSampleRate,
		Channels:   DefaultChannels,
		Layers: [5]LayerParams{
			{Type: LayerPrimaryScream, BaseFreq: 400, FreqRange: 1000, JumpRate: 12, Amplitude: 0.4, Rise: 0.5, Seed: 8080},
			{Type: LayerHarmonicSweep, BaseFreq: 300, SweepRate: 600, FreqRange: 500, JumpRate: 8, Amplitude: 0.2, Seed: 8081},
			{Type: LayerHighShriek, BaseFreq: 1000, FreqRange: 1200, JumpRate: 20, Amplitude: 0.2, Rise: 1.0, Seed: 8082},
			{Type: LayerNoiseBurst, Amplitude: 0.15, Seed: 8083},
			{Type: LayerBackgroundNoise, Amplitude: 0.07},
		},
		Noise:  NoiseParams{BurstRate: 10, Threshold: 0.6, BurstAmp: 0.15, FloorAmp: 0.07, BurstSeed: 8083},
		Filter: robotFilter(),
	},
}
