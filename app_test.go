package main

import "testing"

func TestMenuCommandsBeforeAttachAreNoOps(t *testing.T) {
	app := NewApp()

	// A menu callback can fire before main has attached the application, and
	// unit tests never attach one at all. The guards must absorb that instead
	// of dereferencing a nil runtime.
	app.emitAddFolder()
	app.emitCloseTab()
	app.emitQuickOpen()
	app.emitCommandPalette()
	app.emitTransformCase("upper")
	app.emitTrimWhitespace()
	app.emitZoom("in")
	app.emitTogglePreview()
	app.emitToggleView("wordWrap", false)
	app.SetViewOption("wordWrap", true)
	app.quit()
}

func TestQuitGuardAsksBeforeDroppingUnsavedBuffers(t *testing.T) {
	app := NewApp()

	// Nothing modified: quitting needs no prompt.
	if !app.shouldQuit() {
		t.Fatal("shouldQuit() = false with no unsaved buffers")
	}

	app.SetUnsavedChanges(true)
	if app.shouldQuit() {
		t.Fatal("shouldQuit() = true, want unsaved buffers to hold the quit")
	}

	// Cancelling the prompt clears the request so the next quit asks again.
	app.ResolveClose(false)
	if app.closeRequested.Load() {
		t.Fatal("closeRequested = true after a cancelled prompt")
	}

	// Confirming lets the guarded quit through. Tests never attach a runtime,
	// so the quit itself must stay a no-op.
	app.ResolveClose(true)
	if !app.shouldQuit() {
		t.Fatal("shouldQuit() = false after the user confirmed the quit")
	}
}
