package main

import "errors"

// errNoWindow is returned by the window commands before main creates the
// window. Unit tests call the bindings without one, so the nil check is part of
// the contract.
var errNoWindow = errors.New("window is not available before startup")

// MinimiseWindow minimises the window. The native Window menu calls it
// directly; the command palette reaches it through the generated binding.
func (a *App) MinimiseWindow() error {
	if a.window == nil {
		return errNoWindow
	}
	a.window.Minimise()
	return nil
}

// ZoomWindow toggles the window between its standard and zoomed size, matching
// the macOS Window > Zoom item.
func (a *App) ZoomWindow() error {
	if a.window == nil {
		return errNoWindow
	}
	a.window.Zoom()
	return nil
}

// ToggleFullScreen switches the window between fullscreen and its previous
// size.
func (a *App) ToggleFullScreen() error {
	if a.window == nil {
		return errNoWindow
	}
	a.window.ToggleFullscreen()
	return nil
}
