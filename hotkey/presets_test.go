package hotkey_test

import (
	"testing"

	"github.com/cbarker/go-talkback/hotkey"
	"github.com/cbarker/go-talkback/config"
)

func TestFind_KnownPreset(t *testing.T) {
	for _, p := range hotkey.Presets {
		got := hotkey.Find(p.Name)
		if got.Name != p.Name {
			t.Errorf("Find(%q).Name = %q, want %q", p.Name, got.Name, p.Name)
		}
		if len(got.Mods) == 0 {
			t.Errorf("Find(%q).Mods is empty", p.Name)
		}
	}
}

func TestFind_UnknownName_FallsBackToFirst(t *testing.T) {
	got := hotkey.Find("NonExistentHotkey+X")
	if got.Name != hotkey.Presets[0].Name {
		t.Errorf("Find(unknown).Name = %q, want first preset %q", got.Name, hotkey.Presets[0].Name)
	}
}

func TestPresets_AllHaveNonEmptyNames(t *testing.T) {
	if len(hotkey.Presets) == 0 {
		t.Fatal("Presets is empty")
	}
	for i, p := range hotkey.Presets {
		if p.Name == "" {
			t.Errorf("Presets[%d].Name is empty", i)
		}
		if p.Key == 0 {
			t.Errorf("Presets[%d] (%q) has zero Key", i, p.Name)
		}
	}
}

func TestPresets_NoDuplicateNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range hotkey.Presets {
		if seen[p.Name] {
			t.Errorf("duplicate preset name %q", p.Name)
		}
		seen[p.Name] = true
	}
}

func TestPresets_DefaultIsPresent(t *testing.T) {
	got := hotkey.Find(config.DefaultHotkey)
	if got.Name != config.DefaultHotkey {
		t.Errorf("default hotkey %q not found in Presets", config.DefaultHotkey)
	}
}
