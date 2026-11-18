package document

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kaze/internal/pathutil"
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

func Save(request SaveRequest) (SaveResult, error) {
	path, err := savePath(request.Path)
	if err != nil {
		return SaveResult{}, err
	}
	realPath := pathutil.CanonicalPath(path)

	if !request.Overwrite && request.ExpectedModified > 0 {
		conflict, err := ChangedOnDisk(path, request.ExpectedModified)
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
	if err := pathutil.WriteFileAtomic(path, []byte(text)); err != nil {
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
		RealPath: pathutil.CanonicalPath(path),
		Name:     filepath.Base(path),
		Modified: info.ModTime().UnixMilli(),
	}, nil
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

// ChangedOnDisk reports whether the file's mtime has moved since it was read. A
// file deleted in the meantime is not a conflict: saving recreates it, which is
// what the user asked for.
func ChangedOnDisk(path string, expected int64) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.ModTime().UnixMilli() != expected, nil
}
