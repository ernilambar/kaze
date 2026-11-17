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

	// windowsMu guards the open windows and the pre-window launch queue. Each
	// window has its own session, because tabs, dirty flags and pending launch
	// paths belong to one window and must never leak into another.
	windowsMu sync.Mutex
	windows   map[string]*windowSession
	order     []string
	// pending holds a launch request that arrived before any window existed.
	pending LaunchRequest
	// windowSeq names windows main, window-2, window-3, … so a frontend can
	// address its own session and menu commands can find the focused one.
	windowSeq atomic.Int64

	// allowQuit records that the user answered a close or quit prompt with "go
	// ahead". It lets the termination that follows through without a second
	// prompt; opening a window clears it again.
	allowQuit atomic.Bool
	// quitInProgress keeps Cmd+Q from stacking a second prompt while the
	// windows with unsaved buffers are being asked one at a time.
	quitInProgress atomic.Bool

	// settingsMu serializes load-modify-write cycles on settings.json. Two
	// folder changes at once would otherwise read the same list and one of them
	// would be lost.
	settingsMu sync.Mutex

	// sessionMu serializes session.json, which every window writes its own
	// workspace tab set into. sessionLoaded makes the file's first read happen
	// once; the in-memory copy is then updated under this lock.
	sessionMu     sync.Mutex
	sessionLoaded bool
	sessionStore  sessionFile

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
		windows:         map[string]*windowSession{},
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
	before := settings.Editor
	settings.Editor = editor
	settings.normalize()
	if err := writeSettingsSections(path, settings, settingsSection{"editor", before, settings.Editor}); err != nil {
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
	before := settings.Theme
	settings.Theme = theme
	settings.normalize()
	if err := writeSettingsSections(path, settings, settingsSection{"theme", before, settings.Theme}); err != nil {
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
	before := settings.UI
	settings.UI = ui
	settings.normalize()
	if err := writeSettingsSections(path, settings, settingsSection{"ui", before, settings.UI}); err != nil {
		return settings.UI, err
	}
	return settings.UI, nil
}

// UpdateFilesSettings replaces the files section of settings.json and returns
// the normalized result. The frontend calls it when the explorer's exclude
// patterns change, so the tree hides the same entries the next read does.
func (a *App) UpdateFilesSettings(files FilesSettings) (FilesSettings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return files, err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return files, err
	}
	before := settings.Files
	settings.Files = files
	settings.normalize()
	if err := writeSettingsSections(path, settings, settingsSection{"files", before, settings.Files}); err != nil {
		return settings.Files, err
	}
	return settings.Files, nil
}

// UpdateSearchSettings replaces the search section of settings.json and
// returns the normalized result. Find in Files applies both lists, so this
// only writes the patterns the search window owns.
func (a *App) UpdateSearchSettings(search SearchSettings) (SearchSettings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return search, err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return search, err
	}
	before := settings.Search
	settings.Search = search
	settings.normalize()
	if err := writeSettingsSections(path, settings, settingsSection{"search", before, settings.Search}); err != nil {
		return settings.Search, err
	}
	return settings.Search, nil
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
// `<command> .` twice does not duplicate the root. The path is checked here rather
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

	before := settings.Sidebar
	settings.Sidebar.Folders = normalizeFolderList(change(settings.Sidebar.Folders))
	if err := writeSettingsSections(path, settings, settingsSection{"sidebar", before, settings.Sidebar}); err != nil {
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

// OpenDefaultSettings writes the defaults the app would use on a fresh install to
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

// addFolderEvent asks the sidebar to open the folder picker. A menu click has no
// caller to return to, so the frontend runs the same code path as its own
// button: one busy state, one place errors are reported.
const addFolderEvent = "app:add-folder"

// emitAddFolder tells the frontend to open the folder picker.
func (a *App) emitAddFolder() {
	a.emitEvent(addFolderEvent)
}

// emitEvent sends one menu command to the focused window. The native menu is
// shared by every window, so the command belongs to the one the user is
// looking at. Before main has attached the application there is no one to tell,
// so it is a no-op.
func (a *App) emitEvent(name string) {
	if session := a.focusedSession(); session != nil {
		session.emit(name)
	}
}

// emitEventWith sends a menu command that carries a value to the focused
// window.
func (a *App) emitEventWith(name string, data any) {
	if session := a.focusedSession(); session != nil {
		session.emit(name, data)
	}
}

// emitAll sends an event every window must see, because it changes app-wide
// state such as settings, themes or a View menu toggle. A Wails application
// event carries no sender, which the frontend's listener accepts.
func (a *App) emitAll(name string, data ...any) {
	if a.wails != nil {
		a.wails.Event.Emit(name, data...)
	}
}

// closeTabEvent asks the frontend to close the active tab. The tab list lives in
// the frontend, so the menu only sends the command.
const closeTabEvent = "app:close-tab"

// moveTabToNewWindowEvent asks the frontend to hand its active tab to a new
// window. Only the frontend can read the buffer that moves.
const moveTabToNewWindowEvent = "app:move-tab-to-new-window"

// emitMoveTabToNewWindow tells the focused window to move its active tab.
func (a *App) emitMoveTabToNewWindow() { a.emitEvent(moveTabToNewWindowEvent) }

// The navigation commands work on the frontend's tab list. nextTabEvent and
// previousTabEvent step through it; goToTabEvent carries the 1-based position.
const (
	nextTabEvent        = "app:next-tab"
	previousTabEvent    = "app:previous-tab"
	reopenTabEvent      = "app:reopen-closed-tab"
	goToTabEvent        = "app:go-to-tab"
	openRecentEvent     = "app:open-recent"
	clearRecentEvent    = "app:clear-recent"
	openWorkspaceEvent  = "app:open-workspace"
	saveWorkspaceEvent  = "app:save-workspace-as"
	closeWorkspaceEvent = "app:close-workspace"
	refreshEvent        = "app:refresh"
	findInFilesEvent    = "app:find-in-files"
	toggleSidebarEvent  = "app:toggle-sidebar"
	closeFoldersEvent   = "app:close-all-folders"
	revealFileEvent     = "app:reveal-file"
	copyPathEvent       = "app:copy-path"
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
const quickOpenEvent = "app:quick-open"

// emitQuickOpen tells the frontend to show Quick Open.
func (a *App) emitQuickOpen() {
	a.emitEvent(quickOpenEvent)
}

// commandPaletteEvent asks the frontend to show the command palette. Commands
// are frontend state, so the menu only sends the command.
const commandPaletteEvent = "app:command-palette"

// emitCommandPalette tells the frontend to show the command palette.
func (a *App) emitCommandPalette() {
	a.emitEvent(commandPaletteEvent)
}

// The file menu's commands are frontend state: buffers, their dirty flags and
// their paths all live in the UI, so Go only forwards each click. One name per
// command keeps the menu and the App listener honest about what exists.
const (
	newFileEvent   = "app:new-file"
	newFolderEvent = "app:new-folder"
	newBufferEvent = "app:new-buffer"
	openFileEvent  = "app:open-file"
	saveEvent      = "app:save"
	saveAsEvent    = "app:save-as"
	saveAllEvent   = "app:save-all"
	printEvent     = "app:print"
)

// emitNewFile asks the tree for a New File, which creates the file inside a
// sidebar folder. The frontend picks the folder, because the sidebar roots are
// its state.
func (a *App) emitNewFile() { a.emitEvent(newFileEvent) }

// emitNewFolder asks the tree to create a folder in the selected folder.
func (a *App) emitNewFolder() { a.emitEvent(newFolderEvent) }

// emitNewBuffer tells the frontend to open a new untitled buffer.
func (a *App) emitNewBuffer() { a.emitEvent(newBufferEvent) }

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
const closeRequestEvent = "app:confirm-close"

// The Edit, Selection and View menus are frontend state too: the transforms,
// selection actions, find, preview and zoom all act on Monaco buffers, so Go
// forwards each one.
const (
	transformCaseEvent      = "app:transform-case"
	trimWhitespaceEvent     = "app:trim-whitespace"
	selectionCommandEvent   = "app:selection-command"
	findEvent               = "app:find"
	replaceEvent            = "app:replace"
	goToLineEvent           = "app:go-to-line"
	findNextResultEvent     = "app:find-next-result"
	findPreviousResultEvent = "app:find-previous-result"
	zoomEvent               = "app:zoom"
	togglePreviewEvent      = "app:toggle-preview"
	toggleViewEvent         = "app:toggle-view"
)

// emitTransformCase tells the frontend to convert the selection to id's case.
func (a *App) emitTransformCase(id string) { a.emitEventWith(transformCaseEvent, id) }

// emitTrimWhitespace tells the frontend to trim the active buffer's trailing
// whitespace.
func (a *App) emitTrimWhitespace() { a.emitEvent(trimWhitespaceEvent) }

// emitSelectionCommand tells the frontend to run one Selection menu command on
// the active buffer.
func (a *App) emitSelectionCommand(command string) {
	a.emitEventWith(selectionCommandEvent, command)
}

// emitFind tells the frontend to open Monaco's find widget on the active
// buffer.
func (a *App) emitFind() { a.emitEvent(findEvent) }

// emitReplace tells the frontend to open Monaco's find widget in replace mode.
func (a *App) emitReplace() { a.emitEvent(replaceEvent) }

// emitGoToLine tells the frontend to open Monaco's Go to Line prompt.
func (a *App) emitGoToLine() { a.emitEvent(goToLineEvent) }

// emitFindNextResult and emitFindPreviousResult step through the Find in Files
// results the focused window holds.
func (a *App) emitFindNextResult() { a.emitEvent(findNextResultEvent) }

func (a *App) emitFindPreviousResult() { a.emitEvent(findPreviousResultEvent) }

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

// emitToggleView tells every window that option's checkmark moved. Editor
// settings are shared, so a View menu toggle applies to all of them.
func (a *App) emitToggleView(option string, checked bool) {
	a.emitAll(toggleViewEvent, ViewToggle{Option: option, Checked: checked})
}

// settingsOverlayEvent asks the frontend to open the settings overlay.
const settingsOverlayEvent = "app:settings-overlay"

// emitSettingsOverlay tells the frontend to show the settings overlay.
func (a *App) emitSettingsOverlay() { a.emitEvent(settingsOverlayEvent) }

// appearanceEvent carries the mode the Appearance submenu selected.
const appearanceEvent = "app:appearance"

// emitAppearance tells every window that the menu selected a new mode, because
// the palette is shared by all of them.
func (a *App) emitAppearance(mode string) { a.emitAll(appearanceEvent, mode) }

// ShouldRestoreSession reports whether this window owns the saved tab session.
// It is read before the window collects its launch request, so the first window
// can restore the session while later windows start empty. It does not mark the
// window as listening, so paths arriving before the listener is ready still
// wait in the session.
func (a *App) ShouldRestoreSession(window string) bool {
	session := a.session(window)
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.pending.RestoreSession
}

// GetProductName is the app name from build/config.yml, so the UI does not
// carry its own copy.
func (a *App) GetProductName() string {
	return a.meta.ProductName
}

// GetLaunchRequest hands one window the paths it was started with. The
// frontend calls it once after attaching its listener, so anything passed in
// before the UI was ready is still opened.
func (a *App) GetLaunchRequest(window string) LaunchRequest {
	session := a.session(window)
	if session == nil {
		return emptyLaunchRequest()
	}
	return session.takeLaunch()
}

// queueLaunch records what a command line asked to open and, when the focused
// window is already listening, hands it over immediately.
func (a *App) queueLaunch(request LaunchRequest) {
	a.queueLaunchTo(a.targetSession(), request)
}

// queueLaunchTo records a request for one window. The launch path picks the
// focused window; a file drop passes the window it happened in, so dropping on
// a background window does not open the files somewhere else. A request marked
// NewWindow opens a window of its own instead of reusing the target; the flag
// is consumed here, so no frontend has to know about it.
func (a *App) queueLaunchTo(session *windowSession, request LaunchRequest) {
	if request.NewWindow {
		request.NewWindow = false
		// Before the runtime is attached there is nothing to open a window in:
		// the startup window is already the new window this launch asked for,
		// so its paths are queued and it adopts them.
		if a.wails != nil {
			if err := a.openWindow(request); err == nil {
				return
			}
		}
	}
	if request.Empty() {
		return
	}

	if session == nil {
		a.windowsMu.Lock()
		a.pending.Merge(request)
		a.windowsMu.Unlock()
		return
	}

	payload, ready := session.queueLaunch(request)
	if !ready {
		return
	}
	// A second launch should surface the window, not leave files open behind
	// another app.
	session.show()
	session.emit(launchEvent, payload)
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

// InstallCommandLineTool installs the command line launcher in /usr/local/bin, asking
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
		SetMessage("Installed " + path + ".\n\nOpen a new terminal and run:\n\n  " + commandName + " notes.txt\n  " + commandName + " .").
		Show()
}
