//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#include <dispatch/dispatch.h>

// AppKit must be used on the main thread; Wails runs menu callbacks on a
// goroutine, so hop back before touching NSApplication.
static void callOnMainThread(SEL selector) {
	[[NSApplication sharedApplication] performSelectorOnMainThread:selector withObject:nil waitUntilDone:NO];
}

// aboutStack finds the stack that lays out the standard About panel: app icon,
// name, version and credits. AppKit has no API for adding a button to that
// panel, so the stack is located by its icon instead of by view index.
static NSStackView *aboutStack(NSView *view) {
	if ([view isKindOfClass:[NSStackView class]]) {
		for (NSView *arranged in ((NSStackView *)view).arrangedSubviews) {
			if ([arranged isKindOfClass:[NSImageView class]]) {
				return (NSStackView *)view;
			}
		}
	}
	for (NSView *subview in view.subviews) {
		NSStackView *stack = aboutStack(subview);
		if (stack) {
			return stack;
		}
	}
	return nil;
}

static NSWindow *visibleAboutPanel(void) {
	for (NSWindow *window in [NSApplication sharedApplication].windows) {
		if (window.visible && aboutStack(window.contentView)) {
			return window;
		}
	}
	return nil;
}

// addDismissButton appends an OK button to the panel's stack. The stack's
// constraints then grow the panel to fit; the default Return key closes it.
static void addDismissButton(NSWindow *panel) {
	NSStackView *stack = aboutStack(panel.contentView);
	for (NSView *view in stack.arrangedSubviews) {
		if ([view.identifier isEqualToString:@"kaze.about.ok"]) {
			return;
		}
	}
	NSButton *ok = [NSButton buttonWithTitle:@"OK" target:panel action:@selector(performClose:)];
	ok.identifier = @"kaze.about.ok";
	ok.keyEquivalent = @"\r";
	ok.bezelStyle = NSBezelStyleRounded;
	[stack addArrangedSubview:ok];
}

static void showAboutPanelOnMainThread(void *context) {
	[NSApp orderFrontStandardAboutPanel:nil];
	NSWindow *panel = visibleAboutPanel();
	if (panel) {
		addDismissButton(panel);
	}
}

static void showAboutPanel(void) {
	dispatch_async_f(dispatch_get_main_queue(), NULL, showAboutPanelOnMainThread);
}
static void hideApplication(void) { callOnMainThread(@selector(hide:)); }
static void hideOtherApplications(void) { callOnMainThread(@selector(hideOtherApplications:)); }
static void unhideAllApplications(void) { callOnMainThread(@selector(unhideAllApplications:)); }
*/
import "C"

// showAboutPanel opens the standard macOS About panel. AppKit centers the app
// icon, name, version and copyright, and takes all of them from the bundle's
// Info.plist. The panel has no button of its own, so an OK button is appended
// to its stack; Return dismisses it as well.
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
