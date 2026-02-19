package audio

import "io"

// Generator produces raw PCM audio data (s16le, 48kHz, stereo).
type Generator interface {
	Generate(params ScreamParams) (io.Reader, error)
}


