package audio

import "io"

// AudioGenerator produces raw PCM audio data (s16le, 48kHz, stereo).
type AudioGenerator interface {
	Generate(params ScreamParams) (io.Reader, error)
}
