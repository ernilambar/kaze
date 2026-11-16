package main

import (
	"errors"
	"strings"
)

// SessionState is the set of open tabs a relaunch should restore. Only tabs
// backed by a file are recorded; untitled buffers live only in memory.
type SessionState struct {
	OpenFiles []string `json:"open_files"`
	Active    string   `json:"active"`
}

// AddRecentFile puts path at the top of the Open Recent list, drops an older
// entry for the same file and returns the new list. The backend owns the list
// because it outlives the window.
func (a *App) AddRecentFile(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("no file to remember")
	}

	return a.updateRecentFiles(func(files []string) []string {
		recent := make([]string, 0, len(files)+1)
		recent = append(recent, path)
		for _, existing := range files {
			if existing != path {
				recent = append(recent, existing)
			}
		}
		return recent
	})
}

// ClearRecentFiles empties the Open Recent list and returns the result, which
// is now empty.
func (a *App) ClearRecentFiles() ([]string, error) {
	return a.updateRecentFiles(func([]string) []string { return nil })
}

// updateRecentFiles reads settings.json, applies change to the recent list and
// writes the result back. The list is normalized on both sides so an entry
// trimmed here is the one the menu will show.
func (a *App) updateRecentFiles(change func([]string) []string) ([]string, error) {
	path, err := settingsFilePath()
	if err != nil {
		return nil, err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return settings.Recent.Files, err
	}
	before := settings.Recent
	settings.Recent.Files = change(settings.Recent.Files)
	settings.normalize()
	if err := writeSettingsSections(path, settings, settingsSection{"recent", before, settings.Recent}); err != nil {
		return settings.Recent.Files, err
	}
	return settings.Recent.Files, nil
}

// SetSession records the tabs that should reopen on the next launch. The
// frontend calls it whenever the tab list or the active tab changes.
func (a *App) SetSession(session SessionState) error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return err
	}
	before := settings.Session
	settings.Session.OpenFiles = session.OpenFiles
	settings.Session.Active = session.Active
	settings.normalize()
	return writeSettingsSections(path, settings, settingsSection{"session", before, settings.Session})
}

// SetSidebarVisible records whether the folder tree is showing, so Cmd+B keeps
// the layout the user left behind.
func (a *App) SetSidebarVisible(visible bool) error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return err
	}
	before := settings.Sidebar
	settings.Sidebar.Visible = visible
	return writeSettingsSections(path, settings, settingsSection{"sidebar", before, settings.Sidebar})
}

// SetSidebarAutoReveal records whether switching tabs should expand the tree
// to the active file.
func (a *App) SetSidebarAutoReveal(autoReveal bool) error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	settings, err := loadSettings(path)
	if err != nil {
		return err
	}
	before := settings.Sidebar
	settings.Sidebar.AutoReveal = autoReveal
	return writeSettingsSections(path, settings, settingsSection{"sidebar", before, settings.Sidebar})
}

// RemoveAllSidebarFolders drops every root from the sidebar and returns the
// empty list. Open tabs stay open: they hold their own buffers.
func (a *App) RemoveAllSidebarFolders() ([]string, error) {
	return a.updateSidebarFolders(func([]string) []string { return nil })
}

// CopyText puts text on the system clipboard. A menu click has no caller to
// return to, so the command palette path reports the error and the menu logs
// it.
func (a *App) CopyText(text string) error {
	if a.wails == nil {
		return errors.New("clipboard is not available before startup")
	}
	if !a.wails.Clipboard.SetText(text) {
		return errors.New("the system clipboard is not available")
	}
	return nil
}

// RevealFile shows path in the platform's file manager, selecting it when the
// platform supports that.
func (a *App) RevealFile(path string) error {
	return revealFile(path)
}
