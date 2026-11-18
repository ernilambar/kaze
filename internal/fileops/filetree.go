package fileops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"kaze/internal/pathutil"
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

// TreeResult describes path once a command has created or moved it.
func TreeResult(path string) (FileTreeResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileTreeResult{}, err
	}
	return FileTreeResult{
		Path:     path,
		RealPath: pathutil.CanonicalPath(path),
		Name:     filepath.Base(path),
		IsDir:    info.IsDir(),
	}, nil
}

// ResolveInputPath turns what the user typed into an absolute path. A leading
// ~ means the home folder; anything relative is read from base.
func ResolveInputPath(input, base string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("a destination is required")
	}
	if input == "~" || strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home folder: %w", err)
		}
		input = filepath.Join(home, strings.TrimPrefix(input, "~"))
	}
	if !filepath.IsAbs(input) {
		input = filepath.Join(base, input)
	}
	return filepath.Clean(input), nil
}

// EntryPath validates name as one path segment inside parent. Names that walk
// up the tree, or that carry a separator, are rejected: the tree names the
// folder, so a name is never a path.
func EntryPath(parent, name string) (string, error) {
	if err := ValidateEntryName(name); err != nil {
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

// ValidateEntryName rejects the names the file system would either refuse or
// read as something other than a name.
func ValidateEntryName(name string) error {
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

// DirectoryTarget checks that target is a folder to copy or move into.
func DirectoryTarget(target string) (string, error) {
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

// UniqueCopyPath returns the first free name for a copy of path: photo.png
// keeps its name when it is free, then becomes photo copy.png, then
// photo copy 1.png.
func UniqueCopyPath(path string) (string, error) {
	dir, name := filepath.Split(path)
	ext := filepath.Ext(name)
	// A dotfile such as .env is all name: its copy is ".env copy".
	if ext == name {
		ext = ""
	}
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

// CopyEntry copies source to target. A half-written copy is worse than no copy,
// so a failure removes whatever reached the target.
func CopyEntry(source, target string) error {
	if err := CopyPath(source, target); err != nil {
		_ = os.RemoveAll(target)
		return err
	}
	return nil
}

// CopyPath copies a file or a whole directory tree. A symlink is recreated as a
// link rather than followed, so a copy of a project does not silently pull in
// whatever its links point at.
func CopyPath(source, target string) error {
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
	case info.Mode().IsRegular():
		return copyFile(source, target, info)
	default:
		// Opening a named pipe blocks until a writer appears, and a socket or
		// device has no contents to copy.
		return fmt.Errorf("%s is not a regular file", filepath.Base(source))
	}
}

func copyTree(source, target string, info os.FileInfo) error {
	// The folder stays writable while its children are copied in; a read-only
	// source gets its own mode back afterwards.
	if err := os.MkdirAll(target, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := filepath.Join(source, entry.Name())
		if err := CopyPath(child, filepath.Join(target, entry.Name())); err != nil {
			return err
		}
	}
	return os.Chmod(target, info.Mode().Perm())
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
	// A move deletes the source next; the copy must be on disk first.
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
