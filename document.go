package main

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// maxDocumentSize caps what the editor will read. Monaco slows to a crawl far
// below this, and refusing loudly beats freezing the window.
const maxDocumentSize = 10 << 20

// Document is a file opened in the editor. File I/O lives in Go so the
// frontend never touches the filesystem.
type Document struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// LoadDocument reads a text file for the editor. Directories, special files,
// oversized files and non-UTF-8 content are rejected with a message the UI can
// show as-is, instead of rendering garbage.
func (a *App) LoadDocument(path string) (Document, error) {
	return loadDocument(path)
}

func loadDocument(path string) (Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Document{}, err
	}
	name := filepath.Base(path)
	switch {
	case info.IsDir():
		return Document{}, fmt.Errorf("%s is a folder", name)
	case !info.Mode().IsRegular():
		return Document{}, fmt.Errorf("%s is not a regular file", name)
	case info.Size() > maxDocumentSize:
		return Document{}, fmt.Errorf("%s is larger than %d MB", name, maxDocumentSize>>20)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	if !utf8.Valid(data) {
		return Document{}, fmt.Errorf("%s is not a text file", name)
	}

	return Document{Path: path, Name: name, Content: string(data)}, nil
}
