// Package config loads and saves application settings to
// ~/.config/go-talkback/config.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultHotkey = "Option+Space"

// Config holds all persisted application settings.
type Config struct {
	Hotkey            string   `json:"hotkey"`
	Model             string   `json:"model"`
	ShowFloatButton   bool     `json:"showFloatButton"`
	FloatX            *float64 `json:"floatX,omitempty"`
	FloatY            *float64 `json:"floatY,omitempty"`
	LaunchAtLogin     bool     `json:"launchAtLogin"`
	StripFillerWords  bool     `json:"stripFillerWords"`
}

// Load reads the config file, returning defaults for any missing fields.
func Load() Config {
	c := Config{Hotkey: DefaultHotkey, StripFillerWords: true}
	data, err := os.ReadFile(path())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	if c.Hotkey == "" {
		c.Hotkey = DefaultHotkey
	}
	if c.Model == "" {
		c.Model = "base.en"
	}
	return c
}

// Save writes the config file, creating parent directories as needed.
func Save(c Config) error {
	p := path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "go-talkback", "config.json")
}
