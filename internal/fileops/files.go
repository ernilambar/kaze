package fileops

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kaze/internal/ignore"
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

// ListFiles walks each root and skips excluded trees, ignored trees,
// unreadable directories and symlinks. A stale root is ignored instead of
// failing the whole list, so one folder moved outside the app cannot empty the
// palette.
func ListFiles(roots []string, excludes ignore.Matcher) FileList {
	return listFilesCapped(roots, excludes, maxListedFiles)
}

// listFilesCapped is ListFiles with an explicit cap, so a test can
// reach the truncation branch without creating fifty thousand files.
func listFilesCapped(roots []string, excludes ignore.Matcher, limit int) FileList {
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
	excludes ignore.Matcher,
	rules []ignore.Rule,
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

	if local := ignore.ReadRules(dir); len(local) > 0 {
		merged := make([]ignore.Rule, 0, len(rules)+len(local))
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
			if ignore.VCSDirectories[strings.ToLower(name)] {
				continue
			}
			if excludes.Matches(childRel, true) || ignore.Ignored(rules, path, true) {
				continue
			}
			walkFileRoot(path, childRel, excludes, rules, files, seen, truncated, limit)
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}
		if excludes.Matches(childRel, false) || ignore.Ignored(rules, path, false) {
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
