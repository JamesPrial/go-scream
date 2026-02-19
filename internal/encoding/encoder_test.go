package encoding

import (
	"encoding/binary"
	"math"
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

// ---------------------------------------------------------------------------
// pcmBytesToInt16
// ---------------------------------------------------------------------------

func TestPcmBytesToInt16_KnownValues(t *testing.T) {
	tests := []struct {
		name string
		pcm  []byte
		want []int16
	}{
		{
			name: "zero",
			pcm:  []byte{0x00, 0x00},
			want: []int16{0},
		},
		{
			name: "positive one",
			pcm:  []byte{0x01, 0x00},
			want: []int16{1},
		},
		{
			name: "negative one",
			pcm:  []byte{0xFF, 0xFF},
			want: []int16{-1},
		},
		{
			name: "256 (0x0100)",
			pcm:  []byte{0x00, 0x01},
			want: []int16{256},
		},
		{
			name: "multiple samples",
			pcm:  []byte{0x00, 0x00, 0x01, 0x00, 0xFF, 0xFF},
			want: []int16{0, 1, -1},
		},
		{
			name: "little-endian 0x0201 = 513",
			pcm:  []byte{0x01, 0x02},
			want: []int16{513},
		},
		{
			name: "0x80FF = -32513 in signed",
			pcm:  []byte{0xFF, 0x80},
			want: []int16{-32513},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pcmBytesToInt16(tt.pcm)
			if len(got) != len(tt.want) {
				t.Fatalf("pcmBytesToInt16(%v) returned %d samples, want %d", tt.pcm, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("sample[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPcmBytesToInt16_EmptyInput(t *testing.T) {
	got := pcmBytesToInt16([]byte{})
	if len(got) != 0 {
		t.Errorf("pcmBytesToInt16(empty) returned %d samples, want 0", len(got))
	}

	got = pcmBytesToInt16(nil)
	if len(got) != 0 {
		t.Errorf("pcmBytesToInt16(nil) returned %d samples, want 0", len(got))
	}
}

func TestPcmBytesToInt16_RoundTrip(t *testing.T) {
	// Create known int16 values, convert to bytes, convert back, verify.
	original := []int16{0, 1, -1, 32767, -32768, 12345, -12345, 100, -100}

	// int16 to little-endian bytes
	pcm := make([]byte, len(original)*2)
	for i, v := range original {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(v))
	}

	got := pcmBytesToInt16(pcm)
	if len(got) != len(original) {
		t.Fatalf("round-trip: got %d samples, want %d", len(got), len(original))
	}
	for i := range original {
		if got[i] != original[i] {
			t.Errorf("round-trip sample[%d] = %d, want %d", i, got[i], original[i])
		}
	}
}

func TestPcmBytesToInt16_MaxValues(t *testing.T) {
	tests := []struct {
		name string
		pcm  []byte
		want int16
	}{
		{
			name: "int16 max (32767)",
			pcm:  []byte{0xFF, 0x7F}, // 32767 in little-endian
			want: math.MaxInt16,
		},
		{
			name: "int16 min (-32768)",
			pcm:  []byte{0x00, 0x80}, // -32768 in little-endian
			want: math.MinInt16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pcmBytesToInt16(tt.pcm)
			if len(got) != 1 {
				t.Fatalf("pcmBytesToInt16 returned %d samples, want 1", len(got))
			}
			if got[0] != tt.want {
				t.Errorf("pcmBytesToInt16(%v) = %d, want %d", tt.pcm, got[0], tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkPcmBytesToInt16_1s_Stereo48k(b *testing.B) {
	// 1 second of stereo 48kHz = 48000 * 2 channels * 2 bytes = 192000 bytes
	pcm := make([]byte, 192000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pcmBytesToInt16(pcm)
	}
}
