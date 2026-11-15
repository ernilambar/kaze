package main

import (
	"io/fs"
	"path/filepath"
	"sort"
)

// ListFiles returns every regular file under the sidebar roots, sorted by
// path. The frontend filters this list locally, so each keystroke in Quick Open
// costs nothing. files.exclude decides what is walked, so a folder hidden in
// the tree is hidden here too.
func (a *App) ListFiles() ([]string, error) {
	path, err := settingsFilePath()
	if err != nil {
		return nil, err
	}
	settings, err := loadSettings(path)
	if err != nil {
		return nil, err
	}
	return listFilesExcluding(settings.Sidebar.Folders, newExcludeMatcher(settings.Files.Exclude)), nil
}

// listFilesExcluding walks each root and skips excluded trees, unreadable
// directories and symlinks. A stale root is ignored instead of failing the
// whole list, so one folder moved outside the app cannot empty the palette.
func listFilesExcluding(roots []string, excludes excludeMatcher) []string {
	files := make([]string, 0)
	seen := make(map[string]bool)

	for _, root := range roots {
		root = filepath.Clean(root)
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)

			if entry.IsDir() {
				if path == root {
					return nil
				}
				if excludes.matches(rel, true) {
					return fs.SkipDir
				}
				return nil
			}
			// Symlinks are not regular files: following them could leave the
			// root or loop forever.
			if !entry.Type().IsRegular() || seen[path] {
				return nil
			}
			if excludes.matches(rel, false) {
				return nil
			}
			seen[path] = true
			files = append(files, path)
			return nil
		})
	}

	sort.Strings(files)
	return files
}
