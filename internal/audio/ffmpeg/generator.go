package ffmpeg

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os/exec"

	"github.com/JamesPrial/go-scream/internal/audio"
)

// Compile-time check that Generator implements audio.Generator.
var _ audio.Generator = (*Generator)(nil)

// Generator produces raw PCM audio by invoking the ffmpeg executable.
type Generator struct {
	ffmpegPath string
	logger     *slog.Logger
}

// NewGenerator locates the ffmpeg binary on PATH and returns a generator.
// Returns ErrFFmpegNotFound if ffmpeg is not available.
func NewGenerator(logger *slog.Logger) (*Generator, error) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFFmpegNotFound, err)
	}
	return &Generator{ffmpegPath: path, logger: logger}, nil
}

// NewGeneratorWithPath returns a generator using the given ffmpeg binary path.
// No validation is performed on the path.
func NewGeneratorWithPath(path string, logger *slog.Logger) *Generator {
	return &Generator{ffmpegPath: path, logger: logger}
}

// Generate validates params, invokes ffmpeg, and returns the raw PCM audio as an io.Reader.
// Returns an error wrapping ErrFFmpegFailed if the process exits with a non-zero status.
func (g *Generator) Generate(params audio.ScreamParams) (io.Reader, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	args := buildArgs(params)
	cmd := exec.Command(g.ffmpegPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	g.logger.Debug("running ffmpeg", "path", g.ffmpegPath, "args", args)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFFmpegFailed, stderr.String())
	}

	g.logger.Debug("ffmpeg complete", "output_bytes", stdout.Len())
	return bytes.NewReader(stdout.Bytes()), nil
}
