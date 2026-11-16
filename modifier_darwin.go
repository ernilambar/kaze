//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

static int optionKeyHeld(void) {
	return ([NSEvent modifierFlags] & NSEventModifierFlagOption) != 0;
}
*/
import "C"

import "github.com/wailsapp/wails/v3/pkg/application"

// optionKeyHeld reports whether Option is down right now. A Finder drop is
// delivered by AppKit, so the webview never sees the drag and only the native
// modifier state can tell whether the user asked for a new window.
func optionKeyHeld() bool {
	held := false
	application.InvokeSync(func() {
		held = C.optionKeyHeld() != 0
	})
	return held
}
