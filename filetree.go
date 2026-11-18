package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"kaze/internal/fileops"
	"kaze/internal/pathutil"
)

// CreateFile makes an empty file in parent. An existing name is an error rather
// than an overwrite: the tree is a file manager here, so New File must never
// truncate something the user already has.
func (a *App) CreateFile(parent, name string) (fileops.FileTreeResult, error) {
	path, err := fileops.EntryPath(parent, name)
	if err != nil {
		return fileops.FileTreeResult{}, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fileops.FileTreeResult{}, fmt.Errorf("%s already exists", name)
		}
		return fileops.FileTreeResult{}, fmt.Errorf("create %s: %w", name, err)
	}
	if err := file.Close(); err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("create %s: %w", name, err)
	}
	return fileops.TreeResult(path)
}

// CreateFolder makes one directory in parent. The path is validated here rather
// than trusted from the caller, because a name that walks up the tree would put
// the new folder somewhere the user cannot see.
func (a *App) CreateFolder(parent, name string) (fileops.FileTreeResult, error) {
	path, err := fileops.EntryPath(parent, name)
	if err != nil {
		return fileops.FileTreeResult{}, err
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fileops.FileTreeResult{}, fmt.Errorf("%s already exists", name)
		}
		return fileops.FileTreeResult{}, fmt.Errorf("create %s: %w", name, err)
	}
	return fileops.TreeResult(path)
}

// RenamePath gives path a new name and, when the text carries a folder, a new
// home too; missing parent folders are created first. A bare name stays a
// rename in place, so the common case is unchanged. The caller retargets the
// open tabs afterwards, so a renamed file stays open under its new path.
func (a *App) RenamePath(path, target string) (fileops.FileTreeResult, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return fileops.FileTreeResult{}, errors.New("no path to rename")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("rename %s: %w", filepath.Base(path), err)
	}
	next, err := fileops.ResolveInputPath(target, filepath.Dir(path))
	if err != nil {
		return fileops.FileTreeResult{}, err
	}
	if err := fileops.ValidateEntryName(filepath.Base(next)); err != nil {
		return fileops.FileTreeResult{}, err
	}
	// Renaming to the path it already has changes nothing; it is not an
	// "already exists" failure, because nothing is in the way.
	if next == path {
		return fileops.TreeResult(path)
	}
	// A directory cannot become a child of itself. Check before a missing parent
	// is created inside it, so a refusal leaves no empty folder behind.
	if info.IsDir() && pathutil.IsWithin(path, next) {
		return fileops.FileTreeResult{}, errors.New("a folder cannot be moved inside itself")
	}
	// A case-only rename finds the file itself at the target on a
	// case-insensitive volume. Anything else there is in the way, even when the
	// names differ only by case.
	if targetInfo, statErr := os.Lstat(next); statErr == nil {
		if !os.SameFile(info, targetInfo) {
			return fileops.FileTreeResult{}, fmt.Errorf("%s already exists", filepath.Base(next))
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fileops.FileTreeResult{}, fmt.Errorf("rename %s: %w", filepath.Base(path), statErr)
	}
	if err := os.MkdirAll(filepath.Dir(next), 0o755); err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("create %s: %w", filepath.Dir(next), err)
	}
	if err := os.Rename(path, next); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return fileops.FileTreeResult{}, fmt.Errorf("rename %s: %w", filepath.Base(path), err)
		}
		// A rename that crosses volumes is a copy and a delete, the same
		// fallback MovePath uses. A half-written copy is removed.
		if copyErr := fileops.CopyPath(path, next); copyErr != nil {
			_ = os.RemoveAll(next)
			return fileops.FileTreeResult{}, fmt.Errorf("rename %s: %w", filepath.Base(path), copyErr)
		}
		// The complete copy is the file now; tabs must follow it even when
		// part of the source could not be removed and stays visible.
		if removeErr := os.RemoveAll(path); removeErr != nil {
			log.Printf("rename %s: remove source after copy: %v", path, removeErr)
		}
	}
	return fileops.TreeResult(next)
}

// DuplicatePathSuggestion is the path Duplicate… offers before the user edits
// it: the first free " copy" name beside path, which is what Finder and VS Code
// call a duplicate. It reads no more than the folder listings, so the dialog
// opens with the name the backend would have picked on its own.
func (a *App) DuplicatePathSuggestion(path string) (string, error) {
	source := filepath.Clean(strings.TrimSpace(path))
	if source == "" || source == "." {
		return "", errors.New("no path to duplicate")
	}
	if _, err := os.Lstat(source); err != nil {
		return "", fmt.Errorf("duplicate %s: %w", filepath.Base(source), err)
	}
	return fileops.UniqueCopyPath(source)
}

// DuplicatePath copies path to target, which reads like a rename target: a
// relative path starts in the entry's own folder and missing folders are
// created. A name already there is a refusal rather than an overwrite, because
// the dialog shows the free name the copy will get and a taken one means the
// user typed over something else. A directory is copied whole, so the duplicate
// is a snapshot rather than a second name for the same contents.
func (a *App) DuplicatePath(path, target string) (fileops.FileTreeResult, error) {
	source := filepath.Clean(strings.TrimSpace(path))
	if source == "" || source == "." {
		return fileops.FileTreeResult{}, errors.New("no path to duplicate")
	}
	info, err := os.Lstat(source)
	if err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("duplicate %s: %w", filepath.Base(source), err)
	}
	next, err := fileops.ResolveInputPath(target, filepath.Dir(source))
	if err != nil {
		return fileops.FileTreeResult{}, err
	}
	if err := fileops.ValidateEntryName(filepath.Base(next)); err != nil {
		return fileops.FileTreeResult{}, err
	}
	// Copying a folder into itself would recurse through the copy it is
	// writing, so the guard is checked before any folder is created.
	if info.IsDir() && pathutil.IsWithin(source, next) {
		return fileops.FileTreeResult{}, errors.New("a folder cannot be copied inside itself")
	}
	if _, statErr := os.Lstat(next); statErr == nil {
		return fileops.FileTreeResult{}, fmt.Errorf("%s already exists", filepath.Base(next))
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fileops.FileTreeResult{}, fmt.Errorf("duplicate %s: %w", filepath.Base(source), statErr)
	}
	if err := os.MkdirAll(filepath.Dir(next), 0o755); err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("create %s: %w", filepath.Dir(next), err)
	}
	if err := fileops.CopyEntry(source, next); err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("duplicate %s: %w", filepath.Base(source), err)
	}
	return fileops.TreeResult(next)
}

// CopyPath copies path into targetDir, keeping its name. A name already taken
// there gets the next free " copy" name instead of failing, so pasting the same
// file twice is not an error.
func (a *App) CopyPath(path, targetDir string) (fileops.FileTreeResult, error) {
	source := filepath.Clean(strings.TrimSpace(path))
	if source == "" || source == "." {
		return fileops.FileTreeResult{}, errors.New("no path to copy")
	}
	info, err := os.Lstat(source)
	if err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("copy %s: %w", filepath.Base(source), err)
	}
	dir, err := fileops.DirectoryTarget(targetDir)
	if err != nil {
		return fileops.FileTreeResult{}, err
	}
	// The copy would land inside the tree being read and copy itself again,
	// without end.
	if info.IsDir() && pathutil.IsWithin(source, dir) {
		return fileops.FileTreeResult{}, errors.New("a folder cannot be copied inside itself")
	}
	target, err := fileops.UniqueCopyPath(filepath.Join(dir, filepath.Base(source)))
	if err != nil {
		return fileops.FileTreeResult{}, err
	}
	if err := fileops.CopyEntry(source, target); err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("copy %s: %w", filepath.Base(source), err)
	}
	return fileops.TreeResult(target)
}

// MovePath moves path into targetDir, keeping its name. The rename is the only
// copy-free path; when the two folders are on different volumes the contents are
// copied and the source removed, which is what the user asked for either way.
func (a *App) MovePath(path, targetDir string) (fileops.FileTreeResult, error) {
	source := filepath.Clean(strings.TrimSpace(path))
	if source == "" || source == "." {
		return fileops.FileTreeResult{}, errors.New("no path to move")
	}
	info, err := os.Lstat(source)
	if err != nil {
		return fileops.FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), err)
	}
	dir, err := fileops.DirectoryTarget(targetDir)
	if err != nil {
		return fileops.FileTreeResult{}, err
	}
	// Dropping a file on the folder it already lives in asks for nothing.
	if dir == filepath.Dir(source) {
		return fileops.TreeResult(source)
	}
	// Moving a folder inside itself would detach the subtree from the tree.
	if info.IsDir() && pathutil.IsWithin(source, dir) {
		return fileops.FileTreeResult{}, errors.New("a folder cannot be moved inside itself")
	}
	target := filepath.Join(dir, filepath.Base(source))
	if _, statErr := os.Lstat(target); statErr == nil {
		return fileops.FileTreeResult{}, fmt.Errorf("%s already exists", filepath.Base(source))
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fileops.FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), statErr)
	}
	if err := os.Rename(source, target); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return fileops.FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), err)
		}
		if copyErr := fileops.CopyPath(source, target); copyErr != nil {
			_ = os.RemoveAll(target)
			return fileops.FileTreeResult{}, fmt.Errorf("move %s: %w", filepath.Base(source), copyErr)
		}
		// The complete copy is the file now; tabs must follow it even when
		// part of the source could not be removed and stays visible.
		if removeErr := os.RemoveAll(source); removeErr != nil {
			log.Printf("move %s: remove source after copy: %v", source, removeErr)
		}
	}
	return fileops.TreeResult(target)
}

// TrashPath moves path to the macOS Trash. Deleting for good is not offered by
// the tree, so a mistyped rename or drop can still be undone in Finder.
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
