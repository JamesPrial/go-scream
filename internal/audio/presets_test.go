package audio

import (
	"testing"
)

func TestAllPresets_ReturnsAll6(t *testing.T) {
	presets := AllPresets()
	if len(presets) != 6 {
		t.Fatalf("AllPresets() returned %d presets, want 6", len(presets))
	}

	// Verify all expected names are present
	expected := map[PresetName]bool{
		PresetClassic:    true,
		PresetWhisper:    true,
		PresetDeathMetal: true,
		PresetGlitch:     true,
		PresetBanshee:    true,
		PresetRobot:      true,
	}
	for _, name := range presets {
		if !expected[name] {
			t.Errorf("unexpected preset name: %q", name)
		}
		delete(expected, name)
	}
	for name := range expected {
		t.Errorf("missing preset name: %q", name)
	}
}

func TestGetPreset_AllPresetsValid(t *testing.T) {
	for _, name := range AllPresets() {
		t.Run(string(name), func(t *testing.T) {
			p, ok := GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false", name)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("preset %q failed validation: %v", name, err)
			}
		})
	}
}

func TestGetPreset_Unknown(t *testing.T) {
	_, ok := GetPreset(PresetName("nonexistent"))
	if ok {
		t.Error("GetPreset(\"nonexistent\") returned true, want false")
	}
}

func TestGetPreset_ParameterRanges(t *testing.T) {
	for _, name := range AllPresets() {
		t.Run(string(name), func(t *testing.T) {
			p, ok := GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false", name)
			}

			if p.SampleRate != 48000 {
				t.Errorf("SampleRate = %d, want 48000", p.SampleRate)
			}
			if p.Channels != 2 {
				t.Errorf("Channels = %d, want 2", p.Channels)
			}
			if p.Duration <= 0 {
				t.Errorf("Duration = %v, want > 0", p.Duration)
			}
		})
	}
}
