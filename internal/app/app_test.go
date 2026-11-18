package app

import (
	"testing"

	"kaze/internal/cli"
)

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
	app.emitSelectionCommand("add-cursor-above")
	app.emitZoom("in")
	app.emitTogglePreview()
	app.emitToggleView("wordWrap", false)
	app.SetViewOption("wordWrap", true)
	app.quit()
}

func TestQuitGuardAsksBeforeDroppingUnsavedBuffers(t *testing.T) {
	app := NewApp()
	app.registerWindow("main", cli.EmptyLaunchRequest())

	// Nothing modified: quitting needs no prompt.
	if !app.shouldQuit() {
		t.Fatal("shouldQuit() = false with no unsaved buffers")
	}

	app.SetUnsavedChanges("main", true)
	if app.shouldQuit() {
		t.Fatal("shouldQuit() = true, want unsaved buffers to hold the quit")
	}

	// Cancelling the prompt clears the request so the next quit asks again.
	app.ResolveClose("main", false)
	if app.session("main").closeRequested.Load() {
		t.Fatal("closeRequested = true after a cancelled prompt")
	}

	// Confirming the second prompt lets the guarded quit through. Tests never
	// attach a runtime, so the quit itself must stay a no-op.
	if app.shouldQuit() {
		t.Fatal("shouldQuit() = true while the prompt is still pending")
	}
	app.ResolveClose("main", true)
	if !app.shouldQuit() {
		t.Fatal("shouldQuit() = false after the user confirmed the quit")
	}
}

func TestOnlyTheFirstWindowOwnsTheSession(t *testing.T) {
	app := NewApp()
	first := app.registerWindow("main", cli.EmptyLaunchRequest())
	second := app.registerWindow("window-2", cli.EmptyLaunchRequest())

	if !app.ShouldRestoreSession("main") {
		t.Fatal("ShouldRestoreSession(main) = false")
	}
	if app.ShouldRestoreSession("window-2") {
		t.Fatal("ShouldRestoreSession(window-2) = true for a later window")
	}
	if request := first.takeLaunch(); !request.RestoreSession {
		t.Fatal("the first window did not receive the session")
	}
	if request := second.takeLaunch(); request.RestoreSession {
		t.Fatal("a later window must not restore or overwrite the session")
	}
}

func TestMovedBufferStaysWithItsWindow(t *testing.T) {
	app := NewApp()
	first := app.registerWindow("main", cli.EmptyLaunchRequest())
	moved := cli.LaunchBuffer{Name: "notes.txt", Path: "/notes.txt", Kind: "text", Content: "draft", Dirty: true}
	second := app.registerWindow("window-2", cli.LaunchRequest{Buffers: []cli.LaunchBuffer{moved}})

	if request := first.takeLaunch(); len(request.Buffers) != 0 {
		t.Fatalf("the original window received the moved buffer: %+v", request.Buffers)
	}
	request := second.takeLaunch()
	if len(request.Buffers) != 1 || request.Buffers[0].Content != "draft" || !request.Buffers[0].Dirty {
		t.Fatalf("the new window received %+v, want the dirty buffer", request.Buffers)
	}
}

func TestQuitAsksEachDirtyWindowOnce(t *testing.T) {
	app := NewApp()
	app.registerWindow("main", cli.EmptyLaunchRequest())
	app.registerWindow("window-2", cli.EmptyLaunchRequest())
	app.SetUnsavedChanges("main", true)
	app.SetUnsavedChanges("window-2", true)

	if app.shouldQuit() {
		t.Fatal("shouldQuit() = true with two dirty windows")
	}
	if !app.session("main").closeRequested.Load() {
		t.Fatal("the first dirty window was not asked")
	}
	if app.session("window-2").closeRequested.Load() {
		t.Fatal("the second dirty window was asked before the first answered")
	}

	app.ResolveClose("main", true)
	if !app.session("window-2").closeRequested.Load() {
		t.Fatal("the second dirty window was not asked after the first confirmed")
	}
	app.ResolveClose("window-2", true)
	if !app.allowQuit.Load() {
		t.Fatal("allowQuit = false after every dirty window confirmed")
	}
}
