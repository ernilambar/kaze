package pathutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// CanonicalPath resolves symlinks so the same file reached through a link and
// through its real path compares equal. It is the identity tabs match on, and
// what git paths are compared against the sidebar roots with. A deleted file
// resolves through its parent, because the file itself no longer exists to
// follow.
func CanonicalPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	dir, name := filepath.Split(path)
	if resolved, err := filepath.EvalSymlinks(filepath.Clean(dir)); err == nil {
		return filepath.Join(resolved, name)
	}
	return path
}

// IsWithin reports whether path sits inside folder. It is the Go side of the
// guard that keeps a folder from being dropped into its own subtree.
func IsWithin(folder, path string) bool {
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

// WithinAny reports whether path sits inside one of the sidebar roots. Nested
// roots and a repo wider than a root both land on the same answer.
func WithinAny(path string, folders []string) bool {
	for _, folder := range folders {
		if folder == "" {
			continue
		}
		relative, err := filepath.Rel(CanonicalPath(folder), path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		return true
	}
	return false
}

// WriteFileAtomic replaces path with data without ever exposing a half-written
// file: the bytes land in a temp file in the same directory, are synced, and
// renamed over the target in one step. The target keeps its permissions; a new
// file gets the usual 0644.
func WriteFileAtomic(path string, data []byte) error {
	// A direct symlink is the file's alias, not its identity. Resolve it first
	// so the rename replaces the target and the link keeps pointing at it.
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		path = target
	}

	mode := os.FileMode(0o644)
	exists := false
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode()
		exists = true
		// A rename gives the path a new inode, and the other names of a hard
		// link would keep the old text. Write those in place instead.
		if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Nlink > 1 {
			return writeFileInPlace(path, data)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		// A writable file in a folder that is not writable cannot take a temp
		// sibling; it can still be saved in place.
		if exists && errors.Is(err, os.ErrPermission) {
			return writeFileInPlace(path, data)
		}
		return err
	}
	tempPath := temp.Name()
	renamed := false
	defer func() {
		if !renamed {
			temp.Close()
			os.Remove(tempPath)
		}
	}()

	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Chmod(mode); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	renamed = true

	// The rename itself is a directory entry; sync it so a crash cannot leave
	// the new name unwritten. Best effort: the file's contents are already safe.
	if dirFile, err := os.Open(dir); err == nil {
		dirFile.Sync()
		dirFile.Close()
	}
	return nil
}

// writeFileInPlace overwrites an existing file through its own inode, keeping
// hard links, extended attributes and ownership. It is not atomic: a crash
// mid-write leaves a truncated file, so it is only the fallback.
func writeFileInPlace(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// UniquePaths keeps the first occurrence of each path, preserving order.
func UniquePaths(paths []string) []string {
	unique := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		unique = append(unique, path)
	}
	return unique
}
