// Package floatbutton provides a floating push-to-talk button rendered as a
// small circular NSPanel that hovers above all other windows.
//
// The panel uses NSWindowStyleMaskNonactivatingPanel so it never steals
// keyboard focus — the focused text field (target for text injection) stays
// active the whole time.
//
// Press and Release are channels the caller selects on; they receive a value
// each time the button is pushed down or released.
package floatbutton

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics
#include "floatbutton.h"
*/
import "C"

// Press receives a value each time the floating button is pushed down.
var Press = make(chan struct{}, 1)

// Release receives a value each time the floating button is released.
var Release = make(chan struct{}, 1)

// Position holds a screen coordinate pair reported after a drag.
type Position struct{ X, Y float64 }

// Moved receives the button's new screen position (bottom-left origin) after
// the user drags it to a new location.
var Moved = make(chan Position, 1)

// Show makes the floating button visible, creating it centered on the screen
// the first time.
func Show() { C.showFloatButton() }

// ShowAt makes the floating button visible at the given screen coordinates
// (bottom-left origin of the button frame).
func ShowAt(x, y float64) { C.showFloatButtonAt(C.double(x), C.double(y)) }

// Hide removes the floating button from the screen without destroying it.
func Hide() { C.hideFloatButton() }

// SetRecording updates the button's visual state.
// Pass true while audio is being captured; false otherwise.
func SetRecording(active bool) {
	if active {
		C.setFloatButtonRecording(1)
	} else {
		C.setFloatButtonRecording(0)
	}
}

// floatButtonPressed is called from Objective-C on the main thread when the
// button receives a mouseDown event.
//
//export floatButtonPressed
func floatButtonPressed() {
	select {
	case Press <- struct{}{}:
	default:
	}
}

// floatButtonReleased is called from Objective-C on the main thread when the
// button receives a mouseUp event (no drag occurred).
//
//export floatButtonReleased
func floatButtonReleased() {
	select {
	case Release <- struct{}{}:
	default:
	}
}

// floatButtonMoved is called from Objective-C after the user finishes dragging
// the button to a new position. x, y are the screen-coordinate frame origin.
//
//export floatButtonMoved
func floatButtonMoved(x, y C.double) {
	select {
	case Moved <- Position{X: float64(x), Y: float64(y)}:
	default:
	}
}
