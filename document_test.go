package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExistsReportsMissingFiles(t *testing.T) {
	app := &App{}
	path := filepath.Join(t.TempDir(), "notes.txt")
	if app.FileExists(path) {
		t.Fatalf("FileExists(%q) = true, want false for a missing file", path)
	}
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if !app.FileExists(path) {
		t.Fatalf("FileExists(%q) = false, want true for a file on disk", path)
	}
	// A file deleted after it was opened reads as gone, which is what marks
	// its tab instead of silently skipping it.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	if app.FileExists(path) {
		t.Fatalf("FileExists(%q) = true, want false after deletion", path)
	}
}
