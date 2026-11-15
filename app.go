package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the application struct. Exported methods become callable from the
// frontend through the generated Wails bindings.
type App struct {
	// wails is the running application, set once main has created it. Menu
	// callbacks and dialogs go through it; tests call the exported methods
	// without one, so every use is nil-checked.
	wails *application.App
	// window is the single webview window, used to surface the app when a
	// second `kova <file>` arrives.
	window *application.WebviewWindow

	mu sync.Mutex
	// pending holds a launch request the frontend has not collected yet. It is
	// kept here because a second `kova file.txt` can arrive while the window is
	// still starting up, before anything is listening for the event.
	pending LaunchRequest
	// listening is set once the frontend has attached its launch listener, so
	// later requests can be emitted as events instead of queued.
	listening bool

	// settingsMu serializes load-modify-write cycles on settings.json. Two
	// folder changes at once would otherwise read the same list and one of them
	// would be lost.
	settingsMu sync.Mutex

	// viewItems are the View menu's checkable items by editor option name. They
	// are rebuilt with the menu and let a toggle made from the command palette
	// move the menu's checkmark to match.
	viewItems map[string]*application.MenuItem
	// appearanceItems are the Appearance submenu's radio items by mode, so a
	// change made in the settings overlay moves the menu's checkmark.
	appearanceItems map[string]*application.MenuItem
	// meta is the bundle metadata, kept so a menu rebuild after a keybindings
	// edit can use the same product name.
	meta appMetadata

	// unsaved is set by the frontend whenever any buffer holds edits that are
	// not on disk. Closing the window or quitting asks the frontend first, so
	// the guard can reject the request and let the user decide.
	unsaved atomic.Bool
	// confirmed records that the frontend answered a close request with "go
	// ahead", so the guard stands aside for the quit it asked for.
	confirmed atomic.Bool
	// closeRequested keeps a second Cmd+Q or a close click from stacking a
	// second prompt on the one the frontend is already showing.
	closeRequested atomic.Bool

	// watcher polls the open buffers and expanded folders for external
	// changes; watchOnce starts its loop the first time the frontend registers
	// a path.
	watcher   fileWatcher
	watchOnce sync.Once

	// configWatchOnce starts the themes/keybindings watcher on the frontend's
	// first call.
	configWatchOnce sync.Once
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		pending:         emptyLaunchRequest(),
		viewItems:       map[string]*application.MenuItem{},
		appearanceItems: map[string]*application.MenuItem{},
	}
}

// attach records the running application so dialogs, events and window calls
// have somewhere to go. Main calls it before the app runs; unit tests never do.
func (a *App) attach(wails *application.App) {
	a.wails = wails
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

// UpdateEditorSettings replaces the editor section of settings.json and returns
// the normalized result. The frontend calls it whenever a view toggle changes,
// so the next launch starts with the same editor.
func (a *App) UpdateEditorSettings(editor EditorSettings) (EditorSettings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return editor, err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return editor, err
	}
	settings.Editor = editor
	settings.normalize()
	if err := writeSettings(path, settings); err != nil {
		return settings.Editor, err
	}
	return settings.Editor, nil
}

// UpdateThemeSettings replaces the theme section of settings.json and returns
// the normalized result. The frontend calls it when the appearance mode or one
// of the selected palettes changes.
func (a *App) UpdateThemeSettings(theme ThemeSettings) (ThemeSettings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return theme, err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return theme, err
	}
	settings.Theme = theme
	settings.normalize()
	if err := writeSettings(path, settings); err != nil {
		return settings.Theme, err
	}
	return settings.Theme, nil
}

// UpdateUISettings replaces the interface section of settings.json and returns
// the normalized result. Typography, icon style and sidebar side all travel
// through this one write path.
func (a *App) UpdateUISettings(ui UISettings) (UISettings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return ui, err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return ui, err
	}
	settings.UI = ui
	settings.normalize()
	if err := writeSettings(path, settings); err != nil {
		return settings.UI, err
	}
	return settings.UI, nil
}

// SetAppearanceMode moves the Appearance submenu's radio checkmark when the
// mode changed somewhere other than the native menu, such as the settings
// overlay or a custom keybinding.
func (a *App) SetAppearanceMode(mode string) {
	mode = normalizeThemeMode(mode)
	for id, item := range a.appearanceItems {
		item.SetChecked(id == mode)
	}
}

// SetViewOption moves one View menu checkmark. The frontend owns the editor
// options, so a toggle that runs from the command palette reports its new value
// here; the menu's own click already moved the checkmark natively.
func (a *App) SetViewOption(option string, checked bool) {
	if item, ok := a.viewItems[option]; ok {
		item.SetChecked(checked)
	}
}

// AddSidebarFolder adds one folder to the sidebar's saved list and returns the
// new list. Opening a folder the user already added is a no-op, so launching
// `kova .` twice does not duplicate the root. The path is checked here rather
// than trusted from the caller, because a sidebar root is only useful while it
// is a real folder.
func (a *App) AddSidebarFolder(folder string) ([]string, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return nil, errors.New("no folder to add")
	}
	info, err := os.Stat(folder)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", filepath.Base(folder))
	}

	return a.updateSidebarFolders(func(folders []string) []string {
		return append(folders, folder)
	})
}

// SetSidebarFolders replaces the sidebar roots in one write. Opening a
// workspace uses it so a half-applied list never survives a failed folder.
func (a *App) SetSidebarFolders(folders []string) ([]string, error) {
	normalized := normalizeFolderList(folders)
	for _, folder := range normalized {
		info, err := os.Stat(folder)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a folder", filepath.Base(folder))
		}
	}
	return a.updateSidebarFolders(func([]string) []string { return normalized })
}

// RemoveSidebarFolder drops one folder from the sidebar's saved list and
// returns the new list.
func (a *App) RemoveSidebarFolder(folder string) ([]string, error) {
	return a.updateSidebarFolders(func(folders []string) []string {
		kept := make([]string, 0, len(folders))
		for _, existing := range folders {
			if existing != folder {
				kept = append(kept, existing)
			}
		}
		return kept
	})
}

// updateSidebarFolders reads settings.json, applies change to the folder list
// and writes the result back. The list is normalized on both sides, so the
// backend and the frontend agree on what a valid root is.
func (a *App) updateSidebarFolders(change func([]string) []string) ([]string, error) {
	path, err := settingsFilePath()
	if err != nil {
		return nil, err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return settings.Sidebar.Folders, err
	}

	settings.Sidebar.Folders = normalizeFolderList(change(settings.Sidebar.Folders))
	if err := writeSettings(path, settings); err != nil {
		return settings.Sidebar.Folders, err
	}
	return settings.Sidebar.Folders, nil
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

// OpenSettings opens settings.json in the editor named by the user's
// environment, falling back to the platform's default file handler. The command
// palette reports the error; the menu logs it.
func (a *App) OpenSettings() error {
	return a.openSettingsFile()
}

// OpenDefaultSettings writes the defaults Kova would use on a fresh install to
// default-settings.json and opens it. It never touches the user's settings.json.
func (a *App) OpenDefaultSettings() error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}
	defaultPath := filepath.Join(filepath.Dir(path), "default-settings.json")
	if err := writeSettings(defaultPath, defaultSettings()); err != nil {
		return err
	}
	return openInEditor(defaultPath)
}

// openSettingsFromMenu opens settings.json from the app menu. A menu click has
// no caller to return to, so failures are logged.
func (a *App) openSettingsFromMenu() {
	if err := a.OpenSettings(); err != nil && a.wails != nil {
		a.wails.Logger.Error(err.Error())
	}
}

// SetUnsavedChanges records whether any open buffer has edits that are not on
// disk yet. The frontend pushes it whenever that answer changes, so a close or
// quit can tell whether it must ask before throwing work away.
func (a *App) SetUnsavedChanges(unsaved bool) {
	a.unsaved.Store(unsaved)
}

// allowClose reports whether the app may go away without asking the frontend.
func (a *App) allowClose() bool {
	return !a.unsaved.Load() || a.confirmed.Load()
}

// shouldQuit answers macOS's terminate request. Unsaved buffers cancel the
// quit and hand the decision to the frontend, which shows one prompt and then
// answers with ResolveClose.
func (a *App) shouldQuit() bool {
	if a.allowClose() {
		return true
	}
	a.requestClose()
	return false
}

// requestClose asks the frontend to confirm before unsaved edits are lost.
func (a *App) requestClose() {
	if a.wails == nil || !a.closeRequested.CompareAndSwap(false, true) {
		return
	}
	a.emitCloseRequest()
}

// ResolveClose answers a close request. Cancelling clears the request so the
// next Cmd+Q asks again; quitting records the answer and terminates.
func (a *App) ResolveClose(quit bool) {
	a.closeRequested.Store(false)
	if !quit {
		return
	}
	a.confirmed.Store(true)
	a.quit()
}

// quit shuts the app down so its shutdown path runs.
func (a *App) quit() {
	if a.wails != nil {
		a.wails.Quit()
	}
}

// addFolderEvent asks the sidebar to open the folder picker. A menu click has no
// caller to return to, so the frontend runs the same code path as its own
// button: one busy state, one place errors are reported.
const addFolderEvent = "kova:add-folder"

// emitAddFolder tells the frontend to open the folder picker.
func (a *App) emitAddFolder() {
	a.emitEvent(addFolderEvent)
}

// emitEvent sends one menu command to the frontend. Before main has attached
// the application there is no one to tell, so it is a no-op.
func (a *App) emitEvent(name string) {
	if a.wails != nil {
		a.wails.Event.Emit(name)
	}
}

// emitEventWith sends a menu command that carries a value: a case transform, a
// zoom direction, or a View menu toggle's new state.
func (a *App) emitEventWith(name string, data any) {
	if a.wails != nil {
		a.wails.Event.Emit(name, data)
	}
}

// closeTabEvent asks the frontend to close the active tab. The tab list lives in
// the frontend, so the menu only sends the command.
const closeTabEvent = "kova:close-tab"

// The navigation commands work on the frontend's tab list. nextTabEvent and
// previousTabEvent step through it; goToTabEvent carries the 1-based position.
const (
	nextTabEvent        = "kova:next-tab"
	previousTabEvent    = "kova:previous-tab"
	reopenTabEvent      = "kova:reopen-closed-tab"
	goToTabEvent        = "kova:go-to-tab"
	openRecentEvent     = "kova:open-recent"
	clearRecentEvent    = "kova:clear-recent"
	openWorkspaceEvent  = "kova:open-workspace"
	saveWorkspaceEvent  = "kova:save-workspace-as"
	closeWorkspaceEvent = "kova:close-workspace"
	refreshEvent        = "kova:refresh"
	findInFilesEvent    = "kova:find-in-files"
	toggleSidebarEvent  = "kova:toggle-sidebar"
	closeFoldersEvent   = "kova:close-all-folders"
	revealFileEvent     = "kova:reveal-file"
	copyPathEvent       = "kova:copy-path"
)

// emitNextTab tells the frontend to activate the next tab.
func (a *App) emitNextTab() { a.emitEvent(nextTabEvent) }

// emitPreviousTab tells the frontend to activate the previous tab.
func (a *App) emitPreviousTab() { a.emitEvent(previousTabEvent) }

// emitReopenClosedTab tells the frontend to restore the last closed tab.
func (a *App) emitReopenClosedTab() { a.emitEvent(reopenTabEvent) }

// emitGoToTab tells the frontend to activate the tab at a 1-based position.
func (a *App) emitGoToTab(index int) { a.emitEventWith(goToTabEvent, index) }

// emitOpenRecent tells the frontend to show the recent files palette.
func (a *App) emitOpenRecent() { a.emitEvent(openRecentEvent) }

// emitClearRecent tells the frontend to empty the recent files list.
func (a *App) emitClearRecent() { a.emitEvent(clearRecentEvent) }

// emitOpenWorkspace tells the frontend to show the workspace picker.
func (a *App) emitOpenWorkspace() { a.emitEvent(openWorkspaceEvent) }

// emitSaveWorkspaceAs tells the frontend to save the current roots as a
// workspace file.
func (a *App) emitSaveWorkspaceAs() { a.emitEvent(saveWorkspaceEvent) }

// emitCloseWorkspace tells the frontend to drop the current workspace.
func (a *App) emitCloseWorkspace() { a.emitEvent(closeWorkspaceEvent) }

// emitRefresh tells the frontend to reread the tree and clean buffers.
func (a *App) emitRefresh() { a.emitEvent(refreshEvent) }

// emitFindInFiles tells the frontend to show the project search panel.
func (a *App) emitFindInFiles() { a.emitEvent(findInFilesEvent) }

// emitToggleSidebar tells the frontend to show or hide the folder tree.
func (a *App) emitToggleSidebar() { a.emitEvent(toggleSidebarEvent) }

// emitCloseAllFolders tells the frontend to drop every sidebar root.
func (a *App) emitCloseAllFolders() { a.emitEvent(closeFoldersEvent) }

// emitRevealFile tells the frontend to reveal the active file. The path is
// frontend state, so only the command travels.
func (a *App) emitRevealFile() { a.emitEvent(revealFileEvent) }

// emitCopyPath tells the frontend to copy the active file's path in the form
// named by kind: "file", "relative" or "name".
func (a *App) emitCopyPath(kind string) { a.emitEventWith(copyPathEvent, kind) }

// emitCloseTab tells the frontend to close the active tab.
func (a *App) emitCloseTab() {
	a.emitEvent(closeTabEvent)
}

// quickOpenEvent asks the frontend to show the Quick Open palette. The file
// list lives behind the sidebar roots, so the frontend owns the dialog and the
// menu only sends the command.
const quickOpenEvent = "kova:quick-open"

// emitQuickOpen tells the frontend to show Quick Open.
func (a *App) emitQuickOpen() {
	a.emitEvent(quickOpenEvent)
}

// commandPaletteEvent asks the frontend to show the command palette. Commands
// are frontend state, so the menu only sends the command.
const commandPaletteEvent = "kova:command-palette"

// emitCommandPalette tells the frontend to show the command palette.
func (a *App) emitCommandPalette() {
	a.emitEvent(commandPaletteEvent)
}

// The file menu's commands are frontend state: buffers, their dirty flags and
// their paths all live in the UI, so Go only forwards each click. One name per
// command keeps the menu and the App listener honest about what exists.
const (
	newFileEvent  = "kova:new-file"
	openFileEvent = "kova:open-file"
	saveEvent     = "kova:save"
	saveAsEvent   = "kova:save-as"
	saveAllEvent  = "kova:save-all"
	printEvent    = "kova:print"
)

// emitNewFile tells the frontend to open a new untitled buffer.
func (a *App) emitNewFile() { a.emitEvent(newFileEvent) }

// emitOpenFile tells the frontend to show the Open dialog.
func (a *App) emitOpenFile() { a.emitEvent(openFileEvent) }

// emitSave tells the frontend to save the active buffer.
func (a *App) emitSave() { a.emitEvent(saveEvent) }

// emitSaveAs tells the frontend to save the active buffer under a new path.
func (a *App) emitSaveAs() { a.emitEvent(saveAsEvent) }

// emitSaveAll tells the frontend to save every modified buffer.
func (a *App) emitSaveAll() { a.emitEvent(saveAllEvent) }

// emitPrint tells the frontend to print the active buffer.
func (a *App) emitPrint() { a.emitEvent(printEvent) }

// closeRequestEvent asks the frontend to deal with unsaved buffers before the
// window closes. The prompt is the frontend's, because only it can save them.
const closeRequestEvent = "kova:confirm-close"

// emitCloseRequest tells the frontend to confirm losing unsaved edits.
func (a *App) emitCloseRequest() { a.emitEvent(closeRequestEvent) }

// The Edit and View menus are frontend state too: the transform, whitespace,
// preview and zoom commands all act on Monaco buffers, so Go forwards each one.
const (
	transformCaseEvent  = "kova:transform-case"
	trimWhitespaceEvent = "kova:trim-whitespace"
	zoomEvent           = "kova:zoom"
	togglePreviewEvent  = "kova:toggle-preview"
	toggleViewEvent     = "kova:toggle-view"
)

// emitTransformCase tells the frontend to convert the selection to id's case.
func (a *App) emitTransformCase(id string) { a.emitEventWith(transformCaseEvent, id) }

// emitTrimWhitespace tells the frontend to trim the active buffer's trailing
// whitespace.
func (a *App) emitTrimWhitespace() { a.emitEvent(trimWhitespaceEvent) }

// emitZoom tells the frontend to change the editor font size by command.
func (a *App) emitZoom(command string) { a.emitEventWith(zoomEvent, command) }

// emitTogglePreview tells the frontend to show or hide the Markdown or SVG
// preview of the active buffer.
func (a *App) emitTogglePreview() { a.emitEvent(togglePreviewEvent) }

// ViewToggle is a View menu checkbox click: the editor option the frontend
// should apply and the checkmark's new state.
type ViewToggle struct {
	Option  string `json:"option"`
	Checked bool   `json:"checked"`
}

// emitToggleView tells the frontend that option's checkmark moved.
func (a *App) emitToggleView(option string, checked bool) {
	a.emitEventWith(toggleViewEvent, ViewToggle{Option: option, Checked: checked})
}

// settingsOverlayEvent asks the frontend to open the settings overlay.
const settingsOverlayEvent = "kova:settings-overlay"

// emitSettingsOverlay tells the frontend to show the settings overlay.
func (a *App) emitSettingsOverlay() { a.emitEvent(settingsOverlayEvent) }

// appearanceEvent carries the mode the Appearance submenu selected.
const appearanceEvent = "kova:appearance"

// emitAppearance tells the frontend that the menu selected a new mode.
func (a *App) emitAppearance(mode string) { a.emitEventWith(appearanceEvent, mode) }

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
	if a.window != nil {
		a.window.UnMinimise()
		a.window.Show()
	}
	if a.wails != nil {
		a.wails.Event.Emit(launchEvent, payload)
	}
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
// for an administrator password when the directory is not writable. It reports
// the outcome with dialogs, because the app menu and the command palette have no
// caller to return to.
func (a *App) InstallCommandLineTool() {
	err := installCommandLineTool()
	if err != nil {
		if a.wails != nil {
			a.wails.Logger.Error(err.Error())
			a.wails.Dialog.Error().
				SetTitle("Install Command Line Tool").
				SetMessage(err.Error()).
				Show()
		}
		return
	}

	path := commandInstallPath()
	if a.wails == nil {
		return
	}
	a.wails.Dialog.Info().
		SetTitle("Install Command Line Tool").
		SetMessage("Installed " + path + ".\n\nOpen a new terminal and run:\n\n  kova notes.txt\n  kova .").
		Show()
}
