package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// appMenu builds the macOS application menu. Wails' own AppMenu role renders
// About as a left-aligned alert, so the items are assembled here to open the
// standard macOS About panel instead, which centers the icon and text.
func appMenu(name string, quit func()) *menu.Menu {
	app := menu.NewMenu()
	app.AddText("About "+name, nil, func(*menu.CallbackData) { showAboutPanel() })
	app.AddSeparator()
	app.AddText("Hide "+name, keys.CmdOrCtrl("h"), func(*menu.CallbackData) { hideApplication() })
	app.AddText("Hide Others", keys.Combo("h", keys.CmdOrCtrlKey, keys.OptionOrAltKey), func(*menu.CallbackData) {
		hideOtherApplications()
	})
	app.AddText("Show All", nil, func(*menu.CallbackData) { unhideAllApplications() })
	app.AddSeparator()
	app.AddText("Quit "+name, keys.CmdOrCtrl("q"), func(*menu.CallbackData) { quit() })
	return app
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	meta := loadAppMetadata()

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
		// The app menu is what puts "About Kova" in the macOS menu bar.
		Menu: menu.NewMenuFromItems(
			menu.SubMenu(meta.ProductName, appMenu(meta.ProductName, app.quit)),
		),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
