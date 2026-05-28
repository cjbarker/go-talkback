// Package hotkey wraps golang.design/x/hotkey for push-to-talk registration.
// The default hotkey is Option+Space; callers can override by providing
// different modifiers/key values.
//
// macOS requirement: Register must be called from the main OS thread.
// When using fyne.io/systray the main thread is already managed by systray,
// so the caller should invoke Register inside the systray onReady callback.
package hotkey

import (
	"golang.design/x/hotkey"
)

// Handler holds a registered global hotkey and exposes Press/Release channels.
type Handler struct {
	hk *hotkey.Hotkey
}

// Register creates and registers the given hotkey combination.
// Default push-to-talk: Option+Space.
func Register(mods []hotkey.Modifier, key hotkey.Key) (*Handler, error) {
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		return nil, err
	}
	return &Handler{hk: hk}, nil
}

// Keydown returns a channel that receives a value each time the hotkey is pressed.
func (h *Handler) Keydown() <-chan hotkey.Event {
	return h.hk.Keydown()
}

// Keyup returns a channel that receives a value each time the hotkey is released.
func (h *Handler) Keyup() <-chan hotkey.Event {
	return h.hk.Keyup()
}

// Unregister deregisters the hotkey.
func (h *Handler) Unregister() {
	h.hk.Unregister()
}
