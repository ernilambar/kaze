package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListFilesWalksRootsAndSkipsGeneratedTrees(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "main.go", "package main\n")
	for _, dir := range []string{
		filepath.Join("src", "app"),
		filepath.Join("node_modules", "pkg"),
		".git",
		".hg",
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}
	writeTempFile(t, filepath.Join(root, "src", "app"), "app.ts", "export {};\n")
	writeTempFile(t, filepath.Join(root, "node_modules", "pkg"), "index.js", "\n")
	writeTempFile(t, filepath.Join(root, ".git"), "config", "\n")
	writeTempFile(t, filepath.Join(root, ".hg"), "hgrc", "\n")

	got := listFiles([]string{root})
	want := []string{
		filepath.Join(root, "main.go"),
		filepath.Join(root, "src", "app", "app.ts"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %q, want %q", got, want)
	}
}

func TestListFilesDeduplicatesOverlappingRoots(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, src, "main.go", "package main\n")

	got := listFiles([]string{root, src})
	want := []string{filepath.Join(src, "main.go")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %q, want %q", got, want)
	}
}

func TestListFilesSkipsMissingAndSymlinkedRoots(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "notes.txt", "hello\n")
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got := listFiles([]string{filepath.Join(root, "missing"), root})
	want := []string{filepath.Join(root, "notes.txt")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %q, want %q", got, want)
	}
}

func TestListFilesReturnsEmptySliceWithoutRoots(t *testing.T) {
	got := listFiles(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("listFiles() = %#v, want a non-nil empty slice", got)
	}
}

func TestAppListFilesReadsSidebarFolders(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeTempFile(t, root, "notes.txt", "hello\n")

	path, err := settingsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	settings := defaultSettings()
	settings.Sidebar.Folders = []string{root}
	if err := writeSettings(path, settings); err != nil {
		t.Fatal(err)
	}

	files, err := NewApp().ListFiles()
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	want := []string{filepath.Join(root, "notes.txt")}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("ListFiles() = %q, want %q", files, want)
	}
}
