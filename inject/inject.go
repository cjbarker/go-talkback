package inject

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Foundation -framework CoreFoundation
#include "inject.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// Text types text at the current cursor position in the focused application
// using CGEvent Unicode injection, bypassing keyboard layout.
func Text(s string) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.injectText(cs)
}
