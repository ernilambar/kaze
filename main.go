package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// startupSettings loads settings.json before the menu is built, so the View
// menu starts with the same checkmarks the editor will use. A missing or broken
// file falls back to the defaults the frontend also falls back to.
func startupSettings() Settings {
	path, err := settingsFilePath()
	if err != nil {
		return defaultSettings()
	}
	settings, err := loadSettings(path)
	if err != nil {
		return defaultSettings()
	}
	return settings
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	meta := loadAppMetadata()

	// Help and version print in this process, before the single-instance lock,
	// so they never open or forward to a window.
	switch parseCLIAction(os.Args[1:]) {
	case cliActionHelp:
		fmt.Print(cliUsageText(meta))
		return
	case cliActionVersion:
		fmt.Print(cliVersionText(meta))
		return
	}

	// Paths on the command line are opened by the UI once it is live.
	app.queueLaunch(parseLaunchRequest(os.Args[1:]))

	wailsApp := application.New(application.Options{
		Name:        meta.ProductName,
		Description: "A simple text editor built with Wails and Monaco Editor.",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(app),
		},
		// Quitting with unsaved buffers is the frontend's decision: macOS asks
		// this first, and a false answer keeps the app alive until the user has
		// saved, discarded, or cancelled.
		ShouldQuit: app.shouldQuit,
		Mac: application.MacOptions{
			// The window is the app. Closing it ends the app instead of leaving
			// a process with no window to reopen.
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// One window, one document set: a second `kova <file>` from the terminal
		// or Finder hands its paths to the instance already running.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "net.nilambar.kova",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				// v3 forwards the full os.Args, whose first entry is the
				// executable itself, not a path to open.
				if len(data.Args) == 0 {
					return
				}
				app.queueLaunchPaths(data.Args[1:])
			},
		},
	})
	app.attach(wailsApp)

	// Finder's "Open With" and drag-onto-Dock arrive as application events.
	wailsApp.Event.OnApplicationEvent(events.Common.ApplicationOpenedWithFile, func(event *application.ApplicationEvent) {
		app.queueLaunchPaths([]string{event.Context().Filename()})
	})

	// The app menu is what puts "About Kova" in the macOS menu bar. It is built
	// from settings and keybindings so a custom binding does not leave a stale
	// native accelerator behind.
	app.meta = meta
	settings := startupSettings()
	wailsApp.Menu.Set(buildMenu(wailsApp, app, settings.Editor, settings.Theme, loadKeybindingsForMenu()))

	app.window = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:              meta.ProductName,
		Width:              1200,
		Height:             800,
		MinWidth:           600,
		MinHeight:          400,
		BackgroundColour:   application.NewRGB(30, 30, 30),
		URL:                "/",
		UseApplicationMenu: true,
	})

	// The red close button bypasses ShouldQuit, so it gets the same guard: the
	// close is cancelled and the frontend decides before the window goes away.
	app.window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if app.allowClose() {
			return
		}
		event.Cancel()
		app.requestClose()
	})

	if err := wailsApp.Run(); err != nil {
		println("Error:", err.Error())
	}
}
