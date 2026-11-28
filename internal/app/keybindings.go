package app

import (
	"maps"
	"path/filepath"
	"strings"

	"kaze/internal/keybindings"
)

// keybindingsFilePath is the per-user keybindings file next to settings.json.
func keybindingsFilePath() (string, error) {
	path, err := settingsFilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "keybindings.json"), nil
}

// GetKeybindings returns the accepted bindings and any per-entry problems. The
// frontend installs the list and reports the error without losing the valid
// entries.
func (a *App) GetKeybindings() keybindings.KeybindingsState {
	path, err := keybindingsFilePath()
	if err != nil {
		return keybindings.KeybindingsState{Bindings: []keybindings.Keybinding{}, Error: err.Error(), Parsed: false}
	}
	bindings, problems, err := keybindings.Read(path)
	if err != nil {
		return keybindings.KeybindingsState{Bindings: []keybindings.Keybinding{}, Error: err.Error(), Parsed: false}
	}
	return keybindings.KeybindingsState{
		Bindings: bindings,
		Error:    strings.Join(problems, "; "),
		Parsed:   true,
	}
}

// MenuAccelerators returns the native menu's accelerators by command id, as
// the menu was last built. Context menus show them beside matching items.
func (a *App) MenuAccelerators() map[string]string {
	a.menuItemsMu.Lock()
	defer a.menuItemsMu.Unlock()
	return maps.Clone(a.menuAccelerators)
}

// KeybindingsFile returns the path of keybindings.json for the frontend to
// open in a tab, creating the commented template on first use.
func (a *App) KeybindingsFile() (string, error) {
	path, err := keybindingsFilePath()
	if err != nil {
		return "", err
	}
	if err := keybindings.EnsureFile(path); err != nil {
		return "", err
	}
	return path, nil
}

// loadKeybindingsForMenu is the menu's read path. A malformed file leaves the
// native defaults in place; the frontend reports the parse error.
func loadKeybindingsForMenu() []keybindings.Keybinding {
	path, err := keybindingsFilePath()
	if err != nil {
		return nil
	}
	bindings, _, err := keybindings.Read(path)
	if err != nil {
		return nil
	}
	return bindings
}
