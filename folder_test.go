package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readFolder(path string) ([]FolderEntry, error) {
	return readFolderExcluding(path, path, excludeMatcher{})
}

func TestReadFolderListsDirectoriesFirstThenNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"beta", "Alpha", "notes"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	for _, name := range []string{"zeta.txt", "apple.txt", ".hidden"} {
		writeTempFile(t, dir, name, "x")
	}

	entries, err := readFolder(dir)
	if err != nil {
		t.Fatalf("readFolder() error = %v", err)
	}

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	want := []string{"Alpha", "beta", "notes", ".hidden", "apple.txt", "zeta.txt"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("readFolder() names = %q, want %q", names, want)
	}
}

func TestReadFolderReportsKindAndAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("create src: %v", err)
	}
	writeTempFile(t, dir, "main.go", "package main\n")

	entries, err := readFolder(dir)
	if err != nil {
		t.Fatalf("readFolder() error = %v", err)
	}

	want := []FolderEntry{
		{Path: filepath.Join(dir, "src"), Name: "src", IsDir: true},
		{Path: filepath.Join(dir, "main.go"), Name: "main.go", IsDir: false},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("readFolder() = %+v, want %+v", entries, want)
	}
}

func TestReadFolderReturnsEmptySliceForEmptyFolder(t *testing.T) {
	entries, err := readFolder(t.TempDir())
	if err != nil {
		t.Fatalf("readFolder() error = %v", err)
	}
	if entries == nil {
		t.Fatal("readFolder() = nil, want a non-nil slice so the UI receives an array")
	}
	if len(entries) != 0 {
		t.Fatalf("readFolder() = %+v, want no entries", entries)
	}
}

func TestReadFolderRejectsFile(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "notes.txt", "hello")

	_, err := readFolder(path)
	if err == nil {
		t.Fatal("readFolder() error = nil, want a not a folder error")
	}
	if !strings.Contains(err.Error(), "not a folder") {
		t.Fatalf("readFolder() error = %v, want it to say the path is not a folder", err)
	}
}

func TestReadFolderRejectsMissingPath(t *testing.T) {
	if _, err := readFolder(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("readFolder() error = nil, want a missing folder error")
	}
}

func TestReadFolderTreatsSymlinkedDirectoryAsFolder(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(dir, "linked")); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "broken")); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}

	entries, err := readFolder(dir)
	if err != nil {
		t.Fatalf("readFolder() error = %v", err)
	}

	byName := make(map[string]FolderEntry, len(entries))
	for _, entry := range entries {
		byName[entry.Name] = entry
	}
	if !byName["linked"].IsDir {
		t.Error("a symlink to a folder must be browsable")
	}
	if byName["broken"].IsDir {
		t.Error("a broken symlink must not claim to be a folder")
	}
}
