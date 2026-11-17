package main

import (
	"errors"
	"strings"
)

// PrintRequest is the buffer to print. An image prints from Path, which lets
// the print panel read the file at its full resolution; text prints from
// Content, so an unsaved edit appears on paper exactly as it does on screen.
type PrintRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	// Image picks the picture at Path over the text in Content. The frontend
	// knows the tab's kind, which is more reliable than guessing from the
	// extension after a Save As renamed the buffer.
	Image bool `json:"image"`
}

// PrintDocument opens the macOS print panel for a buffer. Printing is done by
// AppKit so the panel, page setup and printer list are the ones the user
// already knows.
func (a *App) PrintDocument(request PrintRequest) error {
	if a.wails == nil {
		return errors.New("printing is not available before startup")
	}
	if request.Image {
		if request.Path == "" {
			return errors.New("nothing to print")
		}
		return printImage(request.Path)
	}
	if strings.TrimSpace(request.Content) == "" {
		return errors.New("nothing to print")
	}
	return printText(request.Content, printFontFamily())
}

// printFontFamily is the editor font, so a printout matches what the buffer
// looks like on screen. A missing or broken settings file falls back to the
// default rather than failing the print.
func printFontFamily() string {
	path, err := settingsFilePath()
	if err != nil {
		return defaultFontFamily
	}
	settings, err := loadSettings(path)
	if err != nil {
		return defaultFontFamily
	}
	return settings.Editor.FontFamily
}
