# Design Review: Stage 2 -- Audio Encoding

**Verdict: REQUEST_CHANGES**

## Summary

The encoding package is well-structured overall. The interface design is clean, decoupling is properly maintained, error handling follows idiomatic patterns, and the test suite is thorough. However, there are several issues ranging from a correctness bug to design inconsistencies and minor documentation gaps that should be addressed before merge.

---

## Critical Issues

### 1. `errCh` is never closed -- violates interface contract

**Files:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (line 47), `/Users/jamesprial/code/go-scream/internal/encoding/opus.go`

The `OpusFrameEncoder` interface documentation states:

> Both channels are closed after completion.

However, `errCh` is never closed in any code path of `GopusFrameEncoder.EncodeFrames`. Only `frameCh` is closed (via `defer close(frameCh)` on line 62, and explicitly on lines 48 and 55 in the validation error goroutines).

Since `errCh` is a buffered channel of capacity 1, consumers that use `range errCh` or a second `<-errCh` read will block forever. The current consumers happen to work because they only read one value, but this is a contract violation that will cause subtle bugs if any future consumer trusts the documented behavior.

**Fix:** Add `defer close(errCh)` in each goroutine, or change the interface documentation to state that exactly one value is sent on the error channel (and it is not closed). The former is preferred since it matches the documented contract and enables `range errCh` usage.

### 2. Error wrapping inconsistency: `%v` instead of `%w` for inner errors

**Files:** `/Users/jamesprial/code/go-scream/internal/encoding/wav.go` (line 49), `/Users/jamesprial/code/go-scream/internal/encoding/opus.go` (lines 65, 95, 103, 110), `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go` (line 41, 77)

Throughout the package, the pattern for wrapping errors uses `%w` for the sentinel error but `%v` for the underlying cause:

```go
return fmt.Errorf("%w: reading PCM data: %v", ErrWAVWrite, err)
//                                         ^^ should be %w
```

This means the underlying cause error cannot be extracted with `errors.Is` or `errors.Unwrap`. Only the sentinel is unwrappable. While this may be intentional (to avoid double-wrapping ambiguity with Go's single-chain `%w`), it should be a deliberate, documented choice. If callers ever need to inspect the root cause (e.g., distinguish `io.ErrClosedPipe` from `io.EOF`), they cannot.

**Recommendation:** Either use `%w` for the inner error and document that multiple wrapping is intended, or add a comment explaining why `%v` is used for the inner error. Go 1.20+ supports multiple `%w` verbs in a single `fmt.Errorf`, so `fmt.Errorf("%w: reading PCM data: %w", ErrWAVWrite, err)` would allow `errors.Is` to match either.

---

## Design Issues

### 3. WAV encoder reads all PCM data into memory

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/wav.go` (line 46)

```go
pcmData, err := io.ReadAll(src)
```

`WAVEncoder.Encode` calls `io.ReadAll(src)` to buffer the entire PCM payload before writing the header. This is necessary because the WAV header requires the data size upfront. For a 3-second stereo 48kHz scream (~576KB), this is fine. However:

- The `FileEncoder` interface accepts `io.Reader` and `io.Writer`, which suggests streaming semantics.
- If this bot ever produces longer audio or is reused in other contexts, this becomes a memory concern.
- There is no documented maximum input size or memory warning.

**Recommendation:** Add a comment on the `Encode` method (or the `FileEncoder` interface) noting that WAV encoding requires buffering the full PCM payload due to the header size field. This sets expectations for callers. No code change is strictly necessary for current use cases.

### 4. `OGGEncoder` does not validate nil `opus` field

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go` (line 23-25)

```go
func NewOGGEncoderWithOpus(opus OpusFrameEncoder) *OGGEncoder {
    return &OGGEncoder{opus: opus}
}
```

If `nil` is passed, `Encode` will panic at line 32 when calling `e.opus.EncodeFrames(...)`. Consistent with Go conventions, either:
- Add a nil guard in the constructor (return an error or panic with a clear message), or
- Document that `opus` must not be nil.

The `internal/audio` package (Stage 1) does not face this issue because `NativeGenerator` has no injected dependencies, but as the encoding package introduces dependency injection via the `OpusFrameEncoder` interface, nil safety becomes relevant.

### 5. Inconsistent error sentinel naming with Stage 1

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (lines 27-30) vs `/Users/jamesprial/code/go-scream/internal/audio/errors.go` (lines 9-11)

Stage 1 (`audio` package):
```go
ErrInvalidSampleRate = errors.New("sample rate must be positive")
ErrInvalidChannels   = errors.New("channels must be 1 or 2")
```

Stage 2 (`encoding` package):
```go
ErrInvalidSampleRate = errors.New("encoding: sample rate must be positive")
ErrInvalidChannels   = errors.New("encoding: channels must be 1 or 2")
```

The Stage 2 errors include a package prefix (`encoding:`) in the message, which Stage 1 does not. This inconsistency means that identical validation failures produce different error messages depending on which package catches them. Both conventions are valid in Go, but they should be consistent across the project.

**Recommendation:** Either add the `audio:` prefix to Stage 1 errors (in a future cleanup) or remove the `encoding:` prefix from Stage 2 errors. Alternatively, consider sharing the validation sentinel errors in a common package (though this may be premature for a 2-package project).

---

## Minor Issues

### 6. Buffer zeroing could use `clear()` builtin

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/opus.go` (lines 81-83)

```go
for i := range pcmBuf {
    pcmBuf[i] = 0
}
```

Go 1.21+ provides the `clear()` builtin which zeros slices. Since the go.mod specifies Go 1.25.7, this is available and more idiomatic:

```go
clear(pcmBuf)
```

### 7. `pcmBytesToInt16` odd-byte behavior is underdocumented

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (line 57)

The doc comment says "Any trailing odd byte is ignored", but there is no test case for an odd-length input. This is an edge case that should be tested to confirm the documented behavior.

### 8. Magic numbers in OGG encoder

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go` (lines 54-58)

```go
Version:        2,
PayloadType:    111,
...
SSRC:           1,
```

`PayloadType: 111` and `SSRC: 1` are magic numbers. These should be named constants with documentation explaining their meaning (111 is a common dynamic payload type for Opus in WebRTC, and SSRC is an arbitrary stream identifier for the OGG container).

### 9. Test for `NewGopusFrameEncoderWithBitrate` does not verify bitrate is stored

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/opus_test.go` (lines 352-358)

The test only checks that the return value is non-nil. It does not verify that the provided bitrate is actually used. Since `bitrate` is an unexported field, this is difficult to test directly, but an integration test comparing frame sizes at different bitrates (e.g., 16kbps vs 128kbps) would add confidence.

---

## Positive Observations

### Decoupling
The encoding package has zero imports from `internal/audio`. The coupling point is the `io.Reader` of PCM data, which is exactly the right abstraction boundary. This was a key design requirement and it is met cleanly.

### Interface Design
The two-interface split (`OpusFrameEncoder` for streaming frames, `FileEncoder` for container output) is well-considered:
- `OpusFrameEncoder` returns channels, which matches the Discord voice use case (send frames as they are produced).
- `FileEncoder` uses `io.Writer`/`io.Reader`, which is idiomatic for container-format output.
- `OGGEncoder` composes `OpusFrameEncoder`, demonstrating clean layering.

### Extensibility
New encoders (e.g., FLAC, MP3) can implement `FileEncoder` without touching existing code. New frame encoders can implement `OpusFrameEncoder` (despite the name suggesting Opus, the interface is generic enough for any framed codec). The `NewOGGEncoderWithOpus` constructor enables dependency injection for testing, which is demonstrated effectively in `ogg_test.go`.

### Test Quality
- Compile-time interface checks (`var _ FileEncoder = (*WAVEncoder)(nil)`) are present for all concrete types.
- Table-driven tests are used consistently and cover edge cases (empty input, single sample, partial frames, invalid parameters).
- Mock implementations in `ogg_test.go` are well-crafted and avoid CGO dependencies for unit testing.
- The `skipIfNoOpus` pattern handles environments without libopus gracefully.
- Test helpers use `t.Helper()` correctly.
- Benchmarks are included.

### Error Handling
- Sentinel errors with `%w` wrapping enable `errors.Is` matching.
- Error messages include context (parameter values, operation descriptions).
- The OGG encoder correctly drains the frame channel on error to prevent goroutine leaks (lines 37-39).

### Documentation
- All exported types, functions, and constants have doc comments.
- The package-level doc comment accurately describes the package scope.
- Method documentation describes valid parameter ranges and error semantics.

---

## Required Changes (for APPROVE)

1. **Fix `errCh` lifecycle** to match the interface contract ("Both channels are closed after completion"). Add `defer close(errCh)` or update the contract.
2. **Add nil guard** for the `opus` parameter in `NewOGGEncoderWithOpus`, or document that nil panics.
3. **Add a test case** for odd-length byte slice input to `pcmBytesToInt16`.

## Recommended Changes (non-blocking)

4. Decide on a consistent error message prefix convention across `audio` and `encoding`.
5. Replace manual buffer zeroing with `clear(pcmBuf)`.
6. Extract magic numbers in the OGG encoder to named constants.
7. Consider using multiple `%w` verbs (Go 1.20+) for full error chain unwrapping.
8. Add a comment to `WAVEncoder.Encode` noting that the full PCM payload is buffered.
