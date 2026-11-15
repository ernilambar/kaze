package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadDocumentReadsTextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("first\nsecond\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	doc, err := loadDocument(path)
	if err != nil {
		t.Fatalf("loadDocument() error = %v", err)
	}

	want := Document{Path: path, Name: "notes.txt", Kind: documentKindText, Content: "first\nsecond\n"}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	want.Modified = info.ModTime().UnixMilli()
	if doc != want {
		t.Fatalf("loadDocument() = %+v, want %+v", doc, want)
	}
}

func TestLoadDocumentRejectsDirectory(t *testing.T) {
	if _, err := loadDocument(t.TempDir()); err == nil {
		t.Fatal("loadDocument() error = nil, want a folder is not a document error")
	}
}

func TestLoadDocumentRejectsMissingFile(t *testing.T) {
	if _, err := loadDocument(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("loadDocument() error = nil, want a missing file error")
	}
}

func TestLoadDocumentReadsImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diagram.PNG")
	data := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x01}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	doc, err := loadDocument(path)
	if err != nil {
		t.Fatalf("loadDocument() error = %v", err)
	}
	if doc.Kind != documentKindImage {
		t.Fatalf("loadDocument() kind = %q, want %q", doc.Kind, documentKindImage)
	}
	if doc.Content != "" {
		t.Fatalf("loadDocument() content = %q, want empty for an image", doc.Content)
	}
	want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	if doc.Src != want {
		t.Fatalf("loadDocument() src = %q, want %q", doc.Src, want)
	}
}

func TestLoadDocumentMapsImageExtensions(t *testing.T) {
	mimeTypes := map[string]string{
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
	dir := t.TempDir()
	for ext, mimeType := range mimeTypes {
		path := filepath.Join(dir, "image"+ext)
		if err := os.WriteFile(path, []byte{0x00}, 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		doc, err := loadDocument(path)
		if err != nil {
			t.Fatalf("loadDocument(%s) error = %v", ext, err)
		}
		if doc.Kind != documentKindImage {
			t.Fatalf("loadDocument(%s) kind = %q, want %q", ext, doc.Kind, documentKindImage)
		}
		if !strings.HasPrefix(doc.Src, "data:"+mimeType+";base64,") {
			t.Fatalf("loadDocument(%s) src = %q, want %s data URL", ext, doc.Src, mimeType)
		}
	}
}

func TestLoadDocumentRejectsBinaryContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blob.bin")
	if err := os.WriteFile(path, []byte{0x89, 0x50, 0x4e, 0x47, 0xff, 0xfe}, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := loadDocument(path); err == nil {
		t.Fatal("loadDocument() error = nil, want a not a text file error")
	}
}

func TestLoadDocumentRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.txt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := file.Truncate(maxDocumentSize + 1); err != nil {
		file.Close()
		t.Skipf("cannot create a sparse file on this filesystem: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	_, err = loadDocument(path)
	if err == nil {
		t.Fatal("loadDocument() error = nil, want a too large error")
	}
	if !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("loadDocument() error = %v, want it to mention the size limit", err)
	}
}

func TestLoadDocumentRejectsOversizedImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := file.Truncate(maxImageSize + 1); err != nil {
		file.Close()
		t.Skipf("cannot create a sparse file on this filesystem: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	_, err = loadDocument(path)
	if err == nil {
		t.Fatal("loadDocument() error = nil, want a too large error")
	}
	if !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("loadDocument() error = %v, want it to mention the size limit", err)
	}
}

func TestLoadDocumentRejectsSpecialFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test relies on the POSIX null device")
	}

	path := filepath.Join(t.TempDir(), "null")
	if err := os.Symlink("/dev/null", path); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}

	if _, err := loadDocument(path); err == nil {
		t.Fatal("loadDocument() error = nil, want a not a regular file error")
	}
}
