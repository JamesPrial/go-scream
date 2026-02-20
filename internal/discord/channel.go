package discord

import (
	"fmt"
	"log/slog"
)

// FindPopulatedChannel returns the channel ID of the first voice channel
// containing at least one non-bot user. It returns ErrEmptyGuildID if guildID
// is empty, ErrGuildStateFailed if the guild state cannot be retrieved, and
// ErrNoPopulatedChannel if every voice state belongs to the bot or no users
// are present.
func FindPopulatedChannel(session Session, guildID, botUserID string, logger *slog.Logger) (string, error) {
	if guildID == "" {
		return "", ErrEmptyGuildID
	}

	logger.Debug("searching for populated voice channel", "guild", guildID)

	voiceStates, err := session.GuildVoiceStates(guildID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGuildStateFailed, err)
	}

	for _, vs := range voiceStates {
		if vs.ChannelID != "" && vs.UserID != botUserID {
			logger.Debug("found populated channel", "channel", vs.ChannelID, "user", vs.UserID)
			return vs.ChannelID, nil
		}
	}

	return "", ErrNoPopulatedChannel
}
