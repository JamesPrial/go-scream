// Package ffmpeg provides an FFmpeg-based audio generator backend.
package ffmpeg

import "errors"

// ErrFFmpegNotFound is returned when the ffmpeg executable cannot be found on PATH.
var ErrFFmpegNotFound = errors.New("ffmpeg: executable not found on PATH")

// ErrFFmpegFailed is returned when the ffmpeg process exits with a non-zero status.
var ErrFFmpegFailed = errors.New("ffmpeg: process failed")
