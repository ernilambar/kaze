package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxListedFiles caps Quick Open's file list. A large root would otherwise
// build a slice no palette renders; the palette reports when the walk stopped
// early so the user knows the list is incomplete.
const maxListedFiles = 50000

// FileList is what Quick Open reads: every regular file under one window's
// sidebar roots, sorted by path, and whether the walk hit the cap.
type FileList struct {
	Files     []string `json:"files"`
	Truncated bool     `json:"truncated"`
}

// ListFiles returns every regular file under one window's sidebar roots,
// sorted by path. The frontend filters this list locally, so each keystroke in
// Quick Open costs nothing. files.exclude decides what is walked, so a folder
// hidden in the tree is hidden here too; .gitignore files and VCS directories
// are skipped as well.
func (a *App) ListFiles(window string) (FileList, error) {
	path, err := settingsFilePath()
	if err != nil {
		return FileList{}, err
	}
	settings, err := loadSettings(path)
	if err != nil {
		return FileList{}, err
	}
	roots, err := a.LoadSidebarFolders(window)
	if err != nil {
		return FileList{}, err
	}
	return listFilesExcluding(roots, newExcludeMatcher(settings.Files.Exclude)), nil
}

// listFilesExcluding walks each root and skips excluded trees, ignored trees,
// unreadable directories and symlinks. A stale root is ignored instead of
// failing the whole list, so one folder moved outside the app cannot empty the
// palette.
func listFilesExcluding(roots []string, excludes excludeMatcher) FileList {
	return listFilesCapped(roots, excludes, maxListedFiles)
}

// listFilesCapped is listFilesExcluding with an explicit cap, so a test can
// reach the truncation branch without creating fifty thousand files.
func listFilesCapped(roots []string, excludes excludeMatcher, limit int) FileList {
	files := make([]string, 0)
	seen := make(map[string]bool)
	truncated := false

	for _, root := range roots {
		if truncated {
			break
		}
		root = filepath.Clean(root)
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			continue
		}
		walkFileRoot(root, "", excludes, nil, &files, seen, &truncated, limit)
	}

	sort.Strings(files)
	return FileList{Files: files, Truncated: truncated}
}

// walkFileRoot visits one directory and descends into the children that
// survive the filters. rules carries the .gitignore rules of the ancestors,
// because a pattern in a parent directory applies to everything below it.
func walkFileRoot(
	dir string,
	rel string,
	excludes excludeMatcher,
	rules []ignoreRule,
	files *[]string,
	seen map[string]bool,
	truncated *bool,
	limit int,
) {
	if *truncated {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	if local := readIgnoreRules(dir); len(local) > 0 {
		merged := make([]ignoreRule, 0, len(rules)+len(local))
		merged = append(merged, rules...)
		merged = append(merged, local...)
		rules = merged
	}

	for _, entry := range entries {
		if *truncated {
			return
		}
		name := entry.Name()
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		path := filepath.Join(dir, name)

		if entry.IsDir() {
			if vcsDirectories[strings.ToLower(name)] {
				continue
			}
			if excludes.matches(childRel, true) || ignored(rules, path, true) {
				continue
			}
			walkFileRoot(path, childRel, excludes, rules, files, seen, truncated, limit)
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}
		if excludes.matches(childRel, false) || ignored(rules, path, false) {
			continue
		}
		if seen[path] {
			continue
		}
		if len(*files) >= limit {
			*truncated = true
			return
		}
		seen[path] = true
		*files = append(*files, path)
	}
}
