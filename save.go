package main

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"kaze/internal/document"
	"kaze/internal/pathutil"
)

// SaveDocument writes a buffer's text to disk. The frontend owns the text, so
// it passes the whole buffer instead of a path Go could reread: an unsaved edit
// is exactly what should land in the file.
func (a *App) SaveDocument(request document.SaveRequest) (document.SaveResult, error) {
	result, err := document.Save(request)
	if err == nil && !result.Conflict {
		// The watcher sees the write this call just made; treat it as the new
		// baseline so the buffer is not reloaded over itself.
		a.watcher.rebaseline(result.Path)
	}
	return result, err
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
	return pathutil.CanonicalPath(path), nil
}
