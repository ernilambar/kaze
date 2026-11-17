//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// AppKit must be used on the main thread; Wails runs menu callbacks on a
// goroutine, so hop back before touching NSApplication.
static void callOnMainThread(SEL selector) {
	[[NSApplication sharedApplication] performSelectorOnMainThread:selector withObject:nil waitUntilDone:NO];
}

static void showAboutPanel(void) { callOnMainThread(@selector(orderFrontStandardAboutPanel:)); }
static void hideApplication(void) { callOnMainThread(@selector(hide:)); }
static void hideOtherApplications(void) { callOnMainThread(@selector(hideOtherApplications:)); }
static void unhideAllApplications(void) { callOnMainThread(@selector(unhideAllApplications:)); }
*/
import "C"

// showAboutPanel opens the standard macOS About panel. AppKit centers the app
// icon, name, version and copyright, and takes all of them from the bundle's
// Info.plist, so there is nothing to lay out or keep in sync.
func showAboutPanel() {
	C.showAboutPanel()
}

// ShowAbout opens the standard About panel for the command palette. The app
// menu calls showAboutPanel directly; the palette has no native callback to
// ride, so it needs an exported binding.
func (a *App) ShowAbout() {
	showAboutPanel()
}

func hideApplication() {
	C.hideApplication()
}

func hideOtherApplications() {
	C.hideOtherApplications()
}

func unhideAllApplications() {
	C.unhideAllApplications()
}
