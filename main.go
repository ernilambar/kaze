package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"kaze/internal/cli"
	"kaze/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

// startupSettings loads settings.json before the menu is built, so the View
// menu starts with the same checkmarks the editor will use. A missing or broken
// file falls back to the defaults the frontend also falls back to.
func startupSettings() config.Settings {
	path, err := settingsFilePath()
	if err != nil {
		return config.Default()
	}
	settings, err := config.Load(path)
	if err != nil {
		return config.Default()
	}
	return settings
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	meta := appMeta

	// Help and version print in this process, before the single-instance lock,
	// so they never open or forward to a window.
	switch cli.ParseAction(os.Args[1:]) {
	case cli.ActionHelp:
		fmt.Print(cliUsageText(meta))
		return
	case cli.ActionVersion:
		fmt.Print(cliVersionText(meta))
		return
	}

	// Paths on the command line are opened by the UI once it is live.
	app.queueLaunch(cli.ParseLaunchRequest(os.Args[1:]))

	wailsApp := application.New(application.Options{
		Name:        meta.ProductName,
		Description: "A lightweight macOS editor for code and plain text.",
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
			// Closing the last window ends the app instead of leaving a process
			// with no window to reopen. Closing one of several windows only
			// closes that window.
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// One process, several windows: a second launch with a file from the
		// terminal or Finder hands its paths to the window already in front.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: meta.ProductIdentifier,
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

	// The app menu is what puts "About <name>" in the macOS menu bar. It is built
	// from settings and keybindings so a custom binding does not leave a stale
	// native accelerator behind.
	app.meta = meta
	settings := startupSettings()
	wailsApp.Menu.Set(buildMenu(wailsApp, app, settings.Editor, settings.Theme, loadKeybindingsForMenu()))

	// The first window adopts the paths queued before it existed. Later windows
	// are created by New Window, Open Folder in New Window and Move Tab to New
	// Window, each with its own launch request.
	if err := app.openWindow(cli.EmptyLaunchRequest()); err != nil {
		println("Error:", err.Error())
		return
	}

	if err := wailsApp.Run(); err != nil {
		println("Error:", err.Error())
	}
}
