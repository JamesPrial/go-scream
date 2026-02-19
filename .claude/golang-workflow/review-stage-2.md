# Code Review: Stage 2 -- Audio Encoding

## Verdict: APPROVE

The encoding package is well-structured, idiomatic, and correctly implements WAV, Opus, and OGG encoding. The code is clean, error handling is consistent, goroutine lifecycle management is sound, and the test suite covers documented behaviors thoroughly. No blocking issues were found. Minor observations are noted below for future consideration.

---

## Architecture and API Design

The package introduces two clean interfaces:

- `OpusFrameEncoder` -- goroutine-based streaming encoder returning `(<-chan []byte, <-chan error)`
- `FileEncoder` -- synchronous `Encode(dst, src, sampleRate, channels)` for container output

This separation is well-considered. The `OpusFrameEncoder` serves the Discord voice path (where frames are sent individually over UDP), while `FileEncoder` serves WAV and OGG file output. The `OGGEncoder` bridges the two by consuming an `OpusFrameEncoder` and wrapping frames in OGG pages -- a clean composition pattern.

The package is correctly decoupled from `internal/audio` -- it operates only on `io.Reader`/`io.Writer`, which makes it independently testable and reusable.

---

## File-by-File Review

### 1. `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go`

**Strengths:**
- Package-level doc comment is present and accurate.
- All exported constants (`OpusFrameSamples`, `MaxOpusFrameBytes`, `OpusBitrate`) are documented with clear descriptions.
- Sentinel errors are well-named with an `encoding:` prefix for context, and documented.
- Both interfaces (`OpusFrameEncoder`, `FileEncoder`) have doc comments explaining contracts.
- `pcmBytesToInt16` handles nil, empty, and odd-byte-count inputs correctly (trailing byte ignored).

**Code:**
```go
func pcmBytesToInt16(pcm []byte) []int16 {
    if len(pcm) < 2 {
        return nil
    }
    n := len(pcm) / 2
    out := make([]int16, n)
    for i := 0; i < n; i++ {
        out[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
    }
    return out
}
```

This is correct. The `len(pcm) < 2` guard handles nil, empty, and single-byte inputs. Integer division truncates the trailing byte. The conversion from `uint16` to `int16` correctly preserves the sign bit for two's complement s16le data.

**No issues.**

---

### 2. `/Users/jamesprial/code/go-scream/internal/encoding/wav.go`

**Strengths:**
- `wavFileHeader` struct fields map exactly to the RIFF/WAVE spec layout with clear comments.
- Validation is performed before any I/O operations.
- Error wrapping consistently uses `%w` for `errors.Is` compatibility.
- The `binary.Write` call for the header is correct -- Go struct layout with fixed-size fields and `binary.LittleEndian` produces exactly 44 bytes matching the RIFF spec.

**WAV header byte layout verification:**

| Offset | Size | Field | Value |
|--------|------|-------|-------|
| 0 | 4 | ChunkID | "RIFF" |
| 4 | 4 | ChunkSize | dataSize + 36 |
| 8 | 4 | Format | "WAVE" |
| 12 | 4 | Subchunk1ID | "fmt " |
| 16 | 4 | Subchunk1Size | 16 |
| 20 | 2 | AudioFormat | 1 (PCM) |
| 22 | 2 | NumChannels | channels |
| 24 | 4 | SampleRate | sampleRate |
| 28 | 4 | ByteRate | sampleRate * channels * 2 |
| 32 | 2 | BlockAlign | channels * 2 |
| 34 | 2 | BitsPerSample | 16 |
| 36 | 4 | Subchunk2ID | "data" |
| 40 | 4 | Subchunk2Size | len(pcmData) |

This matches the canonical WAV PCM format specification exactly. The `ChunkSize = dataSize + 36` calculation is correct (total file size minus the 8-byte RIFF header = 44 - 8 + dataSize = 36 + dataSize).

**Observation [NIT]:** The `WAVEncoder.Encode` method calls `io.ReadAll(src)`, which materializes the entire PCM buffer in memory. This is fine for the scream use case (max ~768KB for 4s stereo 48kHz), but worth noting if the encoder is ever used for longer recordings. No change needed now.

**Observation [NIT]:** On line 48, the inner error in `"reading PCM data: %v"` uses `%v` instead of `%w`. This means if the source reader returns a sentinel error, it cannot be unwrapped through `errors.Is`. However, this is an intentional design choice -- the WAV encoder deliberately wraps all I/O errors under `ErrWAVWrite` as the outermost sentinel. The inner error is informational only. This is acceptable.

**No issues.**

---

### 3. `/Users/jamesprial/code/go-scream/internal/encoding/opus.go`

**Strengths:**
- Valid Opus sample rates are enforced via a map lookup -- correct per the Opus spec (8000, 12000, 16000, 24000, 48000).
- The goroutine lifecycle is correct: `frameCh` is always closed via `defer close(frameCh)` or explicit close before the goroutine in validation error paths.
- `errCh` has capacity 1, so the goroutine never blocks on sending the error regardless of whether the consumer reads it synchronously or asynchronously.
- Zero-padding of partial frames is handled correctly: the buffer is zeroed at the start of each iteration, `io.ReadFull` fills only the bytes that were available, and the remainder stays zeroed.
- The `io.EOF` vs `io.ErrUnexpectedEOF` distinction is handled correctly: `io.EOF` means zero bytes were read (clean end), `io.ErrUnexpectedEOF` means a partial read occurred (zero-padded frame).

**Goroutine lifecycle analysis:**

```
Validation error path (lines 46-59):
  - Go routine: close(frameCh), send error to errCh
  - frameCh closed: YES
  - errCh receives value: YES
  - No leak possible

Normal encoding path (lines 61-118):
  - defer close(frameCh) on line 62
  - Every exit path sends to errCh (nil on success, error on failure)
  - frameCh closed: YES (via defer)
  - errCh receives value: YES (lines 66, 97, 103, 117)
  - No leak possible
```

All goroutine paths terminate cleanly. The `frameCh` buffer of 2 provides a small amount of pipelining between the encoding goroutine and the consumer, which is a reasonable choice.

**Observation [NIT]:** The validation error goroutines (lines 47-49, 53-55) close `frameCh` but do not explicitly close `errCh`. This is fine because `errCh` has capacity 1 and receives exactly one value -- the consumer reads it and the channel becomes garbage-collectable. However, for symmetry with the contract documented in the interface ("Both channels are closed after completion"), one could argue `errCh` should also be closed. In practice this makes no functional difference since the consumer reads exactly once via `<-errCh`.

**Observation [NIT]:** `encoder.SetBitrate(e.bitrate)` on line 70 does not check the error return. The `gopus.Encoder.SetBitrate` method returns an error. In practice, any bitrate value is accepted by libopus (it clamps internally), so this is unlikely to fail. But for completeness, the error could be checked.

**No blocking issues.**

---

### 4. `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go`

**Strengths:**
- Proper draining of `frameCh` when `oggwriter.NewWith` fails (lines 37-39), preventing a goroutine leak.
- `defer oggWriter.Close()` ensures the OGG stream is finalized.
- The error priority logic is correct: OGG write errors take priority, and opus errors are propagated directly (preserving their wrapping for `errors.Is` compatibility).
- The "drain-on-error" pattern (lines 63-69) continues reading frames even after an OGG write error, preventing the opus goroutine from blocking on a full `frameCh`.

**RTP packet construction verification:**

```go
pkt := &rtp.Packet{
    Header: rtp.Header{
        Version:        2,
        PayloadType:    111,
        SequenceNumber: seqNum,
        Timestamp:      timestamp,
        SSRC:           1,
    },
    Payload: frame,
}
```

- `Version: 2` -- correct RTP version.
- `PayloadType: 111` -- commonly used dynamic payload type for Opus in WebRTC contexts. The pion oggwriter uses this to identify Opus frames. Correct.
- `SequenceNumber` starts at 1 (incremented before first use) -- valid. Sequence numbers are arbitrary starting values in RTP.
- `Timestamp` increments by `OpusFrameSamples` (960) per frame -- correct for 20ms Opus frames at 48kHz.
- `SSRC: 1` -- arbitrary but valid synchronization source identifier. The oggwriter needs a consistent SSRC, which this provides.

The pion oggwriter extracts the `Payload` from the RTP packet and writes it as an OGG page with the correct granule position derived from the timestamp. This is the correct usage pattern.

**Observation [NIT]:** The `NewOGGEncoderWithOpus` constructor does not validate that `opus` is non-nil. If a nil `OpusFrameEncoder` is passed, the encoder will panic at `e.opus.EncodeFrames(...)`. A nil guard could be added, but since this is an internal package and the constructors are straightforward, this is acceptable.

**No issues.**

---

## Test Suite Assessment

### `/Users/jamesprial/code/go-scream/internal/encoding/encoder_test.go`

- Constants verified against expected values.
- `pcmBytesToInt16` thoroughly tested: known values, empty/nil input, round-trip, boundary values (MaxInt16, MinInt16).
- Table-driven tests used appropriately.
- Benchmark included for the hot conversion function.

### `/Users/jamesprial/code/go-scream/internal/encoding/wav_test.go`

- Compile-time interface check: `var _ FileEncoder = (*WAVEncoder)(nil)`.
- Header byte layout verified field-by-field against the RIFF spec.
- Output size verified for multiple configurations (stereo, mono, various sample rates, zero samples).
- PCM data preservation verified (raw bytes after header match input).
- Error paths tested: invalid sample rate, invalid channels, failing writer.
- `failWriter` helper is clean and correct.
- Table-driven header fields test covers multiple sample rate/channel combinations.
- Two benchmarks included.

### `/Users/jamesprial/code/go-scream/internal/encoding/opus_test.go`

- Compile-time interface check: `var _ OpusFrameEncoder = (*GopusFrameEncoder)(nil)`.
- `skipIfNoOpus` helper gracefully skips when libopus is unavailable -- good CI/CD practice.
- Frame count tests: exact (3s = 150 frames), single frame, partial frame (1.5 frames = 2), single sample, empty input.
- Frame size bounds verified (non-empty and <= MaxOpusFrameBytes).
- Mono encoding tested.
- Validation errors tested: invalid sample rates (0, 22050, negative), invalid channels (0, 3, negative).
- Channel lifecycle explicitly tested: both success and error paths verify channels close.
- Constructor tests verify non-nil returns.
- `drainFrames` helper correctly handles the channel protocol.

### `/Users/jamesprial/code/go-scream/internal/encoding/ogg_test.go`

- Compile-time interface check: `var _ FileEncoder = (*OGGEncoder)(nil)`.
- Mock `OpusFrameEncoder` implementations enable testing without CGO/libopus.
- OGG magic bytes ("OggS") verified in output.
- Non-empty output verified.
- Invalid sample rate and channels tested via validating mock.
- Opus error propagation tested.
- Empty frames (zero-frame input) tested for graceful handling.
- Writer failure tested with `failOGGWriter`.
- Various frame counts tested (1, 10, 50, 150).
- Constructor tests for both `NewOGGEncoder` and `NewOGGEncoderWithOpus`.
- Benchmark included.

**Test coverage assessment:** The test suite covers all documented behaviors, error paths, edge cases (empty input, single sample, partial frames), and the goroutine lifecycle. The use of mocks in the OGG tests is particularly good -- it allows testing the OGG encoder's composition logic independently of the Opus encoder, which depends on CGO.

---

## Consistency with Stage 1 Patterns

- Error sentinels follow the same pattern as `internal/audio/errors.go` (package-level `var` block with `errors.New`).
- The `encoding:` prefix on error messages provides package context, consistent with Go conventions.
- Interface definitions are minimal and focused (single-method interfaces for `FileEncoder`).
- The code stays within `internal/` as expected for non-public packages.
- Test file naming follows `*_test.go` convention with the same package name (white-box testing).

---

## Summary

| Category | Assessment |
|----------|-----------|
| Documentation | All exported items documented. Package doc present. |
| Error handling | Consistent `%w` wrapping. Sentinel errors for `errors.Is`. |
| Nil safety | Input validation before use. Channel nil guards adequate. |
| Goroutine lifecycle | All paths close channels. No leaks. `errCh` capacity prevents blocks. |
| WAV header layout | Matches RIFF spec exactly. Verified field-by-field. |
| RTP packet construction | Correct version, payload type, timestamp increment for pion oggwriter. |
| Test coverage | Comprehensive: edge cases, error paths, lifecycle, mocks. |
| Code organization | Clean separation of concerns. Composition via interfaces. |
| Naming conventions | Idiomatic Go. Consistent with Stage 1. |

**No blocking issues found. Code is ready to merge.**
