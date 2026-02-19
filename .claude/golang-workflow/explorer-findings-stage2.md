# Explorer Findings: Stage 2 — Audio Encoding

**Date:** 2026-02-18
**Working Directory:** /Users/jamesprial/code/go-scream
**Go Version:** 1.25.7 (module: `github.com/JamesPrial/go-scream`)

---

## Repository Structure (Complete File Listing)

```
/Users/jamesprial/code/go-scream/
├── go.mod                                          (no go.sum — no dependencies yet)
├── cmd/
│   ├── scream/                                     (EMPTY)
│   └── skill/                                      (EMPTY)
├── internal/
│   ├── audio/
│   │   ├── errors.go
│   │   ├── generator.go
│   │   ├── params.go
│   │   ├── params_test.go
│   │   ├── presets.go
│   │   ├── presets_test.go
│   │   └── native/
│   │       ├── filters.go
│   │       ├── filters_test.go
│   │       ├── generator.go
│   │       ├── generator_test.go
│   │       ├── layers.go
│   │       ├── layers_test.go
│   │       ├── oscillator.go
│   │       └── oscillator_test.go
│   ├── config/                                     (EMPTY)
│   ├── discord/                                    (EMPTY)
│   ├── encoding/                                   (EMPTY — target for Stage 2)
│   └── scream/                                     (EMPTY)
└── pkg/
    └── version/                                    (EMPTY)
```

---

## go.mod Contents

```
module github.com/JamesPrial/go-scream

go 1.25.7
```

**No external dependencies yet.** No `go.sum` file exists. No `vendor/` directory.

Stage 2 will require adding:
- `layeh.com/gopus` — Opus encoder (CGo wrapper around libopus)
- `github.com/pion/webrtc/v3` — for `pkg/media/oggwriter`

---

## AudioGenerator Interface

**File:** `/Users/jamesprial/code/go-scream/internal/audio/generator.go`

```go
package audio

import "io"

// AudioGenerator produces raw PCM audio data (s16le, 48kHz, stereo).
type AudioGenerator interface {
    Generate(params ScreamParams) (io.Reader, error)
}
```

**Contract:**
- Input: `ScreamParams`
- Output: `io.Reader` of raw PCM bytes in **s16le** (signed 16-bit little-endian) format
- Channel layout: same mono sample written to both L and R channels (interleaved: L0 R0 L1 R1 …)
- Fixed parameters in all presets: 48000 Hz, 2 channels

**Byte count formula:**
`totalSamples * channels * 2`
where `totalSamples = int(duration.Seconds() * float64(sampleRate))`

For a 3-second 48kHz stereo scream: `3 * 48000 * 2 * 2 = 576000 bytes`

---

## ScreamParams Struct

**File:** `/Users/jamesprial/code/go-scream/internal/audio/params.go`

```go
type ScreamParams struct {
    Duration   time.Duration
    SampleRate int           // Always 48000 in all presets and Randomize()
    Channels   int           // Always 2 in all presets and Randomize()
    Seed       int64
    Layers     [5]LayerParams
    Noise      NoiseParams
    Filter     FilterParams
}
```

**All 6 presets and `Randomize()` hardcode `SampleRate: 48000, Channels: 2`.**
Duration ranges from 2s (`PresetWhisper`) to 4s (`PresetDeathMetal`, `PresetBanshee`).

---

## NativeGenerator — PCM Output Details

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go`

Key implementation facts for the encoder:

1. **Output format:** `s16le` — little-endian signed 16-bit, interleaved stereo
2. **Stereo layout:** Mono signal duplicated to both channels per sample:
   ```go
   binary.LittleEndian.PutUint16(sampleBuf, uint16(s16))
   for range channels {
       w.WriteByte(sampleBuf[0])
       w.WriteByte(sampleBuf[1])
   }
   ```
   So the byte stream is: `[L_low L_high R_low R_high L_low L_high R_low R_high ...]`
   i.e., each stereo frame = 4 bytes (2 bytes left + 2 bytes right, interleaved)
3. **Complete buffer returned at once:** `bytes.NewReader(w.Bytes())` — the entire audio is materialized in memory before returning. No streaming generation.
4. **Implements:** `audio.AudioGenerator` (verified by interface compliance test)

---

## Opus Frame Math

For the encoder (Discord-compatible Opus at 48kHz stereo):

```
Frame size:    960 samples (20ms at 48kHz)
Channels:      2 (stereo)
Bytes/sample:  2 (s16le)
Bytes/frame:   960 * 2 * 2 = 3840 bytes of PCM per frame

Total frames for a 3s scream (48000 samples/s * 3s = 144000 samples):
  144000 / 960 = 150 frames

For a 4s scream:
  192000 / 960 = 200 frames
```

A partial frame at the end (if `totalSamples % 960 != 0`) needs zero-padding before encoding.

---

## Error Types Available

**File:** `/Users/jamesprial/code/go-scream/internal/audio/errors.go`

```go
var (
    ErrInvalidDuration     = errors.New("duration must be positive")
    ErrInvalidSampleRate   = errors.New("sample rate must be positive")
    ErrInvalidChannels     = errors.New("channels must be 1 or 2")
    ErrInvalidAmplitude    = errors.New("amplitude must be between 0 and 1")
    ErrInvalidFilterCutoff = errors.New("filter cutoff must be non-negative")
    ErrInvalidLimiterLevel = errors.New("limiter level must be between 0 and 1 (exclusive of 0)")
    ErrInvalidCrusherBits  = errors.New("crusher bits must be between 1 and 16")
)

type LayerValidationError struct {
    Layer int
    Err   error
}
```

Stage 2 will need its own error types in `internal/encoding/`.

---

## internal/encoding/ Directory Status

**EMPTY.** The directory exists at `/Users/jamesprial/code/go-scream/internal/encoding/` but contains no files. This is the target package for Stage 2.

---

## External Dependencies Required for Stage 2

Neither dependency is yet present in go.mod.

### 1. `layeh.com/gopus`

- **Purpose:** CGo bindings for libopus encoder/decoder
- **Key API** (from public documentation):
  ```go
  // NewEncoder creates a new Opus encoder
  // sampleRate: 48000
  // channels: 1 or 2
  // application: gopus.Audio (0), gopus.Voip (1), gopus.RestrictedLowdelay (2)
  func NewEncoder(sampleRate, channels, application int) (*Encoder, error)

  // Encode encodes a 16-bit PCM buffer to Opus
  // pcm: []int16, length must be frameSize * channels (e.g., 960*2=1920 for stereo)
  // frameSize: number of samples per channel (must be 960 for 20ms at 48kHz)
  // maxEncodedSize: upper bound on output bytes (e.g., 4000)
  func (e *Encoder) Encode(pcm []int16, frameSize, maxEncodedSize int) ([]byte, error)
  ```
- **Notes:**
  - Requires libopus C library installed (`brew install opus` on macOS)
  - Build tag: CGo must be enabled
  - gopus.Audio = 2048 (constant for audio content)
  - Discord voice uses 48000 Hz, 2 channels, Opus Audio application mode
  - Input to `Encode`: `[]int16` (NOT `[]byte`), length = `frameSize * channels`
  - PCM must be converted from `[]byte` (s16le) to `[]int16` before encoding

### 2. `github.com/pion/webrtc/v3/pkg/media/oggwriter`

- **Purpose:** Writes Opus frames into an OGG container
- **Key API:**
  ```go
  // New creates a new OGG writer
  // fileName: output file path (or use NewWith for io.Writer)
  // sampleRate: 48000
  // channelCount: 2
  func New(fileName string, sampleRate uint32, channelCount uint16) (*OggWriter, error)

  // NewWith creates an OGG writer from an existing io.Writer
  func NewWith(out io.Writer, sampleRate uint32, channelCount uint16) (*OggWriter, error)

  // WriteRTP writes an RTP packet (the Opus frame payload goes in RTP data)
  // Can use media.Sample for simpler usage
  func (o *OggWriter) WriteRTP(packet *rtp.Packet) error

  // Close finalizes the OGG stream
  func (o *OggWriter) Close() error
  ```
- **Notes:**
  - The `pion/webrtc/v3` module is large; only `pkg/media/oggwriter` is needed
  - Alternative lighter option: `github.com/pion/oggwriter` (standalone package)
  - For simple usage, can wrap with `github.com/pion/webrtc/v3/pkg/media` `Sample` struct
  - Direct RTP usage: build minimal RTP header with `pion/rtp` and pass Opus payload
  - pion/webrtc/v3 requires Go 1.17+; current module is Go 1.25.7 (compatible)

---

## WAV File Format for Dry-Run Output

WAV format for PCM s16le stereo 48kHz:
```
RIFF header:     44 bytes
  "RIFF"         4 bytes (magic)
  fileSize-8     4 bytes uint32 LE
  "WAVE"         4 bytes (magic)
  "fmt "         4 bytes (subchunk ID)
  16             4 bytes uint32 LE (PCM fmt chunk size)
  1              2 bytes uint16 LE (PCM audio format)
  2              2 bytes uint16 LE (numChannels)
  48000          4 bytes uint32 LE (sampleRate)
  192000         4 bytes uint32 LE (byteRate = sampleRate * numChannels * bitsPerSample/8)
  4              2 bytes uint16 LE (blockAlign = numChannels * bitsPerSample/8)
  16             2 bytes uint16 LE (bitsPerSample)
  "data"         4 bytes (subchunk ID)
  dataSize       4 bytes uint32 LE (numSamples * numChannels * bitsPerSample/8)
```

**No external dependency needed for WAV** — `encoding/binary` from stdlib is sufficient.

---

## Proposed Stage 2 Package: `internal/encoding`

### Encoder Interface Design

```go
package encoding

import "io"

// OpusEncoder encodes PCM s16le data to Opus frames.
// Each call to Encode consumes exactly one 960-sample frame.
type OpusEncoder interface {
    Encode(pcm []int16) ([]byte, error)
    Close() error
}

// FrameWriter writes encoded audio frames to a container format.
type FrameWriter interface {
    WriteFrame(opusData []byte) error
    Close() error
}
```

### Encoding Pipeline

```
io.Reader (PCM s16le from AudioGenerator)
    |
    v
Read 3840 bytes at a time (960 samples * 2 ch * 2 bytes)
    |
    v
Convert []byte -> []int16 (3840 bytes -> 1920 int16s)
    |
    v
gopus Encoder.Encode(pcm []int16, frameSize=960, maxEncodedSize=4000)
    |
    v
[]byte (Opus frame, typically 20-200 bytes)
    |
    +---> chan []byte (for Discord voice connection)
    |
    +---> OGG writer (for file output)
    |
    +---> WAV writer (for dry-run, writes original PCM)
```

### Key Constants for internal/encoding

```go
const (
    OpusFrameSize     = 960    // samples per channel per frame (20ms at 48kHz)
    OpusSampleRate    = 48000
    OpusChannels      = 2
    PCMBytesPerFrame  = OpusFrameSize * OpusChannels * 2  // 3840
    PCMInt16PerFrame  = OpusFrameSize * OpusChannels      // 1920
    MaxOpusFrameBytes = 4000   // upper bound for gopus.Encode
)
```

---

## Byte-to-int16 Conversion

The PCM reader yields s16le bytes. Converting to `[]int16` for gopus:

```go
// Convert 3840 s16le bytes -> 1920 int16 samples (for gopus.Encode)
func bytesToInt16(b []byte) []int16 {
    samples := make([]int16, len(b)/2)
    for i := range samples {
        samples[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
    }
    return samples
}
```

---

## Stage 1 Test Status (For Reference)

All 62 tests pass, 0 failures, no race conditions:
- `internal/audio`: 87.5% coverage
- `internal/audio/native`: 100.0% coverage

Stage 1 received APPROVE verdict from reviewer on 2026-02-18.

---

## Summary of Gaps / What Stage 2 Must Create

| Item | Status | Target Path |
|------|--------|-------------|
| `internal/encoding/` package | EMPTY | `/Users/jamesprial/code/go-scream/internal/encoding/` |
| Encoder interfaces | Missing | `internal/encoding/encoder.go` |
| Opus frame encoder | Missing | `internal/encoding/opus.go` |
| OGG file writer | Missing | `internal/encoding/ogg.go` |
| WAV file writer | Missing | `internal/encoding/wav.go` |
| `layeh.com/gopus` dependency | Not in go.mod | Must run `go get layeh.com/gopus` |
| `github.com/pion/webrtc/v3` | Not in go.mod | Must run `go get github.com/pion/webrtc/v3` (or lighter `github.com/pion/oggwriter`) |
| `internal/encoding/` tests | Missing | `internal/encoding/*_test.go` |

### Dependency Decision Note

`github.com/pion/webrtc/v3` is a very large module (~40 transitive dependencies). If only OGG writing is needed, consider using one of:
- `github.com/pion/oggwriter` (standalone, minimal deps) — preferred if available
- `github.com/jfreymuth/oggvorbis` (different codec, not applicable)
- Hand-rolled OGG page writer using only stdlib (OGG is a simple container format)

For Discord voice output, the Opus frames are sent directly over UDP via the Discord voice websocket — no container format needed for that path. OGG is only for file persistence.
