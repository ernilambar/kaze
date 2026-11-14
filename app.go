package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the application struct. Exported methods become callable from the
// frontend through the generated Wails bindings.
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved so runtime
// methods can be called later.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetSettings loads the user's settings.json, creating it with defaults on
// first run. The frontend calls this once at startup and applies the values to
// the editor.
func (a *App) GetSettings() (Settings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return defaultSettings(), err
	}
	return loadSettings(path)
}

// OpenSettingsFile opens settings.json in the editor named by the user's
// environment ($VISUAL, then $EDITOR), falling back to the platform's default
// file handler. A missing file is created with defaults first; a malformed one
// is opened as-is so the user can repair it.
func (a *App) OpenSettingsFile() error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}
	if err := ensureSettingsFile(path); err != nil {
		return err
	}
	return openInEditor(path)
}

// quit shuts the app down through Wails so its shutdown path runs.
func (a *App) quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
