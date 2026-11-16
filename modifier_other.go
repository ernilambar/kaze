//go:build !darwin || !cgo

package main

// optionKeyHeld reports whether Option is down. Only macOS delivers a native
// file drag the webview cannot see, so elsewhere a drop never opens a new
// window on a modifier.
func optionKeyHeld() bool {
	return false
}
