package discord

import (
	"context"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Session abstracts the subset of *discordgo.Session methods used by this package.
type Session interface {
	ChannelVoiceJoin(ctx context.Context, guildID, channelID string, mute, deaf bool) (VoiceConn, error)
	GuildVoiceStates(guildID string) ([]*VoiceState, error)
}

// VoiceConn abstracts the subset of *discordgo.VoiceConnection methods.
type VoiceConn interface {
	Speaking(speaking bool) error
	OpusSendChannel() chan<- []byte
	Disconnect(ctx context.Context) error
}

// VoiceState represents a user's voice connection state in a guild.
type VoiceState struct {
	UserID    string
	ChannelID string
	GuildID   string
}

// GoSession wraps *discordgo.Session to satisfy the Session interface.
type GoSession struct {
	S      *discordgo.Session
	Logger *slog.Logger
}

// ChannelVoiceJoin joins the specified voice channel and returns a VoiceConn.
func (d *GoSession) ChannelVoiceJoin(ctx context.Context, guildID, channelID string, mute, deaf bool) (VoiceConn, error) {
	d.Logger.Info("joining voice channel", "guild", guildID, "channel", channelID)
	start := time.Now()
	vc, err := d.S.ChannelVoiceJoin(ctx, guildID, channelID, mute, deaf)
	if err != nil {
		d.Logger.Error("voice join failed", "guild", guildID, "error", err, "elapsed", time.Since(start))
		return nil, err
	}
	d.Logger.Info("voice join succeeded", "guild", guildID, "status", int(vc.Status), "elapsed", time.Since(start))
	return &GoVoiceConn{VC: vc, Logger: d.Logger}, nil
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
	VC     *discordgo.VoiceConnection
	Logger *slog.Logger
}

// Speaking sets the speaking state on the voice connection.
func (d *GoVoiceConn) Speaking(speaking bool) error {
	d.Logger.Debug("setting speaking state", "speaking", speaking, "vc_status", int(d.VC.Status))
	err := d.VC.Speaking(speaking)
	if err != nil {
		d.Logger.Error("speaking failed", "speaking", speaking, "error", err)
	}
	return err
}

// OpusSendChannel returns the channel used to send Opus-encoded audio frames.
func (d *GoVoiceConn) OpusSendChannel() chan<- []byte { return d.VC.OpusSend }

// Disconnect closes the voice connection.
func (d *GoVoiceConn) Disconnect(ctx context.Context) error {
	d.Logger.Info("disconnecting from voice", "vc_status", int(d.VC.Status))
	err := d.VC.Disconnect(ctx)
	if err != nil {
		d.Logger.Error("disconnect failed", "error", err)
	}
	return err
}
