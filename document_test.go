package main

import (
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

	want := Document{Path: path, Name: "notes.txt", Content: "first\nsecond\n"}
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

func TestLoadDocumentRejectsBinaryContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
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
