# Explorer Findings: Stage 3 — FFmpeg Backend

**Date:** 2026-02-18
**Working Directory:** /Users/jamesprial/code/go-scream
**Go Version:** 1.25.7 (module: `github.com/JamesPrial/go-scream`)

---

## 1. AudioGenerator Interface

**File:** `/Users/jamesprial/code/go-scream/internal/audio/generator.go`

```go
package audio

import "io"

// AudioGenerator produces raw PCM audio data (s16le, 48kHz, stereo).
type AudioGenerator interface {
    Generate(params ScreamParams) (io.Reader, error)
}
```

**Contract the FFmpeg generator must satisfy:**
- Accept `ScreamParams` as input
- Return `io.Reader` of raw PCM bytes in **s16le** (signed 16-bit little-endian, interleaved stereo) format
- Return a non-nil error if generation fails
- Must call `params.Validate()` before any work (the native generator does this as its first step)
- Byte count: `int(duration.Seconds() * float64(sampleRate)) * channels * 2`

**Critical detail:** The interface contract says s16le PCM output. The FFmpeg backend must therefore output **raw PCM** via `-f s16le` (not ogg, not opus). The encoding layer (Stage 2) handles the OGG/Opus wrapping. The FFmpeg generator sits in the same slot as the native generator — it just replaces the synthesis source.

---

## 2. ScreamParams, LayerParams, NoiseParams, FilterParams

**File:** `/Users/jamesprial/code/go-scream/internal/audio/params.go`

### ScreamParams
```go
type ScreamParams struct {
    Duration   time.Duration
    SampleRate int     // Always 48000 in presets and Randomize()
    Channels   int     // Always 2 in presets and Randomize()
    Seed       int64
    Layers     [5]LayerParams
    Noise      NoiseParams
    Filter     FilterParams
}
```

### LayerType constants
```go
type LayerType int

const (
    LayerPrimaryScream  LayerType = iota  // 0
    LayerHarmonicSweep                    // 1
    LayerHighShriek                       // 2
    LayerNoiseBurst                       // 3
    LayerBackgroundNoise                  // 4
)
```

### LayerParams
```go
type LayerParams struct {
    Type      LayerType
    BaseFreq  float64  // Base frequency in Hz
    FreqRange float64  // Frequency jump range in Hz
    SweepRate float64  // Linear frequency sweep rate (Hz/s), used by harmonic sweep only
    JumpRate  float64  // How often frequency jumps (Hz)
    Amplitude float64  // Layer amplitude [0, 1]
    Rise      float64  // Exponential amplitude rise over time
    Seed      int64    // RNG seed for this layer
}
```

**Layer-to-index mapping (always `[5]LayerParams`):**
- `Layers[0]` — `LayerPrimaryScream`
- `Layers[1]` — `LayerHarmonicSweep`
- `Layers[2]` — `LayerHighShriek`
- `Layers[3]` — `LayerNoiseBurst`
- `Layers[4]` — `LayerBackgroundNoise`

### NoiseParams
```go
type NoiseParams struct {
    BurstRate float64  // Burst frequency for gated noise (Hz)
    Threshold float64  // Gate threshold [0, 1]
    BurstAmp  float64  // Amplitude for noise bursts
    FloorAmp  float64  // Amplitude for background noise floor
    BurstSeed int64    // RNG seed for burst gating
}
```

### FilterParams
```go
type FilterParams struct {
    HighpassCutoff float64  // High-pass filter cutoff (Hz)
    LowpassCutoff  float64  // Low-pass filter cutoff (Hz)
    CrusherBits    int      // Bit depth for bitcrusher (6-12 in Randomize, 1-16 in Validate)
    CrusherMix     float64  // Mix of crushed vs clean signal [0, 1]
    CompRatio      float64  // Compressor ratio
    CompThreshold  float64  // Compressor threshold in dB (always -20 in practice)
    CompAttack     float64  // Compressor attack in ms (always 5 in practice)
    CompRelease    float64  // Compressor release in ms (always 50 in practice)
    VolumeBoostDB  float64  // Volume boost in dB
    LimiterLevel   float64  // Hard limiter level [0, 1] (always 0.95 in practice)
}
```

### Validate() method — rules the FFmpeg generator must enforce
```go
func (p ScreamParams) Validate() error {
    // Duration must be > 0
    // SampleRate must be > 0
    // Channels must be 1 or 2
    // Each Layer.Amplitude must be in [0, 1]
    // Filter.HighpassCutoff must be >= 0
    // Filter.LowpassCutoff must be >= 0
    // Filter.CrusherBits must be in [1, 16]
    // Filter.LimiterLevel must be in (0, 1]
}
```

---

## 3. NativeGenerator — Patterns to Follow

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go`

### Struct and constructor pattern
```go
type NativeGenerator struct{}

func NewNativeGenerator() *NativeGenerator {
    return &NativeGenerator{}
}
```
No state stored in the struct — all state comes from params at call time. The FFmpeg generator should follow this pattern: stateless struct, no persistent goroutines or connections.

### Generate() method pattern
```go
func (g *NativeGenerator) Generate(params audio.ScreamParams) (io.Reader, error) {
    // Step 1: Validate params — ALWAYS first
    if err := params.Validate(); err != nil {
        return nil, fmt.Errorf("invalid params: %w", err)
    }

    // Step 2: Do the work (synthesis / process spawning)
    // ...

    // Step 3: Return io.Reader over the result
    return bytes.NewReader(w.Bytes()), nil
}
```

**Error wrapping style:** `fmt.Errorf("invalid params: %w", err)` — wrap with context using `%w`.

### Seed derivation pattern (important for FFmpeg backend)
The native generator derives per-layer seeds from the global seed using XOR with prime multipliers:
```go
p0.Seed = lp[0].Seed ^ (globalSeed * 1000003)
p1.Seed = lp[1].Seed ^ (globalSeed * 1000033)
p2.Seed = lp[2].Seed ^ (globalSeed * 1000037)
p3.Seed = lp[3].Seed ^ (globalSeed * 1000039)
noiseWithSeed.BurstSeed = noise.BurstSeed ^ (globalSeed * 1000081)
```
The FFmpeg generator uses the same ScreamParams, so these seeds are already computed at the params level. The FFmpeg aevalsrc expression must embed these derived seeds.

### Layer synthesis formulas (what each aevalsrc expression must replicate)

**PrimaryScreamLayer** (`Layers[0]`):
```
envelope = Amplitude * (1 + Rise * t)
freq     = BaseFreq + FreqRange * seededRandom(Seed, floor(t * JumpRate), 137)
sample   = envelope * sin(freq)
```
Where `seededRandom(layerSeed, step, coprime)` is `splitmix64(layerSeed XOR (step * coprime))`.

**HarmonicSweepLayer** (`Layers[1]`):
```
freq   = BaseFreq + SweepRate * t + FreqRange * seededRandom(Seed, floor(t * JumpRate), 251)
sample = Amplitude * sin(freq)
```

**HighShriekLayer** (`Layers[2]`):
```
envelope = Amplitude * (1 + Rise * t)
freq     = BaseFreq + FreqRange * seededRandom(Seed, floor(t * JumpRate), 89)
sample   = envelope * sin(freq)
```

**NoiseBurstLayer** (`Layers[3]`):
```
gate   = seededRandom(BurstSeed, floor(t * BurstRate), 173)
if gate <= Threshold: sample = 0
else:                 sample = BurstAmp * (2 * random(t * sampleRate) - 1)
```

**BackgroundNoiseLayer** (`Layers[4]`):
```
sample = FloorAmp * (2 * random(t * sampleRate + 7777) - 1)
```

**Oscillator model:** Phase accumulator — BUT the native oscillator is stateful (advances phase each sample). The aevalsrc `sin(2*PI*...)` form is stateless and directly integrates the instantaneous frequency. This means the FFmpeg expressions integrate frequency changes differently from the native oscillator, which produces phase-continuous output. The reference bot accepted this trade-off.

### FilterChain processing order (maps to FFmpeg -af chain)
```
Highpass -> Lowpass -> Bitcrusher -> Compressor -> VolumeBoost -> Limiter
```

---

## 4. How Presets Work

**File:** `/Users/jamesprial/code/go-scream/internal/audio/presets.go`

### API
```go
type PresetName string

const (
    PresetClassic    PresetName = "classic"
    PresetWhisper    PresetName = "whisper"
    PresetDeathMetal PresetName = "death-metal"
    PresetGlitch     PresetName = "glitch"
    PresetBanshee    PresetName = "banshee"
    PresetRobot      PresetName = "robot"
)

func AllPresets() []PresetName
func GetPreset(name PresetName) (ScreamParams, bool)
```

Presets are a `map[PresetName]ScreamParams` — they return fully-populated `ScreamParams` structs. The FFmpeg generator receives these the same way as any other params — no special handling needed. All presets have `SampleRate: 48000, Channels: 2`.

**Notable preset ranges** (useful for FFmpeg filter parameter ranges):
| Preset      | Duration | CrusherBits | CompRatio | VolumeBoostDB | HpCutoff | LpCutoff |
|-------------|----------|-------------|-----------|---------------|----------|----------|
| classic     | 3s       | 8           | 8         | 9 dB          | 120 Hz   | 8000 Hz  |
| whisper     | 2s       | 12          | 4         | 6 dB          | 200 Hz   | 6000 Hz  |
| death-metal | 4s       | 6           | 12        | 12 dB         | 80 Hz    | 12000 Hz |
| glitch      | 3s       | 6           | 6         | 8 dB          | 100 Hz   | 10000 Hz |
| banshee     | 4s       | 10          | 6         | 10 dB         | 150 Hz   | 11000 Hz |
| robot       | 3s       | 6           | 10        | 8 dB          | 100 Hz   | 7000 Hz  |

---

## 5. Error Patterns in internal/audio/errors.go

**File:** `/Users/jamesprial/code/go-scream/internal/audio/errors.go`

### Sentinel errors (package-level vars)
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
```

### Structured error type with Unwrap
```go
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
```

### Pattern for the FFmpeg package's own errors
The FFmpeg package (`internal/audio/ffmpeg`) should define its own sentinel errors and wrap them with context using `fmt.Errorf("...: %w", err)`. Error strings should be prefixed with the package domain, for example:
```go
var (
    ErrFFmpegNotFound  = errors.New("ffmpeg: executable not found on PATH")
    ErrFFmpegFailed    = errors.New("ffmpeg: process exited with non-zero status")
    ErrBuildExpr       = errors.New("ffmpeg: failed to build aevalsrc expression")
)
```
Errors from params.Validate() bubble up wrapped: `fmt.Errorf("invalid params: %w", err)`.

---

## 6. Node.js Reference — FFmpeg aevalsrc Command Construction

**File:** `/tmp/scream-reference/scripts/scream.mjs`

### The 5-layer aevalsrc expression template

```javascript
// Layer 1: Primary scream — amplitude-rising sine with stepped frequency jumps
const layer1 = `${p.l1Amp}*(1+${p.l1Rise}*t)*sin(2*PI*t*(${p.l1Base}+${p.l1Range}*random(floor(t*${p.l1JumpRate})*137+${p.l1Seed})))`;

// Layer 2: Harmonic sweep — sine with linear sweep + stepped jumps (no rise)
const layer2 = `${p.l2Amp}*sin(2*PI*t*(${p.l2Base}+${p.l2SweepRate}*t+${p.l2Range}*random(floor(t*${p.l2JumpRate})*251+${p.l2Seed})))`;

// Layer 3: High shriek — amplitude-rising sine with fast stepped jumps
const layer3 = `${p.l3Amp}*(1+${p.l3Rise}*t)*sin(2*PI*t*(${p.l3Base}+${p.l3Range}*random(floor(t*${p.l3JumpRate})*89+${p.l3Seed})))`;

// Layer 4: Noise burst — gated white noise (gt() = greater-than, returns 0 or 1)
const layer4 = `${p.l4Amp}*gt(random(floor(t*${p.l4BurstRate})*173+${p.l4Seed}),${p.l4Threshold})*(2*random(t*48000)-1)`;

// Layer 5: Background noise — constant low-level white noise
const layer5 = `${p.l5Amp}*(2*random(t*48000+7777)-1)`;

const expr = `${layer1}+${layer2}+${layer3}+${layer4}+${layer5}`;
```

### Full FFmpeg invocation from Node.js reference

```javascript
const ffmpeg = spawn('ffmpeg', [
    '-f', 'lavfi',
    '-i', `aevalsrc='${expr}':s=48000:d=${p.duration.toFixed(2)}`,
    // Post-processing filter chain
    '-af', [
        `highpass=f=${p.hpCutoff.toFixed(0)}`,
        `lowpass=f=${p.lpCutoff.toFixed(0)}`,
        `acrusher=bits=${p.crusherBits}:mix=${p.crusherMix.toFixed(2)}:mode=log:aa=1`,
        `acompressor=ratio=${p.compRatio.toFixed(0)}:attack=5:release=50:threshold=-20dB`,
        `volume=${p.volumeBoost.toFixed(1)}dB`,
        'alimiter=limit=0.95:attack=1:release=10',
    ].join(','),
    '-c:a', 'libopus',
    '-b:a', '96k',
    '-vbr', 'off',
    '-f', 'ogg',
    '-page_duration', '20000',
    'pipe:1',
]);
```

### Key observations about the reference implementation

1. **aevalsrc `random()` is NOT deterministic.** FFmpeg's `random(seed)` function uses its own internal RNG seeded by the integer argument. The reference uses `floor(t*jumpRate)*137 + layerSeed` as the seed, which produces stepped pseudo-random values. This approximates the Go `seededRandom(layerSeed, step, coprime)` but is NOT bit-for-bit identical to the native generator's splitmix64.

2. **Output format in reference:** The reference produces OGG/Opus directly (`-c:a libopus -f ogg`). The Go FFmpeg generator must instead output **raw PCM s16le** to satisfy the `AudioGenerator` interface contract, so it should use `-f s16le` and drop the `-c:a` / `-f ogg` flags.

3. **Reference uses stereo implicitly:** No `-ac` flag in the reference — aevalsrc defaults to mono but FFmpeg auto-handles channel layout for encoding. The Go version must explicitly output stereo with `-ac 2`.

4. **alimiter parameters:** `limit=0.95:attack=1:release=10` — the `attack=1` and `release=10` differ from the `CompAttack=5/CompRelease=50` on the compressor. The limiter has its own time constants.

5. **Noise seeds:** Layer 4 uses `l4Seed` (maps to `Layers[3].Seed`), not `Noise.BurstSeed`. Layer 5 uses the constant `7777` offset. The Go struct separates noise params but the FFmpeg expression encodes the seeds inline.

### Go translation of the FFmpeg command for s16le output

```go
// Go equivalent FFmpeg args for raw PCM s16le output (satisfies AudioGenerator interface)
args := []string{
    "-f", "lavfi",
    "-i", fmt.Sprintf("aevalsrc='%s':s=%d:d=%.2f", expr, sampleRate, duration),
    "-af", filterChain,
    "-f", "s16le",
    "-acodec", "pcm_s16le",
    "-ac", strconv.Itoa(channels),
    "-ar", strconv.Itoa(sampleRate),
    "pipe:1",
}
```

---

## 7. Existing internal/audio/ffmpeg/ Directory

**Path:** `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/`

**Status: EMPTY.** The directory exists but contains zero files. This is the target package for Stage 3.

```
/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/
    (no files)
```

---

## 8. go.mod Dependencies

**File:** `/Users/jamesprial/code/go-scream/go.mod`

```
module github.com/JamesPrial/go-scream

go 1.25.7

require (
    github.com/pion/randutil v0.1.0 // indirect
    github.com/pion/rtp v1.10.1 // indirect
    github.com/pion/webrtc/v3 v3.3.6 // indirect
    layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32 // indirect
)
```

**Stage 2 dependencies are already present.** All four were added during the encoding stage. No new external Go module dependencies are needed for Stage 3 — the FFmpeg generator uses only stdlib (`os/exec`, `bytes`, `fmt`, `io`, `strconv`, `strings`) to shell out to the `ffmpeg` binary.

**The only runtime dependency is the `ffmpeg` binary on PATH** (not a Go module).

---

## 9. Existing Encoding Package (Stage 2 — Already Implemented)

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go`

Stage 2 is complete. The encoding package provides:

```go
// OpusFrameEncoder encodes raw PCM audio into a stream of Opus frames.
type OpusFrameEncoder interface {
    EncodeFrames(src io.Reader, sampleRate, channels int) (<-chan []byte, <-chan error)
}

// FileEncoder encodes raw PCM audio into a container format written to dst.
type FileEncoder interface {
    Encode(dst io.Writer, src io.Reader, sampleRate, channels int) error
}
```

**Key constants:**
```go
const (
    OpusFrameSamples  = 960    // 20ms at 48kHz per channel
    MaxOpusFrameBytes = 3840   // max encoded frame size
    OpusBitrate       = 64000  // default bitrate bps
)
```

The FFmpeg generator's `io.Reader` output will be consumed by these encoders in the same pipeline as the native generator's output.

---

## 10. Architecture: How FFmpegGenerator Fits

```
                    audio.AudioGenerator (interface)
                    Generate(ScreamParams) (io.Reader, error)
                              |
              ________________|________________
             |                                 |
    NativeGenerator                   FFmpegGenerator      <-- Stage 3
    (internal/audio/native)           (internal/audio/ffmpeg)
             |                                 |
             | Pure Go                         | os/exec.Cmd
             | sample-by-sample               | spawns ffmpeg process
             | synthesis                       | pipes stdout
             |                                 |
             |_______________ __________________|
                             |
                     io.Reader (s16le PCM)
                             |
                    internal/encoding
                    (OpusFrameEncoder / FileEncoder)
```

---

## 11. FFmpegGenerator Implementation Plan

### Package declaration
```go
// Package ffmpeg provides an FFmpeg-based audio generator that implements
// audio.AudioGenerator by spawning an ffmpeg subprocess with the aevalsrc filter.
package ffmpeg
```

### Struct
```go
type FFmpegGenerator struct {
    // ffmpegPath is the path to the ffmpeg binary. If empty, "ffmpeg" is resolved
    // from PATH at Generate() time using exec.LookPath.
    ffmpegPath string
}

func NewFFmpegGenerator() *FFmpegGenerator {
    return &FFmpegGenerator{}
}

func NewFFmpegGeneratorWithPath(path string) *FFmpegGenerator {
    return &FFmpegGenerator{ffmpegPath: path}
}
```

### aevalsrc expression construction (mapping ScreamParams to Go expressions)

The seeds used in the FFmpeg expression must incorporate the global seed, mirroring the native generator's seed derivation:

```go
// Mirror NativeGenerator's seed derivation:
seed0 := params.Layers[0].Seed ^ (params.Seed * 1000003)
seed1 := params.Layers[1].Seed ^ (params.Seed * 1000033)
seed2 := params.Layers[2].Seed ^ (params.Seed * 1000037)
seed3 := params.Layers[3].Seed ^ (params.Seed * 1000039)
burstSeed := params.Noise.BurstSeed ^ (params.Seed * 1000081)
```

Layer expression templates (Go fmt.Sprintf equivalents):

```go
l0 := params.Layers[0]
layer1 := fmt.Sprintf(
    "%.4f*(1+%.4f*t)*sin(2*PI*t*(%.4f+%.4f*random(floor(t*%.4f)*137+%d)))",
    l0.Amplitude, l0.Rise, l0.BaseFreq, l0.FreqRange, l0.JumpRate, seed0)

l1 := params.Layers[1]
layer2 := fmt.Sprintf(
    "%.4f*sin(2*PI*t*(%.4f+%.4f*t+%.4f*random(floor(t*%.4f)*251+%d)))",
    l1.Amplitude, l1.BaseFreq, l1.SweepRate, l1.FreqRange, l1.JumpRate, seed1)

l2 := params.Layers[2]
layer3 := fmt.Sprintf(
    "%.4f*(1+%.4f*t)*sin(2*PI*t*(%.4f+%.4f*random(floor(t*%.4f)*89+%d)))",
    l2.Amplitude, l2.Rise, l2.BaseFreq, l2.FreqRange, l2.JumpRate, seed2)

// Note: noise burst uses gt() for gating, random() for noise source
layer4 := fmt.Sprintf(
    "%.4f*gt(random(floor(t*%.4f)*173+%d),%.4f)*(2*random(t*%d)-1)",
    params.Noise.BurstAmp, params.Noise.BurstRate, burstSeed,
    params.Noise.Threshold, params.SampleRate)

layer5 := fmt.Sprintf(
    "%.4f*(2*random(t*%d+7777)-1)",
    params.Noise.FloorAmp, params.SampleRate)
```

### Filter chain construction

```go
fp := params.Filter
filterParts := []string{
    fmt.Sprintf("highpass=f=%.0f", fp.HighpassCutoff),
    fmt.Sprintf("lowpass=f=%.0f", fp.LowpassCutoff),
    fmt.Sprintf("acrusher=bits=%d:mix=%.2f:mode=log:aa=1", fp.CrusherBits, fp.CrusherMix),
    fmt.Sprintf("acompressor=ratio=%.0f:attack=%.0f:release=%.0f:threshold=%.0fdB",
        fp.CompRatio, fp.CompAttack, fp.CompRelease, fp.CompThreshold),
    fmt.Sprintf("volume=%.1fdB", fp.VolumeBoostDB),
    fmt.Sprintf("alimiter=limit=%.2f:attack=1:release=10", fp.LimiterLevel),
}
filterChain := strings.Join(filterParts, ",")
```

### Full args construction (s16le output)

```go
durationSecs := params.Duration.Seconds()
aevalsrcArg := fmt.Sprintf("aevalsrc='%s':s=%d:d=%.2f",
    expr, params.SampleRate, durationSecs)

args := []string{
    "-f", "lavfi",
    "-i", aevalsrcArg,
    "-af", filterChain,
    "-f", "s16le",
    "-acodec", "pcm_s16le",
    "-ac", strconv.Itoa(params.Channels),
    "-ar", strconv.Itoa(params.SampleRate),
    "pipe:1",
}
```

### Process management and io.Reader return

The critical design decision: the Generate() method must return an `io.Reader`. Two approaches:

**Option A: Buffer all output (mirrors NativeGenerator's bytes.NewReader)**
```go
cmd := exec.Command(ffmpegPath, args...)
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
    return nil, fmt.Errorf("ffmpeg: %w: %s", ErrFFmpegFailed, stderr.String())
}
return bytes.NewReader(stdout.Bytes()), nil
```
Pros: Simple, deterministic, matches NativeGenerator's in-memory approach.
Cons: High memory usage for long screams.

**Option B: Streaming via pipe (cmd.StdoutPipe())**
```go
cmd := exec.Command(ffmpegPath, args...)
stdout, err := cmd.StdoutPipe()
// ... cmd.Start() ... return stdout, nil
```
Pros: Lower memory; consumer can start reading before ffmpeg finishes.
Cons: Process lifetime must be managed; errors only appear at read time.

**Recommendation:** Option A (buffer) for correctness and simplicity, matching the NativeGenerator pattern. The 4s max scream at 48kHz stereo s16le = 4 * 48000 * 2 * 2 = 768000 bytes (750 KB) — well within acceptable memory bounds.

---

## 12. Test Pattern to Follow

**File:** `/Users/jamesprial/code/go-scream/internal/audio/native/generator_test.go`

The native generator test suite defines patterns the FFmpeg generator tests should mirror:

```go
// Interface compliance test (compile-time check)
var _ audio.AudioGenerator = NewFFmpegGenerator()

// Tests to replicate:
// - TestFFmpegGenerator_CorrectByteCount  (totalSamples * channels * 2)
// - TestFFmpegGenerator_NonSilent         (at least one non-zero byte)
// - TestFFmpegGenerator_Deterministic     (same params -> same output)
// - TestFFmpegGenerator_DifferentSeeds    (different Seed -> different output)
// - TestFFmpegGenerator_AllPresets        (all 6 presets succeed)
// - TestFFmpegGenerator_InvalidParams     (Duration=0 returns error)
// - TestFFmpegGenerator_MonoOutput        (Channels=1 -> half byte count)
// - TestFFmpegGenerator_S16LERange        (valid int16 range, not all extremes)
// - BenchmarkFFmpegGenerator_Classic      (benchmark)
```

**Build tag for FFmpeg tests:** Tests should only run when `ffmpeg` is available. Use `t.Skip` with `exec.LookPath("ffmpeg")` check inside each test, or use a build tag `//go:build integration`.

---

## 13. Potential Implementation Pitfalls

### aevalsrc expression quoting
The aevalsrc expression contains single-quote wrapping on the command line:
```
aevalsrc='EXPR':s=48000:d=3.00
```
When using `exec.Command` (no shell), the quotes are part of the ffmpeg filter argument string and must be included literally — they are not shell-level quotes. The Go impl should include them in the string passed to `-i`.

Alternatively, some FFmpeg versions accept the expression without quotes when passed as a Go string to exec.Command (since there's no shell expansion). Test both: `aevalsrc=EXPR:s=...` (no quotes) and `aevalsrc='EXPR':s=...` (with quotes).

### Negative seed values
`params.Seed` is `int64`, so derived seeds like `lp[0].Seed ^ (globalSeed * 1000003)` can be negative. The `random(seed)` function in FFmpeg's aevalsrc accepts negative integer seeds — this is fine.

### Numeric formatting
Use consistent format verbs:
- Frequencies: `%.4f` (enough precision, no trailing noise)
- Amplitudes: `%.4f`
- JumpRate/BurstRate: `%.4f`
- FilterParams ints (CrusherBits): `%d`
- FilterParams float filters: `%.0f` for Hz values, `%.2f` for mix ratios

### aevalsrc `c=` channel spec vs `-ac`
For multi-channel output from aevalsrc, the filter itself may need `c=FL+FR` or similar. For mono generation (the reference bot's approach), omit the channel spec and let FFmpeg upmix. The simpler approach matching the reference: generate mono in aevalsrc, let `-ac 2` duplicate to stereo.

---

## 14. Summary Table

| Item | Location | Status |
|------|----------|--------|
| `AudioGenerator` interface | `/Users/jamesprial/code/go-scream/internal/audio/generator.go` | EXISTS — must implement |
| `ScreamParams` + related structs | `/Users/jamesprial/code/go-scream/internal/audio/params.go` | EXISTS — all fields documented above |
| `NativeGenerator` (pattern reference) | `/Users/jamesprial/code/go-scream/internal/audio/native/generator.go` | EXISTS — follow patterns |
| Preset system | `/Users/jamesprial/code/go-scream/internal/audio/presets.go` | EXISTS — 6 presets, all use 48kHz stereo |
| Error patterns | `/Users/jamesprial/code/go-scream/internal/audio/errors.go` | EXISTS — sentinel vars + structured type |
| Node.js FFmpeg reference | `/tmp/scream-reference/scripts/scream.mjs` | EXISTS — full aevalsrc expressions documented |
| `internal/audio/ffmpeg/` directory | `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/` | EXISTS but EMPTY — target for Stage 3 |
| go.mod dependencies | `/Users/jamesprial/code/go-scream/go.mod` | No new Go deps needed for Stage 3 |
| Encoding package (Stage 2) | `/Users/jamesprial/code/go-scream/internal/encoding/` | EXISTS and complete |

---

## 15. Files to Create for Stage 3

| File | Purpose |
|------|---------|
| `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator.go` | `FFmpegGenerator` struct implementing `AudioGenerator` |
| `/Users/jamesprial/code/go-scream/internal/audio/ffmpeg/generator_test.go` | Tests mirroring native generator test suite |

No changes to existing files are expected. The `internal/audio/ffmpeg/` package is a drop-in alternative to `internal/audio/native/`.
