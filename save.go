package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SaveRequest is one buffer on its way to disk. ExpectedModified is the mtime
// LoadDocument reported for the file the buffer was read from; a mismatch means
// another program wrote it in the meantime, and nothing is written. Overwrite
// skips that check, once the user has been told.
type SaveRequest struct {
	Path             string `json:"path"`
	Content          string `json:"content"`
	ExpectedModified int64  `json:"expected_modified"`
	Overwrite        bool   `json:"overwrite"`
	// BOM and EOL are the file's encoding as LoadDocument reported it. The
	// content itself is normalized by Monaco, so a save restores both around
	// the text it is handed.
	BOM bool   `json:"bom"`
	EOL string `json:"eol"`
	// InvalidUTF8 marks a buffer whose file was not valid UTF-8. Writing it
	// replaces the original bytes with U+FFFD, so the save returns without
	// writing until AllowInvalidUTF8 says the user agreed.
	InvalidUTF8      bool `json:"invalid_utf8"`
	AllowInvalidUTF8 bool `json:"allow_invalid_utf8"`
}

// SaveResult reports what a save did. A Conflict means the buffer was not
// written and the user has to decide whether to overwrite; InvalidUTF8 means
// the buffer holds replacement characters and the user has to agree to lose
// them. The other fields are then the file that caused it. Path and Name
// describe where the text belongs, so Save As can move a buffer to its new home
// in one step.
type SaveResult struct {
	Path string `json:"path"`
	// RealPath is Path with every symlink resolved, so the frontend can match
	// the saved file against the tabs it already has (an alias of an open file
	// is the same file).
	RealPath    string `json:"real_path"`
	Name        string `json:"name"`
	Modified    int64  `json:"modified"`
	Conflict    bool   `json:"conflict"`
	InvalidUTF8 bool   `json:"invalid_utf8"`
}

// SaveDocument writes a buffer's text to disk. The frontend owns the text, so
// it passes the whole buffer instead of a path Go could reread: an unsaved edit
// is exactly what should land in the file.
func (a *App) SaveDocument(request SaveRequest) (SaveResult, error) {
	result, err := saveDocument(request)
	if err == nil && !result.Conflict {
		// The watcher sees the write this call just made; treat it as the new
		// baseline so the buffer is not reloaded over itself.
		a.watcher.rebaseline(result.Path)
	}
	return result, err
}

func saveDocument(request SaveRequest) (SaveResult, error) {
	path, err := savePath(request.Path)
	if err != nil {
		return SaveResult{}, err
	}
	realPath := canonicalPath(path)

	if !request.Overwrite && request.ExpectedModified > 0 {
		conflict, err := changedOnDisk(path, request.ExpectedModified)
		if err != nil {
			return SaveResult{}, err
		}
		if conflict {
			return SaveResult{Path: path, RealPath: realPath, Name: filepath.Base(path), Conflict: true}, nil
		}
	}

	if request.InvalidUTF8 && !request.AllowInvalidUTF8 {
		return SaveResult{Path: path, RealPath: realPath, Name: filepath.Base(path), InvalidUTF8: true}, nil
	}

	text := normalizeEOL(request.Content, request.EOL)
	if request.BOM {
		text = byteOrderMark + text
	}
	if err := atomicWriteFile(path, []byte(text)); err != nil {
		return SaveResult{}, fmt.Errorf("save %s: %w", filepath.Base(path), err)
	}

	// The new mtime is what the next save compares against, so read it back
	// instead of guessing.
	info, err := os.Stat(path)
	if err != nil {
		return SaveResult{}, fmt.Errorf("read %s back: %w", filepath.Base(path), err)
	}
	return SaveResult{
		Path:     path,
		RealPath: canonicalPath(path),
		Name:     filepath.Base(path),
		Modified: info.ModTime().UnixMilli(),
	}, nil
}

// atomicWriteFile replaces path with data without ever exposing a half-written
// file: the bytes land in a temp file in the same directory, are synced, and
// renamed over the target in one step. The target keeps its permissions; a new
// file gets the usual 0644.
func atomicWriteFile(path string, data []byte) error {
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

// normalizeEOL rewrites every line ending to the file's chosen sequence. The
// frontend's model may already use it, so this is idempotent; a buffer moved
// from a window with another EOL setting is converted in one pass.
func normalizeEOL(text, eol string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if eol == documentEOLcrlf {
		return strings.ReplaceAll(text, "\n", "\r\n")
	}
	return text
}

// savePath validates where a save may write. A file that does not exist yet is
// fine: that is what Save As does. Only a path that is present and unsuitable
// is rejected, so saving never truncates a directory or a device.
func savePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("no file name to save to")
	}

	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return path, nil
	case err != nil:
		return "", err
	case info.IsDir():
		return "", fmt.Errorf("%s is a folder", filepath.Base(path))
	case !info.Mode().IsRegular():
		return "", fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	return path, nil
}

// changedOnDisk reports whether the file's mtime has moved since it was read. A
// file deleted in the meantime is not a conflict: saving recreates it, which is
// what the user asked for.
func changedOnDisk(path string, expected int64) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.ModTime().UnixMilli() != expected, nil
}

// SaveFileDialog asks where a buffer should be written. suggested is a full
// path when the buffer already has a file, and a plain name (an untitled
// buffer's) when it does not; an empty path means the user cancelled.
func (a *App) SaveFileDialog(suggested string) (string, error) {
	if a.wails == nil {
		return "", errors.New("file dialog is not available before startup")
	}

	dialog := a.wails.Dialog.SaveFile()
	dialog.SetOptions(&application.SaveFileDialogOptions{Title: "Save As"})
	suggested = strings.TrimSpace(suggested)
	if suggested != "" {
		if filepath.IsAbs(suggested) {
			dialog.SetDirectory(filepath.Dir(suggested))
			dialog.SetFilename(filepath.Base(suggested))
		} else {
			dialog.SetFilename(suggested)
		}
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil || path == "" {
		return path, err
	}
	// Resolve the chosen path so a save onto an alias of an open file is
	// recognised as that file, not as a second tab.
	return canonicalPath(path), nil
}
