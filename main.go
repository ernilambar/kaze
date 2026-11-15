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

// appMenu fills the macOS application menu. Wails' own app menu role renders
// About as a left-aligned alert, so the items are assembled here to open the
// standard macOS About panel instead, which centers the icon and text.
func appMenu(m *application.Menu, name string, app *App) {
	m.Add("About " + name).OnClick(func(*application.Context) { showAboutPanel() })
	m.AddSeparator()
	m.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) { app.openSettingsFromMenu() })
	m.Add("Install Command Line Tool…").OnClick(func(*application.Context) {
		app.InstallCommandLineTool()
	})
	m.AddSeparator()
	m.Add("Hide " + name).SetAccelerator("CmdOrCtrl+h").OnClick(func(*application.Context) { hideApplication() })
	m.Add("Hide Others").SetAccelerator("CmdOrCtrl+Alt+h").OnClick(func(*application.Context) {
		hideOtherApplications()
	})
	m.Add("Show All").OnClick(func(*application.Context) { unhideAllApplications() })
	m.AddSeparator()
	m.Add("Quit " + name).SetAccelerator("CmdOrCtrl+q").OnClick(func(*application.Context) { app.quit() })
}

// fileMenu holds the window commands. Adding a folder and closing a tab are
// window actions, so they live here rather than in the app menu.
func fileMenu(m *application.Menu, app *App) {
	m.Add("Command Palette…").SetAccelerator("CmdOrCtrl+Shift+p").OnClick(func(*application.Context) {
		app.emitCommandPalette()
	})
	m.Add("Quick Open…").SetAccelerator("CmdOrCtrl+p").OnClick(func(*application.Context) {
		app.emitQuickOpen()
	})
	m.AddSeparator()
	m.Add("Add Folder to Sidebar…").SetAccelerator("CmdOrCtrl+Shift+o").OnClick(func(*application.Context) {
		app.emitAddFolder()
	})
	m.AddSeparator()
	m.Add("Close Tab").SetAccelerator("CmdOrCtrl+w").OnClick(func(*application.Context) {
		app.emitCloseTab()
	})
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

	// The app menu is what puts "About Kova" in the macOS menu bar.
	menu := wailsApp.NewMenu()
	appMenu(menu.AddSubmenu(meta.ProductName), meta.ProductName, app)
	fileMenu(menu.AddSubmenu("File"), app)
	wailsApp.Menu.Set(menu)

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

	if err := wailsApp.Run(); err != nil {
		println("Error:", err.Error())
	}
}
