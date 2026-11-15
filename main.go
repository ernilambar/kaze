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

// fileMenu holds the document and window commands. Buffers, their paths and
// their tabs live in the frontend, so every item forwards a command instead of
// acting on the file itself.
func fileMenu(m *application.Menu, app *App) {
	m.Add("New File").SetAccelerator("CmdOrCtrl+n").OnClick(func(*application.Context) {
		app.emitNewFile()
	})
	m.Add("Open…").SetAccelerator("CmdOrCtrl+o").OnClick(func(*application.Context) {
		app.emitOpenFile()
	})
	m.AddSeparator()
	m.Add("Save").SetAccelerator("CmdOrCtrl+s").OnClick(func(*application.Context) {
		app.emitSave()
	})
	m.Add("Save As…").SetAccelerator("CmdOrCtrl+Shift+s").OnClick(func(*application.Context) {
		app.emitSaveAs()
	})
	m.Add("Save All").SetAccelerator("CmdOrCtrl+Alt+s").OnClick(func(*application.Context) {
		app.emitSaveAll()
	})
	m.AddSeparator()
	// Print deliberately has no shortcut: Cmd+P is Quick Open.
	m.Add("Print…").OnClick(func(*application.Context) {
		app.emitPrint()
	})
	m.AddSeparator()
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

// caseTransforms is the Edit menu's Transform Case submenu, in display order.
// The id matches the frontend's CaseTransform values.
var caseTransforms = []struct{ label, id string }{
	{"Uppercase", "upper"},
	{"Lowercase", "lower"},
	{"Title Case", "title"},
	{"camelCase", "camel"},
	{"PascalCase", "pascal"},
	{"snake_case", "snake"},
	{"kebab-case", "kebab"},
	{"CONSTANT_CASE", "constant"},
}

// editMenu holds the buffer transforms. They act on Monaco's selection, which
// lives in the frontend, so each item only forwards the command.
func editMenu(m *application.Menu, app *App) {
	transform := m.AddSubmenu("Transform Case")
	for _, entry := range caseTransforms {
		transform.Add(entry.label).OnClick(func(*application.Context) {
			app.emitTransformCase(entry.id)
		})
	}
	m.AddSeparator()
	m.Add("Trim Trailing Whitespace").OnClick(func(*application.Context) {
		app.emitTrimWhitespace()
	})
}

// viewOption is one checkable View menu item. name is the editor settings field
// the frontend applies; label is what the menu prints.
type viewOption struct {
	name    string
	label   string
	checked bool
}

// viewOptions lists the View menu checkboxes in display order.
func viewOptions(editor EditorSettings) []viewOption {
	return []viewOption{
		{"wordWrap", "Word Wrap", editor.WordWrap},
		{"invisibleCharacters", "Invisible Characters", editor.InvisibleCharacters},
		{"lineNumbers", "Line Numbers", editor.LineNumbers},
		{"indentGuides", "Indent Guides", editor.IndentGuides},
		{"codeFolding", "Code Folding", editor.CodeFolding},
		{"autoCloseBrackets", "Auto-Close Brackets", editor.AutoCloseBrackets},
		{"smartIndent", "Smart Indent", editor.SmartIndent},
	}
}

// viewMenu holds zoom, preview and the editor view toggles. A checkbox owns its
// state natively, so each click forwards the new value; SetViewOption keeps the
// checkmark honest when the same toggle runs from the command palette.
func viewMenu(m *application.Menu, app *App, editor EditorSettings) {
	m.Add("Zoom In").SetAccelerator("CmdOrCtrl+=").OnClick(func(*application.Context) {
		app.emitZoom("in")
	})
	m.Add("Zoom Out").SetAccelerator("CmdOrCtrl+-").OnClick(func(*application.Context) {
		app.emitZoom("out")
	})
	m.Add("Actual Size").SetAccelerator("CmdOrCtrl+0").OnClick(func(*application.Context) {
		app.emitZoom("reset")
	})
	m.AddSeparator()
	m.Add("Toggle Preview").SetAccelerator("CmdOrCtrl+Ctrl+V").OnClick(func(*application.Context) {
		app.emitTogglePreview()
	})
	m.AddSeparator()
	for _, option := range viewOptions(editor) {
		item := m.AddCheckbox(option.label, option.checked)
		item.OnClick(func(ctx *application.Context) {
			app.emitToggleView(option.name, ctx.IsChecked())
		})
		app.viewItems[option.name] = item
	}
}

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

	// The app menu is what puts "About Kova" in the macOS menu bar.
	menu := wailsApp.NewMenu()
	appMenu(menu.AddSubmenu(meta.ProductName), meta.ProductName, app)
	fileMenu(menu.AddSubmenu("File"), app)
	editMenu(menu.AddSubmenu("Edit"), app)
	viewMenu(menu.AddSubmenu("View"), app, startupSettings().Editor)
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
