package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kaze/internal/themes"
)

// themesDir is the per-user themes folder next to settings.json.
func themesDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(configDir, appMeta.ProductName, "themes"), nil
}

// ListThemes returns the installed variants, with per-file errors rather than
// one failure for the whole folder.
func (a *App) ListThemes() themes.ThemeCatalog {
	catalog := themes.ThemeCatalog{Themes: []themes.ThemeSummary{}, Errors: []string{}}
	dir, err := themesDir()
	if err != nil {
		// The built-in family is embedded, so an unwritable config directory
		// still leaves a usable set of themes.
		catalog.Errors = append(catalog.Errors, err.Error())
		dir = ""
	} else if err := themes.EnsureDir(dir); err != nil {
		catalog.Errors = append(catalog.Errors, err.Error())
	}
	entries, problems := themes.LoadEntries(dir)
	catalog.Themes = themes.CatalogFromEntries(entries)
	catalog.Errors = append(catalog.Errors, problems...)
	return catalog
}

// LoadTheme resolves one variant and returns the CSS and Monaco values that
// apply it. Missing style keys fall back to the matching built-in palette.
func (a *App) LoadTheme(name string) (themes.ThemePalette, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return themes.ThemePalette{}, errors.New("no theme name")
	}
	dir, err := themesDir()
	if err != nil {
		dir = ""
	} else {
		// A read-only config folder must not hide the embedded palettes.
		_ = themes.EnsureDir(dir)
	}
	entries, _ := themes.LoadEntries(dir)
	entry, ok := themes.FindVariant(entries, name)
	if !ok {
		return themes.ThemePalette{}, fmt.Errorf("theme %q is not installed", name)
	}
	return themes.PaletteFromEntry(entry), nil
}

// OpenThemesFolder reveals the user's themes directory, creating it with the
// built-in family when this is the first visit.
func (a *App) OpenThemesFolder() error {
	dir, err := themesDir()
	if err != nil {
		return err
	}
	if err := themes.EnsureDir(dir); err != nil {
		return err
	}
	return revealFile(dir)
}
