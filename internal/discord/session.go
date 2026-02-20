package discord

import "github.com/bwmarrin/discordgo"

// Session abstracts the subset of *discordgo.Session methods used by this package.
type Session interface {
	ChannelVoiceJoin(guildID, channelID string, mute, deaf bool) (VoiceConn, error)
	GuildVoiceStates(guildID string) ([]*VoiceState, error)
}

// VoiceConn abstracts the subset of *discordgo.VoiceConnection methods.
type VoiceConn interface {
	Speaking(speaking bool) error
	OpusSendChannel() chan<- []byte
	Disconnect() error
}

// VoiceState represents a user's voice connection state in a guild.
type VoiceState struct {
	UserID    string
	ChannelID string
	GuildID   string
}

// GoSession wraps *discordgo.Session to satisfy the Session interface.
type GoSession struct {
	S *discordgo.Session
}

// ChannelVoiceJoin joins the specified voice channel and returns a VoiceConn.
func (d *GoSession) ChannelVoiceJoin(guildID, channelID string, mute, deaf bool) (VoiceConn, error) {
	vc, err := d.S.ChannelVoiceJoin(guildID, channelID, mute, deaf)
	if err != nil {
		return nil, err
	}
	return &GoVoiceConn{VC: vc}, nil
}

// GuildVoiceStates returns the voice states for all users in the given guild.
func (d *GoSession) GuildVoiceStates(guildID string) ([]*VoiceState, error) {
	guild, err := d.S.State.Guild(guildID)
	if err != nil {
		return nil, err
	}
	states := make([]*VoiceState, len(guild.VoiceStates))
	for i, vs := range guild.VoiceStates {
		states[i] = &VoiceState{
			UserID:    vs.UserID,
			ChannelID: vs.ChannelID,
			GuildID:   vs.GuildID,
		}
	}
	return states, nil
}

// GoVoiceConn wraps *discordgo.VoiceConnection to satisfy the VoiceConn interface.
type GoVoiceConn struct {
	VC *discordgo.VoiceConnection
}

// Speaking sets the speaking state on the voice connection.
func (d *GoVoiceConn) Speaking(speaking bool) error { return d.VC.Speaking(speaking) }

// OpusSendChannel returns the channel used to send Opus-encoded audio frames.
func (d *GoVoiceConn) OpusSendChannel() chan<- []byte { return d.VC.OpusSend }

// Disconnect closes the voice connection.
func (d *GoVoiceConn) Disconnect() error { return d.VC.Disconnect() }
