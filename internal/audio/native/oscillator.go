package native

import "math"

// Oscillator generates audio samples using a phase accumulator pattern.
type Oscillator struct {
	phase      float64
	sampleRate float64
}

// NewOscillator creates a new oscillator at the given sample rate.
func NewOscillator(sampleRate int) *Oscillator {
	return &Oscillator{sampleRate: float64(sampleRate)}
}

// Sin generates a sine wave sample at the given frequency and advances the phase.
func (o *Oscillator) Sin(freq float64) float64 {
	sample := math.Sin(2 * math.Pi * o.phase)
	o.phase += freq / o.sampleRate
	// Keep phase in [0, 1) to prevent floating point drift
	if o.phase >= 1.0 {
		o.phase -= 1.0
	}
	return sample
}

// Saw generates a sawtooth wave sample at the given frequency and advances the phase.
func (o *Oscillator) Saw(freq float64) float64 {
	sample := 2*o.phase - 1 // Maps [0,1) to [-1,1)
	o.phase += freq / o.sampleRate
	if o.phase >= 1.0 {
		o.phase -= 1.0
	}
	return sample
}

// Phase returns the current oscillator phase [0, 1).
func (o *Oscillator) Phase() float64 {
	return o.phase
}

// Reset sets the oscillator phase to zero.
func (o *Oscillator) Reset() {
	o.phase = 0
}
