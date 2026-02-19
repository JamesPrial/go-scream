# Design Review (Retry): Stage 2 -- Audio Encoding

**Verdict: APPROVE**

## Fix Verification

All 5 previously requested changes have been applied correctly and completely.

### Fix 1: errCh never closed in EncodeFrames -- VERIFIED

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/opus.go`

The error channel is now closed in all three goroutine paths:

- **Validation error paths (lines 48-51 and 55-58):** Each validation failure goroutine explicitly calls `close(errCh)` after sending the error.
- **Main encoding goroutine (line 65):** Uses `defer close(errCh)` at the top of the goroutine, ensuring closure on all exit paths (success, encoding error, read error).

This matches the `OpusFrameEncoder` interface contract: "Both channels are closed after completion."

### Fix 2: Error wrapping uses %w instead of %v for inner errors -- VERIFIED

**Files:** All `.go` files in `/Users/jamesprial/code/go-scream/internal/encoding/`

A search for `fmt.Errorf.*%v` in the non-test source files returns zero matches. All error wrapping now consistently uses `%w` for both the sentinel error and the underlying cause. Examples:

```go
// wav.go:48
return fmt.Errorf("%w: reading PCM data: %w", ErrWAVWrite, err)

// opus.go:69
errCh <- fmt.Errorf("%w: creating encoder: %w", ErrOpusEncode, err)

// ogg.go:44
return fmt.Errorf("%w: creating OGG writer: %w", ErrOGGWrite, err)
```

This enables full error chain inspection via `errors.Is` and `errors.Unwrap` for both the sentinel and root cause errors.

### Fix 3: No nil guard in NewOGGEncoderWithOpus -- VERIFIED

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/ogg.go` (lines 24-28)

```go
// NewOGGEncoderWithOpus returns an OGGEncoder that uses the provided OpusFrameEncoder.
// It panics if opus is nil.
func NewOGGEncoderWithOpus(opus OpusFrameEncoder) *OGGEncoder {
    if opus == nil {
        panic("encoding: opus encoder must not be nil")
    }
    return &OGGEncoder{opus: opus}
}
```

The nil guard is present with a clear panic message, and the doc comment documents the panic behavior. This follows the Go convention of panicking on programmer errors in constructors (consistent with patterns like `regexp.MustCompile`).

### Fix 4: Use clear(pcmBuf) instead of manual zeroing -- VERIFIED

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/opus.go` (line 84)

```go
clear(pcmBuf)
```

The manual `for i := range pcmBuf { pcmBuf[i] = 0 }` loop has been replaced with the `clear()` builtin. This is appropriate given the Go 1.25.7 target in `go.mod`.

### Fix 5: Magic numbers in OGG encoder extracted to constants -- VERIFIED

**File:** `/Users/jamesprial/code/go-scream/internal/encoding/encoder.go` (lines 23-32)

```go
// OGG/RTP constants used when writing Opus frames into an OGG container.
const (
    // oggPayloadType is the RTP payload type for Opus audio as registered
    // with IANA (dynamic payload type 111).
    oggPayloadType = 111

    // oggSSRC is the fixed RTP synchronisation source identifier used for
    // single-stream OGG output.
    oggSSRC = 1
)
```

The constants are unexported (correct, since they are implementation details), well-documented with explanations of what the values represent, and used consistently in `ogg.go` at lines 59-60.

---

## Design Assessment

### Package Organization

The encoding package is cleanly organized across four files:

- `encoder.go` -- shared types, interfaces, constants, sentinel errors, and the `pcmBytesToInt16` utility
- `wav.go` -- WAV container encoder
- `opus.go` -- Opus frame encoder (CGO-dependent)
- `ogg.go` -- OGG/Opus container encoder (composes OpusFrameEncoder)

This separation follows the single-responsibility principle. Each file is focused and reasonably sized (80-122 lines for implementation files).

### Interface Design

The two-interface split remains well-considered:

- **`OpusFrameEncoder`** returns channels for streaming frame-by-frame consumption (matching Discord voice send patterns).
- **`FileEncoder`** uses `io.Writer`/`io.Reader` for container-format output (idiomatic Go I/O patterns).
- **`OGGEncoder`** composes `OpusFrameEncoder` via dependency injection, enabling the mock-based testing demonstrated in `ogg_test.go`.

### Decoupling

The encoding package has zero imports from `internal/audio`. The coupling point is `io.Reader` of raw PCM data, which is the correct abstraction boundary for this architecture.

### Error Handling

Error handling is now fully consistent:

- Sentinel errors (`ErrInvalidSampleRate`, `ErrInvalidChannels`, `ErrOpusEncode`, `ErrWAVWrite`, `ErrOGGWrite`) enable `errors.Is` matching.
- All `fmt.Errorf` calls use `%w` for both the sentinel and the underlying cause, enabling full error chain inspection.
- Error messages include contextual information (parameter values, operation descriptions).
- The OGG encoder correctly drains the frame channel on writer creation failure (lines 41-43 of `ogg.go`) to prevent goroutine leaks.

### Documentation

All exported types, functions, constants, and interfaces have doc comments. The package-level doc comment accurately describes the package scope. Method documentation describes valid parameter ranges, error semantics, and panic conditions.

### Naming Conventions

- Exported types and functions follow Go conventions (`WAVEncoder`, `NewGopusFrameEncoder`, `OpusFrameSamples`).
- Unexported constants use camelCase (`oggPayloadType`, `oggSSRC`).
- The `wavFileHeader` struct uses descriptive field names matching the WAV specification.
- Sentinel error names follow the `Err` prefix convention.

### Test Quality

- Compile-time interface checks are present for all concrete types.
- Table-driven tests cover edge cases (empty input, single sample, partial frames, invalid parameters, boundary values).
- Mock implementations in `ogg_test.go` effectively decouple OGG container testing from CGO/libopus.
- The `skipIfNoOpus` pattern handles environments without libopus gracefully.
- Test helpers use `t.Helper()` correctly.
- Benchmarks are included for all encoders.
- Channel lifecycle tests verify both channels are closed on success and error paths.

### Consistency with Existing Codebase

The encoding package follows patterns established in `internal/audio`:

- Sentinel error variables in a `var` block (matching `audio/errors.go`).
- Constructor functions returning pointer types (`NewWAVEncoder`, `NewGopusFrameEncoder`, matching `NewNativeGenerator`).
- Validation at the top of public methods with early error returns.
- The only inconsistency is the `encoding:` prefix in error messages vs no prefix in `audio` errors. This was noted in the prior review as a non-blocking recommendation and remains a minor style difference that does not affect correctness.

---

## Remaining Non-Blocking Observations

These items were noted in the prior review and are not required for approval:

1. **Mock implementations in `ogg_test.go` do not close `errCh`.** The `mockOpusEncoder` and `mockOpusEncoderValidating` test helpers send a value on `errCh` but never close it. Since the OGG encoder only reads one value from `errCh` (line 77 of `ogg.go`), this works correctly, but it does not match the interface contract. Adding `defer close(errCh)` to the mock goroutines would be a minor improvement.

2. **No test case for odd-byte-length input to `pcmBytesToInt16`.** The doc comment states "Any trailing odd byte is ignored" but no test verifies this behavior. A single test case with a 3-byte or 5-byte input would document the edge case.

3. **Error message prefix inconsistency** between `audio` and `encoding` packages remains a minor style difference.

Neither of these items blocks merge.
