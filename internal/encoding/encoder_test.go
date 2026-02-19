package encoding

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"OpusFrameSamples", OpusFrameSamples, 960},
		{"MaxOpusFrameBytes", MaxOpusFrameBytes, 3840},
		{"OpusBitrate", OpusBitrate, 64000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}
