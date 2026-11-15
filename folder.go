package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FolderEntry is one child of a folder in the sidebar tree. Directories are
// expanded on demand by calling ReadFolder again, so opening a large repository
// never walks the whole tree up front.
type FolderEntry struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
}

// ReadFolder lists one level of a folder for the sidebar. The tree loads the
// rest lazily, so only the entries the user can see are read.
func (a *App) ReadFolder(path string) ([]FolderEntry, error) {
	return readFolder(path)
}

func readFolder(path string) ([]FolderEntry, error) {
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
	entries := make([]FolderEntry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		entries = append(entries, FolderEntry{
			Path:  filepath.Join(path, entry.Name()),
			Name:  entry.Name(),
			IsDir: isDirectory(path, entry),
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

// OpenFolderDialog asks for a folder to add to the sidebar. An empty string
// means the user cancelled, which is not an error.
func (a *App) OpenFolderDialog() (string, error) {
	if a.ctx == nil {
		return "", errors.New("folder dialog is not available before startup")
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Add Folder to Sidebar",
	})
}
