package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
unsigned char *renderEmojiPNG(const char *emoji, int size, int *outLen);
unsigned char *renderEmojiRedPNG(const char *emoji, int size, int *outLen);
*/
import "C"
import (
	"bytes"
	"image"
	"image/png"
	"sync"
	"time"
	"unsafe"

	"fyne.io/systray"
)

var (
	iconMic    []byte
	iconMicRed []byte
	iconEmpty  []byte
)

// initIcons renders the menu bar icons. Must be called after Cocoa / systray
// has started (i.e. from onReady) because NSString drawing needs a running app.
func initIcons() {
	iconMic = renderEmoji("🎤", 22)
	iconMicRed = renderEmojiRed("🎤", 22)
	iconEmpty = makeEmptyPNG()
}

func renderEmoji(emoji string, size int) []byte {
	cs := C.CString(emoji)
	defer C.free(unsafe.Pointer(cs))
	var outLen C.int
	ptr := C.renderEmojiPNG(cs, C.int(size), &outLen)
	if ptr == nil {
		return makeEmptyPNG()
	}
	data := C.GoBytes(unsafe.Pointer(ptr), outLen)
	C.free(unsafe.Pointer(ptr))
	return data
}

func renderEmojiRed(emoji string, size int) []byte {
	cs := C.CString(emoji)
	defer C.free(unsafe.Pointer(cs))
	var outLen C.int
	ptr := C.renderEmojiRedPNG(cs, C.int(size), &outLen)
	if ptr == nil {
		return makeEmptyPNG()
	}
	data := C.GoBytes(unsafe.Pointer(ptr), outLen)
	C.free(unsafe.Pointer(ptr))
	return data
}

func makeEmptyPNG() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 22, 22))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// ---- blink state --------------------------------------------------------

var (
	blinkMu   sync.Mutex
	blinkStop chan struct{}
)

// startBlinking alternates the menu bar icon between the red 🎤 and
// transparent at 600 ms intervals to signal active recording.
func startBlinking() {
	blinkMu.Lock()
	defer blinkMu.Unlock()
	if blinkStop != nil {
		return // already blinking
	}
	blinkStop = make(chan struct{})
	go func(stop <-chan struct{}) {
		tick := time.NewTicker(600 * time.Millisecond)
		defer tick.Stop()
		red := true
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				if red {
					systray.SetIcon(iconMic)
				} else {
					systray.SetIcon(iconMicRed)
				}
				red = !red
			}
		}
	}(blinkStop)
	systray.SetIcon(iconMicRed) // start red
}

// stopBlinking halts the blink goroutine.
func stopBlinking() {
	blinkMu.Lock()
	defer blinkMu.Unlock()
	if blinkStop != nil {
		close(blinkStop)
		blinkStop = nil
	}
}

// ---- icon helpers called from main.go -----------------------------------

func setIdleIcon() {
	stopBlinking()
	systray.SetIcon(iconMic)
}

func setListeningIcon() {
	startBlinking()
}

func setProcessingIcon() {
	stopBlinking()
	systray.SetIcon(iconMic)
}
