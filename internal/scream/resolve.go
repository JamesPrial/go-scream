package scream

import (
	"github.com/JamesPrial/go-scream/internal/audio"
	"github.com/JamesPrial/go-scream/internal/config"
)

// resolveParams derives audio.ScreamParams from the provided Config.
// If cfg.Preset is set, it looks up the named preset and returns an error
// if the name is unknown. If cfg.Preset is empty, Randomize is used to
// generate random parameters. In either case, a positive cfg.Duration
// overrides the duration from the preset or random params.
func resolveParams(cfg config.Config) (audio.ScreamParams, error) {
	var params audio.ScreamParams

	if cfg.Preset != "" {
		p, ok := audio.GetPreset(audio.PresetName(cfg.Preset))
		if !ok {
			return audio.ScreamParams{}, ErrUnknownPreset
		}
		params = p
	} else {
		params = audio.Randomize(0)
	}

	if cfg.Duration > 0 {
		params.Duration = cfg.Duration
	}

	return params, nil
}
