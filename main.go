package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

// appMenu builds the macOS application menu. Wails' own AppMenu role renders
// About as a left-aligned alert, so the items are assembled here to open the
// standard macOS About panel instead, which centers the icon and text.
func appMenu(name string, app *App) *menu.Menu {
	appMenu := menu.NewMenu()
	appMenu.AddText("About "+name, nil, func(*menu.CallbackData) { showAboutPanel() })
	appMenu.AddSeparator()
	appMenu.AddText("Settings…", keys.CmdOrCtrl(","), func(*menu.CallbackData) { app.openSettingsFromMenu() })
	appMenu.AddText("Install Command Line Tool…", nil, func(*menu.CallbackData) {
		app.installCommandLineToolFromMenu()
	})
	appMenu.AddSeparator()
	appMenu.AddText("Hide "+name, keys.CmdOrCtrl("h"), func(*menu.CallbackData) { hideApplication() })
	appMenu.AddText("Hide Others", keys.Combo("h", keys.CmdOrCtrlKey, keys.OptionOrAltKey), func(*menu.CallbackData) {
		hideOtherApplications()
	})
	appMenu.AddText("Show All", nil, func(*menu.CallbackData) { unhideAllApplications() })
	appMenu.AddSeparator()
	appMenu.AddText("Quit "+name, keys.CmdOrCtrl("q"), func(*menu.CallbackData) { app.quit() })
	return appMenu
}

// fileMenu holds the sidebar commands. Adding a folder is a window action, so it
// lives here rather than in the app menu.
func fileMenu(app *App) *menu.Menu {
	fileMenu := menu.NewMenu()
	fileMenu.AddText("Add Folder to Sidebar…", keys.Combo("o", keys.CmdOrCtrlKey, keys.ShiftKey), func(*menu.CallbackData) {
		app.emitAddFolder()
	})
	return fileMenu
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	meta := loadAppMetadata()

	// Paths on the command line are opened by the UI once it is live.
	app.queueLaunch(parseLaunchRequest(os.Args[1:]))

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Kova",
		Width:     1200,
		Height:    800,
		MinWidth:  600,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 30, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		// One window, one document set: a second `kova <file>` from the terminal
		// or Finder hands its paths to the instance already running.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "net.nilambar.kova",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				app.queueLaunchPaths(data.Args)
			},
		},
		Mac: &mac.Options{
			// Finder's "Open With" and drag-onto-Dock arrive here.
			OnFileOpen: func(path string) { app.queueLaunchPaths([]string{path}) },
		},
		// The app menu is what puts "About Kova" in the macOS menu bar.
		Menu: menu.NewMenuFromItems(
			menu.SubMenu(meta.ProductName, appMenu(meta.ProductName, app)),
			menu.SubMenu("File", fileMenu(app)),
		),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
