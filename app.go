package main

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the application struct. Exported methods become callable from the
// frontend through the generated Wails bindings.
type App struct {
	ctx context.Context

	mu sync.Mutex
	// pending holds a launch request the frontend has not collected yet. It is
	// kept here because a second `kova file.txt` can arrive while the window is
	// still starting up, before anything is listening for the event.
	pending LaunchRequest
	// listening is set once the frontend has attached its launch listener, so
	// later requests can be emitted as events instead of queued.
	listening bool
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{pending: emptyLaunchRequest()}
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

// openSettingsFile opens settings.json in the editor named by the user's
// environment ($VISUAL, then $EDITOR), falling back to the platform's default
// file handler. A missing file is created with defaults first; a malformed one
// is opened as-is so the user can repair it.
func (a *App) openSettingsFile() error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}
	if err := ensureSettingsFile(path); err != nil {
		return err
	}
	return openInEditor(path)
}

// openSettingsFromMenu opens settings.json from the app menu. A menu click has
// no caller to return to, so failures are logged.
func (a *App) openSettingsFromMenu() {
	if err := a.openSettingsFile(); err != nil {
		runtime.LogError(a.ctx, err.Error())
	}
}

// quit shuts the app down through Wails so its shutdown path runs.
func (a *App) quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// GetLaunchRequest hands the frontend the paths Kova was started with. The
// frontend calls it once after attaching its listener, so anything passed in
// before the UI was ready is still opened.
func (a *App) GetLaunchRequest() LaunchRequest {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.listening = true
	request := a.pending.normalized()
	a.pending = emptyLaunchRequest()
	return request
}

// queueLaunch records what a command line asked to open and, when the UI is
// already listening, hands it over immediately.
func (a *App) queueLaunch(request LaunchRequest) {
	if request.Empty() {
		return
	}

	a.mu.Lock()
	a.pending.Merge(request)
	ready := a.listening
	var payload LaunchRequest
	if ready {
		payload = a.pending.normalized()
		a.pending = emptyLaunchRequest()
	}
	a.mu.Unlock()

	if !ready {
		return
	}
	// A second launch should surface the window, not leave files open behind
	// another app.
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowShow(a.ctx)
	runtime.EventsEmit(a.ctx, launchEvent, payload)
}

// queueLaunchPaths is queueLaunch for callers that only have path strings: the
// macOS "Open With" handler and the second-instance notification. Those paths
// are already absolute, either from Finder or from the installed launcher,
// which resolves relative arguments in the caller's shell — Wails reports the
// second instance's working directory as the bundle's executable directory, so
// it cannot be trusted for this.
func (a *App) queueLaunchPaths(paths []string) {
	a.queueLaunch(parseLaunchRequest(paths))
}

// InstallCommandLineTool installs the `kova` command in /usr/local/bin, asking
// for an administrator password when the directory is not writable. The menu
// item reports the outcome, since a menu click has no caller to return to.
func (a *App) installCommandLineToolFromMenu() {
	err := installCommandLineTool()
	if err != nil {
		runtime.LogError(a.ctx, err.Error())
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Install Command Line Tool",
			Message: err.Error(),
		})
		return
	}

	path := commandInstallPath()
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Install Command Line Tool",
		Message: "Installed " + path + ".\n\nOpen a new terminal and run:\n\n  kova notes.txt\n  kova .",
	})
}
