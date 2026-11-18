package app

import (
	"errors"
	"os"

	"kaze/internal/document"
)

// LoadDocument reads a file for the editor. Directories, special files,
// oversized files and binary content are rejected with a message the UI can
// show as-is, instead of rendering garbage. Text that is merely not valid UTF-8
// still opens with replacement characters and is flagged, because refusing a
// small mis-encoded log would be less useful than showing it. Supported image
// formats are returned as a data URL for the image viewer.
func (a *App) LoadDocument(path string) (document.Document, error) {
	return document.Load(path)
}

// FileExists reports whether path still exists. A refresh that can no longer
// read an open file uses it to tell a deletion from any other read failure,
// so only a file that is really gone marks its tab.
func (a *App) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

// OpenFileDialog asks for one file to open. An empty string means the user
// cancelled, which is not an error. No filter is added, so images can be picked
// for the viewer and text files for the editor from the same dialog.
func (a *App) OpenFileDialog() (string, error) {
	if a.wails == nil {
		return "", errors.New("file dialog is not available before startup")
	}
	return a.wails.Dialog.OpenFile().
		SetTitle("Open File").
		PromptForSingleSelection()
}
