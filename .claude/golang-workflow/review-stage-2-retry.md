# Code Review: Stage 2 Retry -- Audio Encoding

**Reviewer:** Go Code Reviewer (Opus 4.6)
**Date:** 2026-02-18
**Files Reviewed:**
- `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/wav.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/opus.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/encoder_test.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/wav_test.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/opus_test.go`
- `/Users/jamesprial/code/go-scream/internal/encoding/ogg_test.go`

---

## Fix Verification

All 5 fixes from the design review have been correctly applied:

### Fix 1: errCh closed in all goroutine paths (opus.go)

**Status: VERIFIED**

Three `close(errCh)` calls cover every goroutine exit path:
- Line 50: validation error path for invalid sample rate
- Line 58: validation error path for invalid channels
- Line 65: `defer close(errCh)` in the main encoding goroutine

Both early-exit validation goroutines close `frameCh` before sending the error on `errCh`, then close `errCh`. The main goroutine uses `defer close(frameCh)` and `defer close(errCh)`, which correctly covers all exits (normal completion, partial frame, and error paths). The contract that both channels are always closed is fully satisfied.

### Fix 2: Error wrapping uses %w for inner errors

**Status: VERIFIED**

All `fmt.Errorf` calls across `wav.go`, `opus.go`, and `ogg.go` use `%w` for wrapping sentinel errors and inner errors. Multiple `%w` verbs are used where both a sentinel and an inner error are wrapped (e.g., `fmt.Errorf("%w: reading PCM data: %w", ErrWAVWrite, err)` at `wav.go:48`). This is valid Go 1.20+ syntax and the project targets Go 1.25.7 per `go.mod`. The test files verify `errors.Is` works for all sentinel error types.

### Fix 3: NewOGGEncoderWithOpus has nil guard with panic

**Status: VERIFIED**

At `ogg.go:24-28`, `NewOGGEncoderWithOpus` checks `opus == nil` and panics with a clear message `"encoding: opus encoder must not be nil"`. The doc comment on line 23 documents the panic behavior. This follows the Go convention of panicking on programmer errors in constructors (cf. `regexp.MustCompile`).

### Fix 4: clear(pcmBuf) replaces manual zeroing loop

**Status: VERIFIED**

At `opus.go:84`, `clear(pcmBuf)` is used to zero the buffer before each `io.ReadFull` call. The `clear` builtin is available since Go 1.21 and the project targets Go 1.25.7. This is cleaner and likely more efficient than a manual loop.

### Fix 5: Magic numbers extracted to named constants

**Status: VERIFIED**

At `encoder.go:24-32`, `oggPayloadType = 111` and `oggSSRC = 1` are defined as named constants with documentation. They are used at `ogg.go:59` and `ogg.go:62` respectively. The constants are unexported (lowercase) which is correct since they are internal implementation details.

---

## Code Quality Assessment

### Error Handling

All error returns consistently wrap sentinel errors using `%w`, enabling callers to use `errors.Is` for programmatic error discrimination. The error hierarchy is:
- `ErrInvalidSampleRate` / `ErrInvalidChannels` for validation failures
- `ErrOpusEncode` for encoding failures
- `ErrWAVWrite` / `ErrOGGWrite` for output failures

The `OGGEncoder.Encode` method (ogg.go) correctly prioritizes OGG write errors over opus errors (line 80), and propagates opus errors directly without re-wrapping (line 87-88), preserving the error chain from the opus encoder. This is a good design choice documented by the comment on lines 84-86.

### Nil Safety

- `pcmBytesToInt16` handles nil/empty input (encoder.go:70-72)
- `NewOGGEncoderWithOpus` panics on nil opus (ogg.go:25-27)
- The `OGGEncoder.Encode` method drains channels on early `oggwriter.NewWith` failure (ogg.go:41-43) to prevent goroutine leaks
- OGG write errors are recorded but frame draining continues (ogg.go:67-73) to avoid blocking the producer goroutine

### Concurrency Correctness

The opus encoder goroutine lifecycle is sound:
- Both channels are always closed on all paths
- `frameCh` has buffer capacity 2, preventing unnecessary blocking
- `errCh` has buffer capacity 1, allowing the goroutine to send without waiting for a receiver
- The OGG encoder drains `frameCh` via `range` and then reads from `errCh`, which is the correct ordering to avoid deadlock

### Documentation

All exported types, functions, constants, and interfaces have doc comments. The `OpusFrameEncoder` interface documents the channel semantics. The sentinel errors document their usage contexts. The `EncodeFrames` method documents valid sample rates.

### Test Coverage

Tests are well-structured using table-driven patterns:
- Compile-time interface checks for `WAVEncoder`, `GopusFrameEncoder`, and `OGGEncoder`
- Validation error paths tested for both WAV and Opus encoders
- Channel lifecycle tests verify both channels are closed on success and error paths
- OGG tests use mock `OpusFrameEncoder` implementations to test the OGG layer in isolation
- Edge cases covered: empty input, single sample, partial frames, writer failures

### Minor Observations (non-blocking)

1. The mock encoders in `ogg_test.go` (`mockOpusEncoder`, `mockOpusEncoderValidating`) do not close `errCh`, which is a slight deviation from the `OpusFrameEncoder` contract. However, since `errCh` is buffered with capacity 1 and the OGG encoder reads from it with `<-errCh` (not `range`), this does not cause any issues in practice. The mocks still correctly simulate the behavior needed for testing.

2. There is no test that exercises the nil-guard panic in `NewOGGEncoderWithOpus`. A test using `defer func() { recover() }()` to verify the panic message would improve coverage of the fix. This is a minor gap.

3. `TestGopusFrameEncoder_ChannelsClosed` (opus_test.go:310-313) has a bare `select` with a single case and no `default` -- this is equivalent to a plain receive. It works but reads slightly oddly. Not a correctness issue.

---

## Verdict

**APPROVE**

All 5 design review fixes have been correctly applied. The code follows Go idioms consistently: proper error wrapping with sentinels, clean channel lifecycle management, good documentation, and thorough table-driven tests. No correctness issues or blocking concerns found.
