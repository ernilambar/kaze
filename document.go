package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// maxDocumentSize caps what the editor will read. Monaco slows to a crawl far
// below this, and refusing loudly beats freezing the window.
const maxDocumentSize = 10 << 20

// maxImageSize caps what the image viewer will read. The bytes are base64
// encoded on the way to the WebView, which adds about a third more data, and
// the decoder holds the whole image in memory, so this is deliberately lower
// than the files a photo editor would accept.
const maxImageSize = 32 << 20

// Document kinds the frontend uses to choose how a tab is rendered.
const (
	documentKindText  = "text"
	documentKindImage = "image"
)

// Document is a file opened in the editor. File I/O lives in Go so the
// frontend never touches the filesystem. Content holds text for text documents;
// Src holds a data URL for images.
type Document struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
	Src     string `json:"src"`
	// Modified is the file's mtime in Unix milliseconds when the content was
	// read. The frontend hands it back on save, so a write never silently
	// overwrites a change another program made. Milliseconds, not nanoseconds:
	// the value travels through JavaScript, whose numbers lose precision above
	// 2^53.
	Modified int64 `json:"modified"`
}

// imageMIMETypes are the formats the viewer can display, keyed by lower-case
// extension. The MIME type goes into the data URL the WebView decodes.
var imageMIMETypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".tif":  "image/tiff",
	".tiff": "image/tiff",
	".ico":  "image/x-icon",
}

func imageMIMEType(path string) (string, bool) {
	mimeType, ok := imageMIMETypes[strings.ToLower(filepath.Ext(path))]
	return mimeType, ok
}

// LoadDocument reads a file for the editor. Directories, special files,
// oversized files and non-UTF-8 content are rejected with a message the UI can
// show as-is, instead of rendering garbage. Supported image formats are
// returned as a data URL for the image viewer.
func (a *App) LoadDocument(path string) (Document, error) {
	return loadDocument(path)
}

func loadDocument(path string) (Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Document{}, err
	}
	name := filepath.Base(path)
	modified := info.ModTime().UnixMilli()
	switch {
	case info.IsDir():
		return Document{}, fmt.Errorf("%s is a folder", name)
	case !info.Mode().IsRegular():
		return Document{}, fmt.Errorf("%s is not a regular file", name)
	}

	if mimeType, ok := imageMIMEType(path); ok {
		if info.Size() > maxImageSize {
			return Document{}, fmt.Errorf("%s is larger than %d MB", name, maxImageSize>>20)
		}
		return loadImage(path, name, mimeType, modified)
	}

	if info.Size() > maxDocumentSize {
		return Document{}, fmt.Errorf("%s is larger than %d MB", name, maxDocumentSize>>20)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	if !utf8.Valid(data) {
		return Document{}, fmt.Errorf("%s is not a text file", name)
	}

	return Document{
		Path:     path,
		Name:     name,
		Kind:     documentKindText,
		Content:  string(data),
		Modified: modified,
	}, nil
}

// loadImage reads an image and returns it as a data URL. Sending the bytes
// inline keeps the viewer out of the filesystem, matching how text documents
// are loaded, and works for formats the asset server cannot map to a path.
func loadImage(path, name, mimeType string, modified int64) (Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return Document{
		Path:     path,
		Name:     name,
		Kind:     documentKindImage,
		Src:      "data:" + mimeType + ";base64," + encoded,
		Modified: modified,
	}, nil
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
