package discord

import (
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// FindPopulatedChannel tests
// ---------------------------------------------------------------------------

func TestFindPopulatedChannel_OneUser(t *testing.T) {
	sess := &mockSession{
		voiceStates: []*VoiceState{
			{UserID: "u1", ChannelID: "c1", GuildID: "g1"},
		},
	}

	got, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if err != nil {
		t.Fatalf("FindPopulatedChannel() unexpected error: %v", err)
	}
	if got != "c1" {
		t.Errorf("FindPopulatedChannel() = %q, want %q", got, "c1")
	}
}

func TestFindPopulatedChannel_MultipleUsers(t *testing.T) {
	sess := &mockSession{
		voiceStates: []*VoiceState{
			{UserID: "u1", ChannelID: "c1", GuildID: "g1"},
			{UserID: "u2", ChannelID: "c1", GuildID: "g1"},
		},
	}

	got, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if err != nil {
		t.Fatalf("FindPopulatedChannel() unexpected error: %v", err)
	}
	if got != "c1" {
		t.Errorf("FindPopulatedChannel() = %q, want %q", got, "c1")
	}
}

func TestFindPopulatedChannel_MultipleChannels(t *testing.T) {
	sess := &mockSession{
		voiceStates: []*VoiceState{
			{UserID: "u1", ChannelID: "c1", GuildID: "g1"},
			{UserID: "u2", ChannelID: "c2", GuildID: "g1"},
		},
	}

	got, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if err != nil {
		t.Fatalf("FindPopulatedChannel() unexpected error: %v", err)
	}
	// Should return the first channel with a non-bot user.
	if got != "c1" {
		t.Errorf("FindPopulatedChannel() = %q, want %q (first match)", got, "c1")
	}
}

func TestFindPopulatedChannel_OnlyBot(t *testing.T) {
	sess := &mockSession{
		voiceStates: []*VoiceState{
			{UserID: "bot", ChannelID: "c1", GuildID: "g1"},
		},
	}

	_, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if !errors.Is(err, ErrNoPopulatedChannel) {
		t.Errorf("FindPopulatedChannel() error = %v, want ErrNoPopulatedChannel", err)
	}
}

func TestFindPopulatedChannel_BotAndUser(t *testing.T) {
	sess := &mockSession{
		voiceStates: []*VoiceState{
			{UserID: "bot", ChannelID: "c1", GuildID: "g1"},
			{UserID: "u1", ChannelID: "c1", GuildID: "g1"},
		},
	}

	got, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if err != nil {
		t.Fatalf("FindPopulatedChannel() unexpected error: %v", err)
	}
	if got != "c1" {
		t.Errorf("FindPopulatedChannel() = %q, want %q", got, "c1")
	}
}

func TestFindPopulatedChannel_BotInOneUserInAnother(t *testing.T) {
	sess := &mockSession{
		voiceStates: []*VoiceState{
			{UserID: "bot", ChannelID: "c1", GuildID: "g1"},
			{UserID: "u1", ChannelID: "c2", GuildID: "g1"},
		},
	}

	got, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if err != nil {
		t.Fatalf("FindPopulatedChannel() unexpected error: %v", err)
	}
	if got != "c2" {
		t.Errorf("FindPopulatedChannel() = %q, want %q", got, "c2")
	}
}

func TestFindPopulatedChannel_Empty(t *testing.T) {
	sess := &mockSession{
		voiceStates: []*VoiceState{},
	}

	_, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if !errors.Is(err, ErrNoPopulatedChannel) {
		t.Errorf("FindPopulatedChannel() error = %v, want ErrNoPopulatedChannel", err)
	}
}

func TestFindPopulatedChannel_EmptyGuildID(t *testing.T) {
	sess := &mockSession{}

	_, err := FindPopulatedChannel(sess, "", "bot", discardLogger)
	if !errors.Is(err, ErrEmptyGuildID) {
		t.Errorf("FindPopulatedChannel() error = %v, want ErrEmptyGuildID", err)
	}
}

func TestFindPopulatedChannel_StateError(t *testing.T) {
	sess := &mockSession{
		stateErr: errors.New("underlying state error"),
	}

	_, err := FindPopulatedChannel(sess, "g1", "bot", discardLogger)
	if !errors.Is(err, ErrGuildStateFailed) {
		t.Errorf("FindPopulatedChannel() error = %v, want ErrGuildStateFailed", err)
	}
}

// ---------------------------------------------------------------------------
// Table-driven combined
// ---------------------------------------------------------------------------

func TestFindPopulatedChannel_Cases(t *testing.T) {
	tests := []struct {
		name        string
		voiceStates []*VoiceState
		stateErr    error
		guildID     string
		botUserID   string
		wantChannel string
		wantErr     error
	}{
		{
			name: "single non-bot user returns their channel",
			voiceStates: []*VoiceState{
				{UserID: "u1", ChannelID: "ch-a"},
			},
			guildID:     "g1",
			botUserID:   "bot",
			wantChannel: "ch-a",
		},
		{
			name: "bot user excluded, second user returned",
			voiceStates: []*VoiceState{
				{UserID: "bot", ChannelID: "ch-bot"},
				{UserID: "human", ChannelID: "ch-human"},
			},
			guildID:     "g1",
			botUserID:   "bot",
			wantChannel: "ch-human",
		},
		{
			name:        "no voice states returns ErrNoPopulatedChannel",
			voiceStates: []*VoiceState{},
			guildID:     "g1",
			botUserID:   "bot",
			wantErr:     ErrNoPopulatedChannel,
		},
		{
			name:      "empty guild ID returns ErrEmptyGuildID",
			guildID:   "",
			botUserID: "bot",
			wantErr:   ErrEmptyGuildID,
		},
		{
			name:      "state retrieval error returns ErrGuildStateFailed",
			stateErr:  errors.New("boom"),
			guildID:   "g1",
			botUserID: "bot",
			wantErr:   ErrGuildStateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := &mockSession{
				voiceStates: tt.voiceStates,
				stateErr:    tt.stateErr,
			}

			got, err := FindPopulatedChannel(sess, tt.guildID, tt.botUserID, discardLogger)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FindPopulatedChannel() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("FindPopulatedChannel() unexpected error: %v", err)
			}
			if got != tt.wantChannel {
				t.Errorf("FindPopulatedChannel() = %q, want %q", got, tt.wantChannel)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkFindPopulatedChannel_100States(b *testing.B) {
	states := make([]*VoiceState, 100)
	for i := 0; i < 100; i++ {
		uid := "user-" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		cid := "channel-" + string(rune('A'+i%10))
		states[i] = &VoiceState{
			UserID:    uid,
			ChannelID: cid,
			GuildID:   "g1",
		}
	}
	sess := &mockSession{voiceStates: states}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FindPopulatedChannel(sess, "g1", "bot-id-that-matches-none", discardLogger)
	}
}
