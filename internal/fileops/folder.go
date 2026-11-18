package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kaze/internal/ignore"
	"kaze/internal/pathutil"
)

// FolderEntry is one child of a folder in the sidebar tree. Directories are
// expanded on demand by calling ReadFolder again, so opening a large repository
// never walks the whole tree up front.
type FolderEntry struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
}

func ReadFolder(path, root string, excludes ignore.Matcher) ([]FolderEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", filepath.Base(path))
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// A non-nil slice keeps the generated binding encoding an empty array
	// instead of null, which the sidebar reads as a list.
	// A root that does not contain path (or none at all) falls back to path
	// itself, which still applies unanchored patterns by name.
	prefix := ""
	if root != "" && pathutil.IsWithin(root, path) {
		if rel, err := filepath.Rel(root, path); err == nil && rel != "." {
			prefix = filepath.ToSlash(rel) + "/"
		}
	}

	entries := make([]FolderEntry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		isDir := isDirectory(path, entry)
		if excludes.Matches(prefix+entry.Name(), isDir) {
			continue
		}
		entries = append(entries, FolderEntry{
			Path:  filepath.Join(path, entry.Name()),
			Name:  entry.Name(),
			IsDir: isDir,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		left, right := strings.ToLower(entries[i].Name), strings.ToLower(entries[j].Name)
		if left != right {
			return left < right
		}
		// Names that differ only in case keep a stable, predictable order.
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// isDirectory reports whether an entry is a directory to browse into. A symlink
// to a directory counts, which is what editors like VS Code do; a broken link
// does not, so the tree cannot show a chevron it can never expand.
func isDirectory(parent string, entry os.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&os.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(parent, entry.Name()))
	return err == nil && info.IsDir()
}
