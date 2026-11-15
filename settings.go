package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultFontFamily = "Menlo"
	defaultFontSize   = 13.0
	defaultFontWeight = 400.0
)

// Settings is the contents of the user's settings.json file.
type Settings struct {
	Editor  EditorSettings  `json:"editor"`
	Sidebar SidebarSettings `json:"sidebar"`
}

// EditorSettings tunes the Monaco editor instance.
type EditorSettings struct {
	FontFamily string  `json:"font_family"`
	FontSize   float64 `json:"font_size"`
	FontWeight float64 `json:"font_weight"`
}

// SidebarSettings is the folder tree's saved state. A future workspace feature
// will replace it with a named project list.
type SidebarSettings struct {
	Folders []string `json:"folders"`
}

func defaultSettings() Settings {
	return Settings{
		Editor: EditorSettings{
			FontFamily: defaultFontFamily,
			FontSize:   defaultFontSize,
			FontWeight: defaultFontWeight,
		},
		Sidebar: SidebarSettings{Folders: []string{}},
	}
}

// normalize replaces absent or unusable values with their defaults. The
// settings file is user-editable, so a missing key must not produce a broken
// editor.
func (s *Settings) normalize() {
	s.Editor.FontFamily = strings.TrimSpace(s.Editor.FontFamily)
	if s.Editor.FontFamily == "" {
		s.Editor.FontFamily = defaultFontFamily
	}
	if s.Editor.FontSize <= 0 {
		s.Editor.FontSize = defaultFontSize
	}
	if s.Editor.FontWeight <= 0 {
		s.Editor.FontWeight = defaultFontWeight
	}

	s.Sidebar.Folders = normalizeFolderList(s.Sidebar.Folders)
}

// normalizeFolderList trims paths, drops blanks and removes repeats while
// keeping the user's order. The settings file may be edited by hand, so the
// list is repaired rather than rejected.
func normalizeFolderList(folders []string) []string {
	normalized := make([]string, 0, len(folders))
	for _, folder := range folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}
		normalized = append(normalized, folder)
	}
	return uniquePaths(normalized)
}

// settingsFilePath returns the per-user settings location, following the
// platform config directory: ~/Library/Application Support/Kova/settings.json
// on macOS.
func settingsFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "Kova", "settings.json"), nil
}

// loadSettings reads path and normalizes the result. When the file does not
// exist (or is empty) the defaults are written there so the user has something
// to edit.
func loadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(bytes.TrimSpace(data)) == 0) {
		settings := defaultSettings()
		if err := writeSettings(path, settings); err != nil {
			return settings, err
		}
		return settings, nil
	}
	if err != nil {
		return defaultSettings(), fmt.Errorf("read settings file: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), fmt.Errorf("parse %s: %w", path, err)
	}
	settings.normalize()
	return settings, nil
}

// ensureSettingsFile creates path with defaults when it is missing. Unlike
// loadSettings it does not fail on malformed content, so the user can still
// open and repair a broken file.
func ensureSettingsFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat settings file: %w", err)
	}
	return writeSettings(path, defaultSettings())
}

func writeSettings(path string, settings Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	data = append(data, '\n')

	// Write through a temporary file and rename, so a concurrent read never sees
	// a half-written settings file.
	tmp, err := os.CreateTemp(dir, "settings-*.json")
	if err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write settings file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	return nil
}
