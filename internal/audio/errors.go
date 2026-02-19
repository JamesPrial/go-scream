package audio

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidDuration     = errors.New("duration must be positive")
	ErrInvalidSampleRate   = errors.New("sample rate must be positive")
	ErrInvalidChannels     = errors.New("channels must be 1 or 2")
	ErrInvalidAmplitude    = errors.New("amplitude must be between 0 and 1")
	ErrInvalidFilterCutoff = errors.New("filter cutoff must be non-negative")
	ErrInvalidLimiterLevel = errors.New("limiter level must be between 0 and 1 (exclusive of 0)")
	ErrInvalidCrusherBits  = errors.New("crusher bits must be between 1 and 16")
)

// LayerValidationError wraps an error with the layer index.
type LayerValidationError struct {
	Layer int
	Err   error
}

func (e *LayerValidationError) Error() string {
	return fmt.Sprintf("layer %d: %s", e.Layer, e.Err)
}

func (e *LayerValidationError) Unwrap() error {
	return e.Err
}
