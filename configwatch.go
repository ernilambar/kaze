package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"kaze/internal/config"
	"kaze/internal/keybindings"
)

// configChangedEvent tells the frontend that the themes folder,
// keybindings.json or settings.json changed on disk. All are user-editable
// while the app runs.
const configChangedEvent = "app:config-changed"

// configChange says which of the live-reload inputs changed.
type configChange struct {
	Themes      bool `json:"themes"`
	Keybindings bool `json:"keybindings"`
	Settings    bool `json:"settings"`
}

// WatchConfig starts the config watcher. The first call wins; later calls are
// no-ops, so the frontend can call it on every mount safely.
func (a *App) WatchConfig() {
	a.configWatchOnce.Do(func() {
		go a.configWatchLoop()
	})
}

// configWatchLoop polls the themes folder, keybindings.json and settings.json. It is a second
// goroutine next to the workspace watcher and follows the same one-second
// cadence: theme files are tiny and only change when the user edits them.
func (a *App) configWatchLoop() {
	dir, err := themesDir()
	if err != nil {
		return
	}
	keyPath, err := keybindingsFilePath()
	if err != nil {
		return
	}
	settingsPath, err := settingsFilePath()
	if err != nil {
		return
	}
	// Create every input before taking the baseline, so first-run creation
	// does not look like a user edit.
	_ = ensureThemesDir(dir)
	_ = keybindings.EnsureFile(keyPath)
	_ = config.EnsureFile(settingsPath)

	themes := themeSnapshot(dir)
	keys := statWatch(keyPath)
	settings := statWatch(settingsPath)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()
	for range ticker.C {
		nextThemes := themeSnapshot(dir)
		nextKeys := statWatch(keyPath)
		nextSettings := statWatch(settingsPath)

		change := configChange{}
		if !sameWatchSnapshot(themes, nextThemes) {
			change.Themes = true
			themes = nextThemes
		}
		if nextKeys != keys {
			change.Keybindings = true
			keys = nextKeys
			// Rebuild the native menu only for a file that parses. A broken
			// file keeps the previous menu, matching the frontend's decision to
			// keep the previous bindings.
			if _, _, err := keybindings.Read(keyPath); err == nil {
				a.rebuildMenu()
			}
		}
		if nextSettings != settings {
			change.Settings = true
			settings = nextSettings
			// The View and Appearance checkmarks come from settings.json. A
			// broken file keeps the previous menu, as the frontend keeps the
			// previous settings.
			if _, err := config.Load(settingsPath); err == nil {
				a.rebuildMenu()
			}
		}
		if change.Themes || change.Keybindings || change.Settings {
			a.emitAll(configChangedEvent, change)
		}
	}
}

// themeSnapshot records every JSON file in the themes folder. A map rather than
// a directory mtime catches content edits, additions, deletions and renames.
func themeSnapshot(dir string) map[string]watchState {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return map[string]watchState{}
	}
	snapshot := make(map[string]watchState, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		snapshot[path] = statWatch(path)
	}
	return snapshot
}

func sameWatchSnapshot(a, b map[string]watchState) bool {
	if len(a) != len(b) {
		return false
	}
	for path, state := range a {
		if other, ok := b[path]; !ok || other != state {
			return false
		}
	}
	return true
}
