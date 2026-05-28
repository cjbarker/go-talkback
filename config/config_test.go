package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cbarker/go-talkback/config"
)

// isolate redirects HOME to a temp dir so Load/Save don't touch the real
// ~/.config directory.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	isolate(t)
	c := config.Load()
	if c.Hotkey != config.DefaultHotkey {
		t.Errorf("Hotkey = %q, want %q", c.Hotkey, config.DefaultHotkey)
	}
}

func TestLoad_EmptyHotkey_FallsBackToDefault(t *testing.T) {
	isolate(t)
	// Write a config with an empty hotkey field.
	writeRaw(t, `{"hotkey":""}`)
	c := config.Load()
	if c.Hotkey != config.DefaultHotkey {
		t.Errorf("Hotkey = %q, want %q", c.Hotkey, config.DefaultHotkey)
	}
}

func TestLoad_ValidFile_ReturnsStoredValues(t *testing.T) {
	isolate(t)
	writeRaw(t, `{"hotkey":"Ctrl+Space"}`)
	c := config.Load()
	if c.Hotkey != "Ctrl+Space" {
		t.Errorf("Hotkey = %q, want %q", c.Hotkey, "Ctrl+Space")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	isolate(t)
	want := config.Config{Hotkey: "Option+R"}
	if err := config.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := config.Load()
	if got.Hotkey != want.Hotkey {
		t.Errorf("after round-trip: Hotkey = %q, want %q", got.Hotkey, want.Hotkey)
	}
}

func TestSave_CreatesParentDirs(t *testing.T) {
	isolate(t)
	// Config dir doesn't exist yet; Save should create it.
	if err := config.Save(config.Config{Hotkey: "Cmd+Shift+R"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".config", "go-talkback", "config.json")
	if _, err := os.Stat(p); err != nil {
		t.Errorf("expected config file at %s: %v", p, err)
	}
}

func TestSave_WritesValidJSON(t *testing.T) {
	isolate(t)
	if err := config.Save(config.Config{Hotkey: "Cmd+Shift+Space"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".config", "go-talkback", "config.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("config file is not valid JSON: %v\n%s", err, data)
	}
}

// writeRaw writes raw JSON content to the config file location.
func writeRaw(t *testing.T, content string) {
	t.Helper()
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "go-talkback")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
