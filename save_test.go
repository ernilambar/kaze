package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveDocumentWritesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")

	result, err := saveDocument(SaveRequest{Path: path, Content: "hello\n"})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}
	if result.Conflict {
		t.Fatal("saveDocument() conflict = true, want a plain write")
	}
	if result.Path != path || result.Name != "notes.txt" {
		t.Fatalf("saveDocument() result = %+v, want it to name %s", result, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("saved content = %q, want %q", data, "hello\n")
	}
}

// A file opened with a BOM and CRLF endings must come back byte-for-byte when
// its buffer is saved unchanged: the editor strips the BOM to show the text and
// normalizes the model to the file's EOL, and the save restores both.
func TestSaveDocumentRestoresBOMAndCRLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "windows.txt")
	original := []byte("\uFEFFfirst\r\nsecond\r\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	doc, err := loadDocument(path)
	if err != nil {
		t.Fatalf("loadDocument() error = %v", err)
	}
	// Monaco hands the text back with its model EOL, which is the file's.
	result, err := saveDocument(SaveRequest{
		Path:    path,
		Content: doc.Content,
		BOM:     doc.BOM,
		EOL:     doc.EOL,
	})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}
	if result.Conflict || result.InvalidUTF8 {
		t.Fatalf("saveDocument() result = %+v, want a plain write", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("saved content = %q, want %q", data, original)
	}
}

// A buffer whose file was not valid UTF-8 must not quietly replace the original
// bytes. The first save says so without writing; the retry, after the user
// agreed, writes the replacement characters.
func TestSaveDocumentWarnsBeforeWritingInvalidUTF8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latin1.txt")
	original := []byte{'c', 'a', 'f', 0xe9, '\n'}
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := saveDocument(SaveRequest{
		Path:        path,
		Content:     "caf\uFFFD\n",
		InvalidUTF8: true,
	})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}
	if !result.InvalidUTF8 {
		t.Fatal("saveDocument() invalid_utf8 = false, want the write to wait for agreement")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("file content = %q, want the original bytes untouched", data)
	}

	result, err = saveDocument(SaveRequest{
		Path:             path,
		Content:          "caf\uFFFD\n",
		InvalidUTF8:      true,
		AllowInvalidUTF8: true,
	})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}
	if result.InvalidUTF8 {
		t.Fatal("saveDocument() invalid_utf8 = true, want the agreed write to go through")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "caf\uFFFD\n" {
		t.Fatalf("file content = %q, want the replacement characters", data)
	}
}

// A buffer saved and then reopened must agree with the mtime the save reported,
// or the very next save would claim the file changed on disk.
func TestSaveDocumentModifiedMatchesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")

	saved, err := saveDocument(SaveRequest{Path: path, Content: "text\n"})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}

	doc, err := loadDocument(path)
	if err != nil {
		t.Fatalf("loadDocument() error = %v", err)
	}
	if doc.Modified != saved.Modified {
		t.Fatalf("loadDocument() modified = %d, want %d", doc.Modified, saved.Modified)
	}
}

func TestSaveDocumentRejectsChangedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	doc, err := loadDocument(path)
	if err != nil {
		t.Fatalf("loadDocument() error = %v", err)
	}

	// Another program writes the file after the buffer was read.
	if err := os.WriteFile(path, []byte("theirs"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatalf("set file time: %v", err)
	}

	result, err := saveDocument(SaveRequest{
		Path:             path,
		Content:          "mine",
		ExpectedModified: doc.Modified,
	})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}
	if !result.Conflict {
		t.Fatal("saveDocument() conflict = false, want the changed file to block the write")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "theirs" {
		t.Fatalf("file content = %q, want the other program's text untouched", data)
	}
}

func TestSaveDocumentOverwritesChangedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	doc, err := loadDocument(path)
	if err != nil {
		t.Fatalf("loadDocument() error = %v", err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatalf("set file time: %v", err)
	}

	result, err := saveDocument(SaveRequest{
		Path:             path,
		Content:          "mine",
		ExpectedModified: doc.Modified,
		Overwrite:        true,
	})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}
	if result.Conflict {
		t.Fatal("saveDocument() conflict = true, want the overwrite to go through")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "mine" {
		t.Fatalf("file content = %q, want the buffer's text", data)
	}
}

// A file deleted while its buffer is open is recreated: that is what saving
// means, and reporting a conflict would only make the user reload an empty file.
func TestSaveDocumentRecreatesDeletedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	doc, err := loadDocument(path)
	if err != nil {
		t.Fatalf("loadDocument() error = %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	result, err := saveDocument(SaveRequest{
		Path:             path,
		Content:          "kept",
		ExpectedModified: doc.Modified,
	})
	if err != nil {
		t.Fatalf("saveDocument() error = %v", err)
	}
	if result.Conflict {
		t.Fatal("saveDocument() conflict = true, want a deleted file to be recreated")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file is missing: %v", err)
	}
}

func TestSaveDocumentRejectsFolder(t *testing.T) {
	if _, err := saveDocument(SaveRequest{Path: t.TempDir(), Content: "x"}); err == nil {
		t.Fatal("saveDocument() error = nil, want a folder to be rejected")
	}
}

func TestSaveDocumentRejectsEmptyPath(t *testing.T) {
	if _, err := saveDocument(SaveRequest{Path: "  ", Content: "x"}); err == nil {
		t.Fatal("saveDocument() error = nil, want an empty path to be rejected")
	}
}
