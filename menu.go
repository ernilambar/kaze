package main

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// menuBuilder adds command items and applies keybindings.json. An item whose
// command is rebound, or whose default accelerator is claimed by another
// binding, loses its native accelerator: the frontend keybinding listener runs
// the command instead, so one press cannot fire twice.
type menuBuilder struct {
	app        *App
	suppressed map[string]bool
	claimed    map[string]bool
}

func newMenuBuilder(app *App, bindings []Keybinding) *menuBuilder {
	suppressed, claimed := menuSuppression(bindings)
	return &menuBuilder{app: app, suppressed: suppressed, claimed: claimed}
}

// item adds a plain command item. id may be empty for non-command rows such as
// About; those are never suppressed.
func (b *menuBuilder) item(
	parent *application.Menu,
	id string,
	label string,
	accelerator string,
	onClick func(*application.Context),
) *application.MenuItem {
	item := parent.Add(label)
	if accelerator != "" && !b.suppressed[id] && !b.claimed[normalizeAccelerator(accelerator)] {
		item.SetAccelerator(accelerator)
	}
	item.OnClick(onClick)
	return item
}

// reportMenuError surfaces a failure from an app-menu item, which has no caller
// to return an error to.
func (a *App) reportMenuError(title string, err error) {
	if a.wails == nil {
		return
	}
	a.wails.Logger.Error(err.Error())
	a.wails.Dialog.Error().SetTitle(title).SetMessage(err.Error()).Show()
}

// appMenu fills the macOS application menu. Wails' own app menu role renders
// About as a left-aligned alert, so the items are assembled here to open the
// standard macOS About panel instead, which centers the icon and text.
func appMenu(m *application.Menu, name string, app *App, b *menuBuilder) {
	b.item(m, "", "About "+name, "", func(*application.Context) { showAboutPanel() })
	m.AddSeparator()
	b.item(m, "settings", "Settings…", "CmdOrCtrl+,", func(*application.Context) {
		app.emitSettingsOverlay()
	})
	b.item(m, "open-settings-json", "Open Settings JSON", "", func(*application.Context) {
		app.openSettingsFromMenu()
	})
	b.item(m, "open-default-settings", "Open Default Settings", "", func(*application.Context) {
		if err := app.OpenDefaultSettings(); err != nil {
			app.reportMenuError("Open Default Settings", err)
		}
	})
	b.item(m, "open-themes-folder", "Open Themes Folder", "", func(*application.Context) {
		if err := app.OpenThemesFolder(); err != nil {
			app.reportMenuError("Open Themes Folder", err)
		}
	})
	b.item(m, "open-keybindings", "Open Keybindings…", "", func(*application.Context) {
		if err := app.OpenKeybindings(); err != nil {
			app.reportMenuError("Open Keybindings", err)
		}
	})
	b.item(m, "", "Install Command Line Tool…", "", func(*application.Context) {
		app.InstallCommandLineTool()
	})
	m.AddSeparator()
	b.item(m, "", "Hide "+name, "CmdOrCtrl+h", func(*application.Context) { hideApplication() })
	b.item(m, "", "Hide Others", "CmdOrCtrl+Alt+h", func(*application.Context) {
		hideOtherApplications()
	})
	b.item(m, "", "Show All", "", func(*application.Context) { unhideAllApplications() })
	m.AddSeparator()
	b.item(m, "", "Quit "+name, "CmdOrCtrl+q", func(*application.Context) { app.quit() })
}

// fileMenu holds the document and window commands. Buffers, their paths and
// their tabs live in the frontend, so every item forwards a command instead of
// acting on the file itself.
func fileMenu(m *application.Menu, app *App, b *menuBuilder) {
	b.item(m, "new-file", "New File", "CmdOrCtrl+n", func(*application.Context) {
		app.emitNewFile()
	})
	b.item(m, "open-file", "Open…", "CmdOrCtrl+o", func(*application.Context) {
		app.emitOpenFile()
	})
	// Control-R, not Command-R: Quick Open owns Cmd+P, and Cmd+R has no
	// document meaning here.
	b.item(m, "open-recent", "Open Recent…", "Ctrl+r", func(*application.Context) {
		app.emitOpenRecent()
	})
	b.item(m, "clear-recent", "Clear Recent", "", func(*application.Context) { app.emitClearRecent() })
	m.AddSeparator()
	b.item(m, "save", "Save", "CmdOrCtrl+s", func(*application.Context) { app.emitSave() })
	b.item(m, "save-as", "Save As…", "CmdOrCtrl+Shift+s", func(*application.Context) {
		app.emitSaveAs()
	})
	b.item(m, "save-all", "Save All", "CmdOrCtrl+Alt+s", func(*application.Context) {
		app.emitSaveAll()
	})
	m.AddSeparator()
	// Print deliberately has no shortcut: Cmd+P is Quick Open.
	b.item(m, "print", "Print…", "", func(*application.Context) { app.emitPrint() })
	m.AddSeparator()
	b.item(m, "command-palette", "Command Palette…", "CmdOrCtrl+Shift+p", func(*application.Context) {
		app.emitCommandPalette()
	})
	b.item(m, "quick-open", "Quick Open…", "CmdOrCtrl+p", func(*application.Context) {
		app.emitQuickOpen()
	})
	b.item(m, "find-in-files", "Find in Files…", "CmdOrCtrl+Shift+f", func(*application.Context) {
		app.emitFindInFiles()
	})
	m.AddSeparator()
	b.item(m, "add-folder", "Add Folder to Sidebar…", "CmdOrCtrl+Shift+o", func(*application.Context) {
		app.emitAddFolder()
	})
	b.item(m, "close-all-folders", "Close All Folders", "", func(*application.Context) {
		app.emitCloseAllFolders()
	})
	m.AddSeparator()
	b.item(m, "close-tab", "Close Tab", "CmdOrCtrl+w", func(*application.Context) {
		app.emitCloseTab()
	})
	b.item(m, "reopen-closed-tab", "Reopen Closed Tab", "CmdOrCtrl+Shift+t", func(*application.Context) {
		app.emitReopenClosedTab()
	})
	b.item(m, "next-tab", "Next Tab", "Ctrl+Tab", func(*application.Context) { app.emitNextTab() })
	b.item(m, "previous-tab", "Previous Tab", "Ctrl+Shift+Tab", func(*application.Context) {
		app.emitPreviousTab()
	})
	goToTab := m.AddSubmenu("Go to Tab")
	for index := 1; index <= 9; index++ {
		id := fmt.Sprintf("go-to-tab-%d", index)
		position := index
		b.item(goToTab, id, fmt.Sprintf("Tab %d", index), fmt.Sprintf("CmdOrCtrl+%d", index), func(*application.Context) {
			app.emitGoToTab(position)
		})
	}
	m.AddSeparator()
	b.item(m, "reveal-file", "Reveal in Finder", "", func(*application.Context) { app.emitRevealFile() })
	copyPath := m.AddSubmenu("Copy Path")
	for _, entry := range []struct{ id, label, kind string }{
		{"copy-file-path", "File Path", "file"},
		{"copy-relative-path", "Relative Path", "relative"},
		{"copy-file-name", "File Name", "name"},
	} {
		kind := entry.kind
		b.item(copyPath, entry.id, entry.label, "", func(*application.Context) {
			app.emitCopyPath(kind)
		})
	}
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
func editMenu(m *application.Menu, app *App, b *menuBuilder) {
	transform := m.AddSubmenu("Transform Case")
	for _, entry := range caseTransforms {
		id := "case-" + entry.id
		transformID := entry.id
		b.item(transform, id, entry.label, "", func(*application.Context) {
			app.emitTransformCase(transformID)
		})
	}
	m.AddSeparator()
	b.item(m, "trim-whitespace", "Trim Trailing Whitespace", "", func(*application.Context) {
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

// viewMenu holds zoom, preview, appearance and the editor view toggles. A
// checkbox owns its state natively, so each click forwards the new value;
// SetViewOption keeps the checkmark honest when the same toggle runs from the
// command palette.
func viewMenu(m *application.Menu, app *App, b *menuBuilder, editor EditorSettings, theme ThemeSettings) {
	b.item(m, "toggle-sidebar", "Toggle Sidebar", "CmdOrCtrl+b", func(*application.Context) {
		app.emitToggleSidebar()
	})
	b.item(m, "refresh", "Refresh", "", func(*application.Context) { app.emitRefresh() })
	m.AddSeparator()
	b.item(m, "zoom-in", "Zoom In", "CmdOrCtrl+=", func(*application.Context) { app.emitZoom("in") })
	b.item(m, "zoom-out", "Zoom Out", "CmdOrCtrl+-", func(*application.Context) { app.emitZoom("out") })
	b.item(m, "zoom-reset", "Actual Size", "CmdOrCtrl+0", func(*application.Context) { app.emitZoom("reset") })
	m.AddSeparator()
	b.item(m, "toggle-preview", "Toggle Preview", "CmdOrCtrl+Ctrl+V", func(*application.Context) {
		app.emitTogglePreview()
	})
	m.AddSeparator()

	appearance := m.AddSubmenu("Appearance")
	for _, mode := range []struct{ id, label string }{
		{"appearance-system", "System"},
		{"appearance-light", "Light"},
		{"appearance-dark", "Dark"},
	} {
		modeValue := mode.id[len("appearance-"):]
		item := appearance.AddRadio(mode.label, theme.Mode == modeValue)
		item.OnClick(func(*application.Context) { app.emitAppearance(modeValue) })
		app.appearanceItems[modeValue] = item
	}
	b.item(appearance, "settings", "Color Theme…", "", func(*application.Context) {
		app.emitSettingsOverlay()
	})
	m.AddSeparator()

	for _, option := range viewOptions(editor) {
		option := option
		item := m.AddCheckbox(option.label, option.checked)
		item.OnClick(func(ctx *application.Context) {
			app.emitToggleView(option.name, ctx.IsChecked())
		})
		app.viewItems[option.name] = item
	}
}

// buildMenu assembles the whole native menu. It is run at startup and again
// after keybindings.json changes, so custom bindings never leave a stale
// accelerator behind.
func buildMenu(
	wailsApp *application.App,
	app *App,
	editor EditorSettings,
	theme ThemeSettings,
	bindings []Keybinding,
) *application.Menu {
	app.viewItems = map[string]*application.MenuItem{}
	app.appearanceItems = map[string]*application.MenuItem{}

	builder := newMenuBuilder(app, bindings)
	menu := wailsApp.NewMenu()
	appMenu(menu.AddSubmenu(app.meta.ProductName), app.meta.ProductName, app, builder)
	fileMenu(menu.AddSubmenu("File"), app, builder)
	editMenu(menu.AddSubmenu("Edit"), app, builder)
	viewMenu(menu.AddSubmenu("View"), app, builder, editor, theme)
	return menu
}

// rebuildMenu replaces the native menu after a keybindings edit. The editor and
// theme sections are reread so checkmarks keep the values the frontend saved.
func (a *App) rebuildMenu() {
	if a.wails == nil {
		return
	}
	settings := a.currentSettings()
	a.wails.Menu.Set(buildMenu(a.wails, a, settings.Editor, settings.Theme, loadKeybindingsForMenu()))
}

// currentSettings returns settings.json content, or the defaults when it
// cannot be read.
func (a *App) currentSettings() Settings {
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
