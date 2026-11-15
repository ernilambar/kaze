package main

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// errNoWindow is returned by the window commands when the named window is not
// open. Unit tests call the bindings without a runtime, so the nil check is
// part of the contract.
var errNoWindow = errors.New("window is not available before startup")

// windowFor returns one window's native handle.
func (a *App) windowFor(name string) (*application.WebviewWindow, error) {
	session := a.session(name)
	if session == nil || session.window == nil {
		return nil, errNoWindow
	}
	return session.window, nil
}

// focusedWindowName is the name the native Window menu acts on. An empty name
// makes the binding fall back to the focused window.
func (a *App) focusedWindowName() string {
	if session := a.focusedSession(); session != nil {
		return session.name
	}
	return ""
}

// MinimiseWindow minimises one window. The native Window menu calls it for the
// focused one; the command palette reaches it through the generated binding.
func (a *App) MinimiseWindow(window string) error {
	target, err := a.windowFor(window)
	if err != nil {
		return err
	}
	target.Minimise()
	return nil
}

// ZoomWindow toggles a window between its standard and zoomed size, matching
// the macOS Window > Zoom item.
func (a *App) ZoomWindow(window string) error {
	target, err := a.windowFor(window)
	if err != nil {
		return err
	}
	target.Zoom()
	return nil
}

// ToggleFullScreen switches a window between fullscreen and its previous size.
func (a *App) ToggleFullScreen(window string) error {
	target, err := a.windowFor(window)
	if err != nil {
		return err
	}
	target.ToggleFullscreen()
	return nil
}
