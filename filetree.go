package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// FileTreeResult is one path a tree command produced: the spelling that was
// written and the same path with symlinks resolved. Open tabs match on the
// resolved one, so a rename through a symlinked root retargets the tab that was
// opened directly and the other way round.
type FileTreeResult struct {
	Path     string `json:"path"`
	RealPath string `json:"real_path"`
	Name     string `json:"name"`
	IsDir    bool   `json:"is_dir"`
}

// treeResult describes path once a command has created or moved it.
func treeResult(path string) (FileTreeResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileTreeResult{}, err
	}
	return FileTreeResult{
		Path:     path,
		RealPath: canonicalPath(path),
		Name:     filepath.Base(path),
		IsDir:    info.IsDir(),
	}, nil
}

// CreateFile makes an empty file in parent. An existing name is an error rather
// than an overwrite: the tree is a file manager here, so New File must never
// truncate something the user already has.
func (a *App) CreateFile(parent, name string) (FileTreeResult, error) {
	path, err := entryPath(parent, name)
	if err != nil {
		return FileTreeResult{}, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return FileTreeResult{}, fmt.Errorf("%s already exists", name)
		}
		return FileTreeResult{}, fmt.Errorf("create %s: %w", name, err)
	}
	if err := file.Close(); err != nil {
		return FileTreeResult{}, fmt.Errorf("create %s: %w", name, err)
	}
	return treeResult(path)
}

// CreateFolder makes one directory in parent. The path is validated here rather
// than trusted from the caller, because a name that walks up the tree would put
// the new folder somewhere the user cannot see.
func (a *App) CreateFolder(parent, name string) (FileTreeResult, error) {
	path, err := entryPath(parent, name)
	if err != nil {
		return FileTreeResult{}, err
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return FileTreeResult{}, fmt.Errorf("%s already exists", name)
		}
		return FileTreeResult{}, fmt.Errorf("create %s: %w", name, err)
	}
	return treeResult(path)
}

// RenamePath gives path a new name inside the folder it already lives in. The
// caller retargets the open tabs afterwards, so a renamed file stays open under
// its new path.
func (a *App) RenamePath(path, name string) (FileTreeResult, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return FileTreeResult{}, errors.New("no path to rename")
	}
	if _, err := os.Lstat(path); err != nil {
		return FileTreeResult{}, fmt.Errorf("rename %s: %w", filepath.Base(path), err)
	}
	target, err := entryPath(filepath.Dir(path), name)
	if err != nil {
		return FileTreeResult{}, err
	}
	// Renaming a file to the name it already has changes nothing; it is not an
	// "already exists" failure, because nothing is in the way.
	if target == path {
		return treeResult(path)
	}
	// macOS volumes are case-insensitive by default, so a case-only rename finds
	// the file itself at the target. That one is allowed and renamed in place.
	if _, statErr := os.Lstat(target); statErr == nil {
		if !strings.EqualFold(filepath.Base(path), name) {
			return FileTreeResult{}, fmt.Errorf("%s already exists", name)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return FileTreeResult{}, fmt.Errorf("rename %s: %w", filepath.Base(path), statErr)
	}
	// A directory cannot become a child of itself.
	if err := os.Rename(path, target); err != nil {
		return FileTreeResult{}, fmt.Errorf("rename %s: %w", filepath.Base(path), err)
	}
	return treeResult(target)
}

// DuplicatePath copies path next to itself under a " copy" name, which is what
// Finder and VS Code do. A directory is copied whole, so the duplicate is a
// snapshot rather than a second name for the same contents.
func (a *App) DuplicatePath(path string) (FileTreeResult, error) {
	return a.CopyPath(path, filepath.Dir(filepath.Clean(path)))
}

// CopyPath copies path into targetDir, keeping its name. A name already taken
// there gets the next free " copy" name instead of failing, so pasting the same
// file twice is not an error.
func (a *App) CopyPath(path, targetDir string) (FileTreeResult, error) {
	source := filepath.Clean(strings.TrimSpace(path))
	if source == "" || source == "." {
		return FileTreeResult{}, errors.New("no path to copy")
	}
	if _, err := os.Lstat(source); err != nil {
		return FileTreeResult{}, fmt.Errorf("copy %s: %w", filepath.Base(source), err)
	}
	dir, err := directoryTarget(targetDir)
	if err != nil {
		return FileTreeResult{}, err
	}
	target, err := uniqueCopyPath(filepath.Join(dir, filepath.Base(source)))
	if err != nil {
		return FileTreeResult{}, err
	}
	if err := copyPath(source, target); err != nil {
		// A half-written copy is worse than no copy, so a failure leaves
		// nothing behind.
		_ = os.RemoveAll(target)
		return FileTreeResult{}, fmt.Errorf("copy %s: %w", filepath.Base(source), err)
	}
	return treeResult(target)
}

// MovePath moves path into targetDir, keeping its name. The rename is the only
// copy-free path; when the two folders are on different volumes the contents are
// copied and the source removed, which is what the user asked for either way.
func (a *App) MovePath(path, targetDir string) (FileTreeResult, error) {
	source := filepath.Clean(strings.TrimSpace(path))
	if source == "" || source == "." {
		return FileTreeResult{}, errors.New("no path to move")
	}
	info, err := os.Lstat(source)
	if err != nil {
		return FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), err)
	}
	dir, err := directoryTarget(targetDir)
	if err != nil {
		return FileTreeResult{}, err
	}
	// Dropping a file on the folder it already lives in asks for nothing.
	if dir == filepath.Dir(source) {
		return treeResult(source)
	}
	// Moving a folder inside itself would detach the subtree from the tree.
	if info.IsDir() && isWithinPath(source, dir) {
		return FileTreeResult{}, errors.New("a folder cannot be moved inside itself")
	}
	target := filepath.Join(dir, filepath.Base(source))
	if _, statErr := os.Lstat(target); statErr == nil {
		return FileTreeResult{}, fmt.Errorf("%s already exists", filepath.Base(source))
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), statErr)
	}
	if err := os.Rename(source, target); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), err)
		}
		if copyErr := copyPath(source, target); copyErr != nil {
			_ = os.RemoveAll(target)
			return FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), copyErr)
		}
		if removeErr := os.RemoveAll(source); removeErr != nil {
			return FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), removeErr)
		}
	}
	return treeResult(target)
}

// MovePathToFolder moves path into a typed destination folder, creating the
// folder and any missing parents first. Move… is a text prompt rather than a
// picker, so the destination is not guaranteed to exist the way a drop target
// is. A relative folder is read from the entry's own folder, which makes "docs"
// mean the sibling next to the file.
func (a *App) MovePathToFolder(path, folder string) (FileTreeResult, error) {
	source := filepath.Clean(strings.TrimSpace(path))
	if source == "" || source == "." {
		return FileTreeResult{}, errors.New("no path to move")
	}
	info, err := os.Lstat(source)
	if err != nil {
		return FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), err)
	}
	dir, err := resolveFolderPath(folder, filepath.Dir(source))
	if err != nil {
		return FileTreeResult{}, err
	}
	// Check the subtree before creating anything: a folder moved inside itself
	// must not leave a new empty directory behind on the way to the refusal.
	if info.IsDir() && isWithinPath(source, dir) {
		return FileTreeResult{}, errors.New("a folder cannot be moved inside itself")
	}
	target, err := os.Stat(dir)
	switch {
	case err == nil && !target.IsDir():
		return FileTreeResult{}, fmt.Errorf("%s is not a folder", filepath.Base(dir))
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return FileTreeResult{}, fmt.Errorf("create %s: %w", dir, err)
		}
	default:
		return FileTreeResult{}, fmt.Errorf("%s: %w", filepath.Base(dir), err)
	}
	return a.MovePath(source, dir)
}

// resolveFolderPath turns what the user typed into an absolute destination.
// A leading ~ means the home folder; anything relative is read from base.
func resolveFolderPath(folder, base string) (string, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return "", errors.New("a destination folder is required")
	}
	if folder == "~" || strings.HasPrefix(folder, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home folder: %w", err)
		}
		folder = filepath.Join(home, strings.TrimPrefix(folder, "~"))
	}
	if !filepath.IsAbs(folder) {
		folder = filepath.Join(base, folder)
	}
	return filepath.Clean(folder), nil
}

// TrashPath moves path to the platform's Trash. Deleting for good is not
// offered by the tree, so a mistyped rename or drop can still be undone in
// Finder.
func (a *App) TrashPath(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return errors.New("no path to move to the Trash")
	}
	if _, err := os.Lstat(path); err != nil {
		return fmt.Errorf("move %s to the Trash: %w", filepath.Base(path), err)
	}
	return trashPath(path)
}

// entryPath validates name as one path segment inside parent. Names that walk
// up the tree, or that carry a separator, are rejected: the tree names the
// folder, so a name is never a path.
func entryPath(parent, name string) (string, error) {
	if err := validateEntryName(name); err != nil {
		return "", err
	}
	info, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("%s: %w", filepath.Base(parent), err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", filepath.Base(parent))
	}
	return filepath.Join(parent, name), nil
}

// validateEntryName rejects the names the file system would either refuse or
// read as something other than a name.
func validateEntryName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("a name is required")
	case name == "." || name == "..":
		return fmt.Errorf("%q is not a usable name", name)
	case strings.ContainsAny(name, "/\\"):
		return errors.New("a name cannot contain a path separator")
	case strings.ContainsRune(name, 0):
		return errors.New("a name cannot contain a NUL byte")
	}
	return nil
}

// directoryTarget checks that target is a folder to copy or move into.
func directoryTarget(target string) (string, error) {
	target = filepath.Clean(strings.TrimSpace(target))
	if target == "" || target == "." {
		return "", errors.New("no destination folder")
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("%s: %w", filepath.Base(target), err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", filepath.Base(target))
	}
	return target, nil
}

// isWithinPath reports whether path sits inside folder. It is the Go side of the
// guard that keeps a folder from being dropped into its own subtree.
func isWithinPath(folder, path string) bool {
	folder = filepath.Clean(folder)
	path = filepath.Clean(path)
	if folder == path {
		return true
	}
	rel, err := filepath.Rel(folder, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// uniqueCopyPath returns the first free name for a copy of path: photo.png
// keeps its name when it is free, then becomes photo copy.png, then
// photo copy 1.png.
func uniqueCopyPath(path string) (string, error) {
	dir, name := filepath.Split(path)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for index := 0; index < 1000; index++ {
		suffix := ""
		if index == 1 {
			suffix = " copy"
		} else if index > 1 {
			suffix = fmt.Sprintf(" copy %d", index-1)
		}
		candidate := filepath.Join(dir, stem+suffix+ext)
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("copy %s: %w", name, err)
		}
	}
	return "", fmt.Errorf("copy %s: too many copies already", name)
}

// copyPath copies a file or a whole directory tree. A symlink is recreated as a
// link rather than followed, so a copy of a project does not silently pull in
// whatever its links point at.
func copyPath(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		link, err := os.Readlink(source)
		if err != nil {
			return err
		}
		return os.Symlink(link, target)
	case info.IsDir():
		return copyTree(source, target, info)
	default:
		return copyFile(source, target, info)
	}
}

func copyTree(source, target string, info os.FileInfo) error {
	if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := filepath.Join(source, entry.Name())
		if err := copyPath(child, filepath.Join(target, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, target string, info os.FileInfo) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
