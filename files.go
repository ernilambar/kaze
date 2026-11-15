package main

import (
	"io/fs"
	"path/filepath"
	"sort"
)

// quickOpenSkipDirs are directory names that Quick Open never walks: version
// control metadata and installed packages hold thousands of files nobody opens
// by hand, and including them would make the palette slow and noisy.
var quickOpenSkipDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
}

// ListFiles returns every regular file under the sidebar roots, sorted by
// path. The frontend filters this list locally, so each keystroke in Quick Open
// costs nothing.
func (a *App) ListFiles() ([]string, error) {
	path, err := settingsFilePath()
	if err != nil {
		return nil, err
	}
	settings, err := loadSettings(path)
	if err != nil {
		return nil, err
	}
	return listFiles(settings.Sidebar.Folders), nil
}

// listFiles walks each root and skips generated trees, unreadable directories
// and symlinks. A stale root is ignored instead of failing the whole list, so
// one folder moved outside Kova cannot empty the palette.
func listFiles(roots []string) []string {
	files := make([]string, 0)
	seen := make(map[string]bool)

	for _, root := range roots {
		root = filepath.Clean(root)
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if path == root && !entry.IsDir() {
				return nil
			}
			if entry.IsDir() {
				if quickOpenSkipDirs[entry.Name()] {
					return fs.SkipDir
				}
				return nil
			}
			// Symlinks are not regular files: following them could leave the
			// root or loop forever.
			if !entry.Type().IsRegular() || seen[path] {
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
