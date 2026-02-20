package discord

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// mockVoiceConn implements VoiceConn for testing. It captures all interactions
// so tests can verify the protocol was followed correctly.
type mockVoiceConn struct {
	mu            sync.Mutex
	sentFrames    [][]byte
	speakingCalls []bool
	disconnected  bool
	opusSend      chan []byte
	speakingErr   error // injected error for Speaking()
	disconnectErr error // injected error for Disconnect()

	// collector goroutine lifecycle
	collectDone chan struct{}
}

func newMockVoiceConn() *mockVoiceConn {
	m := &mockVoiceConn{
		opusSend:    make(chan []byte, 256),
		collectDone: make(chan struct{}),
	}
	// Start a goroutine to collect frames sent to the opus channel.
	// This goroutine runs until the opusSend channel is closed via drainAndClose.
	go func() {
		defer close(m.collectDone)
		for frame := range m.opusSend {
			cp := make([]byte, len(frame))
			copy(cp, frame)
			m.mu.Lock()
			m.sentFrames = append(m.sentFrames, cp)
			m.mu.Unlock()
		}
	}()
	return m
}

func (m *mockVoiceConn) Speaking(speaking bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.speakingCalls = append(m.speakingCalls, speaking)
	return m.speakingErr
}

func (m *mockVoiceConn) OpusSendChannel() chan<- []byte {
	return m.opusSend
}

func (m *mockVoiceConn) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnected = true
	return m.disconnectErr
}

// drainAndCollect closes the opusSend channel (simulating cleanup after Play
// returns) and waits for the collector goroutine to drain all buffered frames.
// This must be called AFTER Play returns. Returns the collected frames.
func (m *mockVoiceConn) drainAndCollect(t *testing.T) [][]byte {
	t.Helper()
	// Close the channel so the collector goroutine exits after draining.
	close(m.opusSend)
	select {
	case <-m.collectDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for frame collection to complete")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([][]byte, len(m.sentFrames))
	copy(out, m.sentFrames)
	return out
}

// mockSession implements Session for testing.
type mockSession struct {
	mu          sync.Mutex
	voiceConn   *mockVoiceConn
	joinErr     error
	joinCalls   []joinCall
	voiceStates []*VoiceState
	stateErr    error
}

type joinCall struct {
	guildID   string
	channelID string
	mute      bool
	deaf      bool
}

func (m *mockSession) ChannelVoiceJoin(guildID, channelID string, mute, deaf bool) (VoiceConn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.joinCalls = append(m.joinCalls, joinCall{guildID, channelID, mute, deaf})
	if m.joinErr != nil {
		return nil, m.joinErr
	}
	return m.voiceConn, nil
}

func (m *mockSession) GuildVoiceStates(guildID string) ([]*VoiceState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stateErr != nil {
		return nil, m.stateErr
	}
	return m.voiceStates, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeFrames creates a channel pre-loaded with n distinct frames, then closes it.
func makeFrames(n int) <-chan []byte {
	ch := make(chan []byte, n)
	for i := 0; i < n; i++ {
		ch <- []byte{byte(i), byte(i + 1), byte(i + 2)}
	}
	close(ch)
	return ch
}

// setupPlayer creates a Player with a mock session and voice connection.
func setupPlayer() (*Player, *mockSession, *mockVoiceConn) {
	vc := newMockVoiceConn()
	sess := &mockSession{voiceConn: vc}
	player := NewPlayer(sess, discardLogger)
	return player, sess, vc
}

// ---------------------------------------------------------------------------
// Compile-time interface check
// ---------------------------------------------------------------------------

var _ VoicePlayer = (*Player)(nil)

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewPlayer_NotNil(t *testing.T) {
	sess := &mockSession{}
	player := NewPlayer(sess, discardLogger)
	if player == nil {
		t.Fatal("NewPlayer() returned nil")
	}
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestSilenceFrame_Content(t *testing.T) {
	want := []byte{0xF8, 0xFF, 0xFE}
	if !bytes.Equal(silenceFrame, want) {
		t.Errorf("silenceFrame = %v, want %v", silenceFrame, want)
	}
}

func TestSilenceFrameCount_Value(t *testing.T) {
	if silenceFrameCount != 5 {
		t.Errorf("silenceFrameCount = %d, want 5", silenceFrameCount)
	}
}

// ---------------------------------------------------------------------------
// Normal playback
// ---------------------------------------------------------------------------

func TestPlayer_Play_10Frames(t *testing.T) {
	player, _, vc := setupPlayer()
	frames := makeFrames(10)

	err := player.Play(context.Background(), "g1", "c1", frames)
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	sent := vc.drainAndCollect(t)

	// Expect 10 data frames + 5 silence frames = 15 total.
	if len(sent) != 15 {
		t.Fatalf("sent %d frames, want 15 (10 data + 5 silence)", len(sent))
	}

	// Verify the first 10 are the data frames.
	for i := 0; i < 10; i++ {
		want := []byte{byte(i), byte(i + 1), byte(i + 2)}
		if !bytes.Equal(sent[i], want) {
			t.Errorf("frame[%d] = %v, want %v", i, sent[i], want)
		}
	}

	// Verify the last 5 are silence frames.
	silence := []byte{0xF8, 0xFF, 0xFE}
	for i := 10; i < 15; i++ {
		if !bytes.Equal(sent[i], silence) {
			t.Errorf("frame[%d] = %v, want silence %v", i, sent[i], silence)
		}
	}

	// Speaking protocol.
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if len(vc.speakingCalls) < 2 {
		t.Fatalf("speaking calls = %d, want at least 2", len(vc.speakingCalls))
	}
	if vc.speakingCalls[0] != true {
		t.Errorf("first Speaking() call = %v, want true", vc.speakingCalls[0])
	}
	if vc.speakingCalls[len(vc.speakingCalls)-1] != false {
		t.Errorf("last Speaking() call = %v, want false", vc.speakingCalls[len(vc.speakingCalls)-1])
	}

	// Disconnect called.
	if !vc.disconnected {
		t.Error("Disconnect() was not called")
	}
}

func TestPlayer_Play_EmptyChannel(t *testing.T) {
	player, _, vc := setupPlayer()
	ch := make(chan []byte)
	close(ch) // closed immediately, 0 data frames

	err := player.Play(context.Background(), "g1", "c1", ch)
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	sent := vc.drainAndCollect(t)

	// Expect exactly 5 silence frames, 0 data frames.
	if len(sent) != 5 {
		t.Fatalf("sent %d frames, want 5 (0 data + 5 silence)", len(sent))
	}

	silence := []byte{0xF8, 0xFF, 0xFE}
	for i, frame := range sent {
		if !bytes.Equal(frame, silence) {
			t.Errorf("frame[%d] = %v, want silence %v", i, frame, silence)
		}
	}

	// Full protocol still observed.
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if len(vc.speakingCalls) < 2 {
		t.Fatalf("speaking calls = %d, want at least 2", len(vc.speakingCalls))
	}
	if vc.speakingCalls[0] != true {
		t.Errorf("first Speaking() call = %v, want true", vc.speakingCalls[0])
	}
	if vc.speakingCalls[len(vc.speakingCalls)-1] != false {
		t.Errorf("last Speaking() call = %v, want false", vc.speakingCalls[len(vc.speakingCalls)-1])
	}
	if !vc.disconnected {
		t.Error("Disconnect() was not called")
	}
}

func TestPlayer_Play_1Frame(t *testing.T) {
	player, _, vc := setupPlayer()
	frames := makeFrames(1)

	err := player.Play(context.Background(), "g1", "c1", frames)
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	sent := vc.drainAndCollect(t)

	// 1 data + 5 silence = 6 total.
	if len(sent) != 6 {
		t.Fatalf("sent %d frames, want 6 (1 data + 5 silence)", len(sent))
	}

	want := []byte{0, 1, 2}
	if !bytes.Equal(sent[0], want) {
		t.Errorf("frame[0] = %v, want %v", sent[0], want)
	}

	silence := []byte{0xF8, 0xFF, 0xFE}
	for i := 1; i < 6; i++ {
		if !bytes.Equal(sent[i], silence) {
			t.Errorf("frame[%d] = %v, want silence %v", i, sent[i], silence)
		}
	}

	vc.mu.Lock()
	defer vc.mu.Unlock()
	if !vc.disconnected {
		t.Error("Disconnect() was not called")
	}
}

// ---------------------------------------------------------------------------
// Speaking protocol
// ---------------------------------------------------------------------------

func TestPlayer_Play_SpeakingProtocol(t *testing.T) {
	player, _, vc := setupPlayer()
	frames := makeFrames(3)

	err := player.Play(context.Background(), "g1", "c1", frames)
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	// Drain to ensure all frames collected before inspecting state.
	vc.drainAndCollect(t)

	vc.mu.Lock()
	defer vc.mu.Unlock()

	if len(vc.speakingCalls) != 2 {
		t.Fatalf("Speaking() called %d times, want exactly 2", len(vc.speakingCalls))
	}
	if vc.speakingCalls[0] != true {
		t.Errorf("Speaking() call[0] = %v, want true", vc.speakingCalls[0])
	}
	if vc.speakingCalls[1] != false {
		t.Errorf("Speaking() call[1] = %v, want false", vc.speakingCalls[1])
	}
}

// ---------------------------------------------------------------------------
// Silence frames
// ---------------------------------------------------------------------------

func TestPlayer_Play_silenceFrames(t *testing.T) {
	player, _, vc := setupPlayer()
	frames := makeFrames(2)

	err := player.Play(context.Background(), "g1", "c1", frames)
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	sent := vc.drainAndCollect(t)

	// 2 data + 5 silence = 7
	if len(sent) != 7 {
		t.Fatalf("sent %d frames, want 7", len(sent))
	}

	silence := []byte{0xF8, 0xFF, 0xFE}
	silenceCount := 0
	for i := 2; i < len(sent); i++ {
		if bytes.Equal(sent[i], silence) {
			silenceCount++
		} else {
			t.Errorf("frame[%d] = %v, want silence %v", i, sent[i], silence)
		}
	}
	if silenceCount != 5 {
		t.Errorf("silence frame count = %d, want exactly 5", silenceCount)
	}
}

// ---------------------------------------------------------------------------
// Disconnect
// ---------------------------------------------------------------------------

func TestPlayer_Play_DisconnectCalled(t *testing.T) {
	player, _, vc := setupPlayer()
	frames := makeFrames(1)

	err := player.Play(context.Background(), "g1", "c1", frames)
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}

	// Play has returned, so Disconnect (via defer) has already been called.
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if !vc.disconnected {
		t.Error("Disconnect() was not called on normal completion")
	}
}

func TestPlayer_Play_DisconnectOnError(t *testing.T) {
	// Speaking(true) fails, but Disconnect should still be called
	// because join succeeded and defer should fire.
	vc := newMockVoiceConn()
	vc.speakingErr = errors.New("mock speaking error")
	sess := &mockSession{voiceConn: vc}
	player := NewPlayer(sess, discardLogger)

	frames := makeFrames(1)
	err := player.Play(context.Background(), "g1", "c1", frames)
	if err == nil {
		t.Fatal("Play() expected error when Speaking fails, got nil")
	}

	// Play has returned, so Disconnect (via defer) has already been called.
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if !vc.disconnected {
		t.Error("Disconnect() was not called when Speaking(true) failed")
	}
}

// ---------------------------------------------------------------------------
// Join parameters
// ---------------------------------------------------------------------------

func TestPlayer_Play_JoinParams(t *testing.T) {
	player, sess, vc := setupPlayer()
	frames := makeFrames(1)

	err := player.Play(context.Background(), "guild-abc", "channel-xyz", frames)
	if err != nil {
		t.Fatalf("Play() unexpected error: %v", err)
	}
	vc.drainAndCollect(t)

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if len(sess.joinCalls) != 1 {
		t.Fatalf("ChannelVoiceJoin called %d times, want 1", len(sess.joinCalls))
	}
	call := sess.joinCalls[0]
	if call.guildID != "guild-abc" {
		t.Errorf("join guildID = %q, want %q", call.guildID, "guild-abc")
	}
	if call.channelID != "channel-xyz" {
		t.Errorf("join channelID = %q, want %q", call.channelID, "channel-xyz")
	}
	if call.mute != false {
		t.Errorf("join mute = %v, want false", call.mute)
	}
	if call.deaf != true {
		t.Errorf("join deaf = %v, want true", call.deaf)
	}
}

// ---------------------------------------------------------------------------
// Error conditions
// ---------------------------------------------------------------------------

func TestPlayer_Play_EmptyGuildID(t *testing.T) {
	player, _, _ := setupPlayer()
	frames := makeFrames(1)

	err := player.Play(context.Background(), "", "c1", frames)
	if !errors.Is(err, ErrEmptyGuildID) {
		t.Errorf("Play() error = %v, want ErrEmptyGuildID", err)
	}
}

func TestPlayer_Play_EmptyChannelID(t *testing.T) {
	player, _, _ := setupPlayer()
	frames := makeFrames(1)

	err := player.Play(context.Background(), "g1", "", frames)
	if !errors.Is(err, ErrEmptyChannelID) {
		t.Errorf("Play() error = %v, want ErrEmptyChannelID", err)
	}
}

func TestPlayer_Play_NilFrames(t *testing.T) {
	player, _, _ := setupPlayer()

	err := player.Play(context.Background(), "g1", "c1", nil)
	if !errors.Is(err, ErrNilFrameChannel) {
		t.Errorf("Play() error = %v, want ErrNilFrameChannel", err)
	}
}

func TestPlayer_Play_JoinFails(t *testing.T) {
	sess := &mockSession{joinErr: errors.New("underlying join failure")}
	player := NewPlayer(sess, discardLogger)
	frames := makeFrames(1)

	err := player.Play(context.Background(), "g1", "c1", frames)
	if !errors.Is(err, ErrVoiceJoinFailed) {
		t.Errorf("Play() error = %v, want ErrVoiceJoinFailed", err)
	}
}

func TestPlayer_Play_SpeakingTrueFails(t *testing.T) {
	vc := newMockVoiceConn()
	vc.speakingErr = errors.New("speaking failure")
	sess := &mockSession{voiceConn: vc}
	player := NewPlayer(sess, discardLogger)
	frames := makeFrames(1)

	err := player.Play(context.Background(), "g1", "c1", frames)
	if !errors.Is(err, ErrSpeakingFailed) {
		t.Errorf("Play() error = %v, want ErrSpeakingFailed", err)
	}
}

func TestPlayer_Play_CancelledContext(t *testing.T) {
	player, _, _ := setupPlayer()
	frames := makeFrames(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before Play

	err := player.Play(ctx, "g1", "c1", frames)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Play() error = %v, want context.Canceled", err)
	}
}

// ---------------------------------------------------------------------------
// Context cancellation mid-playback
// ---------------------------------------------------------------------------

func TestPlayer_Play_CancelMidPlayback(t *testing.T) {
	vc := newMockVoiceConn()
	sess := &mockSession{voiceConn: vc}
	player := NewPlayer(sess, discardLogger)

	ctx, cancel := context.WithCancel(context.Background())

	// Use an unbuffered channel that we control so we can interleave sends
	// and cancellation precisely. We send a few frames, then cancel.
	frames := make(chan []byte, 100)
	frames <- []byte{0x01}
	frames <- []byte{0x02}
	frames <- []byte{0x03}

	// Cancel after a brief delay to allow some frames to be consumed.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
		// Close the channel after cancel so Play's loop can exit.
		close(frames)
	}()

	err := player.Play(ctx, "g1", "c1", frames)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Play() error = %v, want context.Canceled", err)
	}

	// Play has returned, so Disconnect (via defer) has already been called.
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if !vc.disconnected {
		t.Error("Disconnect() was not called after context cancellation")
	}
}

// ---------------------------------------------------------------------------
// Error validation table-driven (combined)
// ---------------------------------------------------------------------------

func TestPlayer_Play_ValidationErrors(t *testing.T) {
	validFrames := makeFrames(1)

	tests := []struct {
		name      string
		guildID   string
		channelID string
		frames    <-chan []byte
		wantErr   error
	}{
		{
			name:      "empty guild ID",
			guildID:   "",
			channelID: "c1",
			frames:    validFrames,
			wantErr:   ErrEmptyGuildID,
		},
		{
			name:      "empty channel ID",
			guildID:   "g1",
			channelID: "",
			frames:    validFrames,
			wantErr:   ErrEmptyChannelID,
		},
		{
			name:      "nil frame channel",
			guildID:   "g1",
			channelID: "c1",
			frames:    nil,
			wantErr:   ErrNilFrameChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh player per subtest to avoid shared mock state issues.
			player, _, _ := setupPlayer()
			err := player.Play(context.Background(), tt.guildID, tt.channelID, tt.frames)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Play() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkPlayer_Play_150Frames(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vc := newMockVoiceConn()
		sess := &mockSession{voiceConn: vc}
		player := NewPlayer(sess, discardLogger)

		ch := make(chan []byte, 150)
		for j := 0; j < 150; j++ {
			ch <- []byte{byte(j), byte(j + 1)}
		}
		close(ch)

		b.StartTimer()
		err := player.Play(context.Background(), "g1", "c1", ch)
		b.StopTimer()

		if err != nil {
			b.Fatalf("Play() unexpected error: %v", err)
		}
		// Close the opus channel so the collector goroutine can drain and exit.
		close(vc.opusSend)
		<-vc.collectDone
	}
}
