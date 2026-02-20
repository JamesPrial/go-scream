package native

import "math"

// oscillator generates audio samples using a phase accumulator pattern.
type oscillator struct {
	pos        float64 // phase accumulator in [0, 1)
	sampleRate float64
}

// newOscillator creates a new oscillator at the given sample rate.
func newOscillator(sampleRate int) *oscillator {
	return &oscillator{sampleRate: float64(sampleRate)}
}

// sin generates a sine wave sample at the given frequency and advances the phase.
func (o *oscillator) sin(freq float64) float64 {
	sample := math.Sin(2 * math.Pi * o.pos)
	o.pos += freq / o.sampleRate
	// Keep phase in [0, 1) to prevent floating point drift
	if o.pos >= 1.0 {
		o.pos -= 1.0
	}
	return sample
}

// saw generates a sawtooth wave sample at the given frequency and advances the phase.
func (o *oscillator) saw(freq float64) float64 {
	sample := 2*o.pos - 1 // Maps [0,1) to [-1,1)
	o.pos += freq / o.sampleRate
	if o.pos >= 1.0 {
		o.pos -= 1.0
	}
	return sample
}

// phase returns the current oscillator phase [0, 1).
func (o *oscillator) phase() float64 {
	return o.pos
}

// reset sets the oscillator phase to zero.
func (o *oscillator) reset() {
	o.pos = 0
}
