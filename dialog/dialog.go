// Package dialog provides native macOS modal dialogs via Cocoa NSAlert/NSPanel.
package dialog

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include "dialog.h"
*/
import "C"

// ConfirmModelDownload shows a native macOS modal alert asking the user
// whether to download the base.en model. Returns true if the user clicks
// "Download", false if they click "Quit".
func ConfirmModelDownload() bool {
	return C.showDownloadConfirmDialog() == 1
}

// ShowProgressWindow opens a floating progress panel for the model download.
// Must be called before UpdateProgress / CloseProgressWithSuccess.
func ShowProgressWindow() {
	C.showProgressWindow()
}

// UpdateProgress updates the progress bar and percentage label (0–100).
// Safe to call from any goroutine.
func UpdateProgress(pct int) {
	C.updateProgressBar(C.int(pct))
}

// CloseProgressWithSuccess closes the progress panel and shows a "ready to
// use" success alert. Blocks until the user clicks "Get Started".
func CloseProgressWithSuccess() {
	C.closeProgressWithSuccess()
}

// CloseProgressWindow closes the progress panel without a success message.
// Used when the download fails.
func CloseProgressWindow() {
	C.closeProgressWindow()
}

// ConfirmLaunchAtLogin shows a native macOS modal alert asking the user
// whether to enable launch at login. Returns true if the user clicks "Enable".
func ConfirmLaunchAtLogin() bool {
	return C.showLaunchAtLoginDialog() == 1
}
