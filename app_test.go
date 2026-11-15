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
	app.quit()
}
