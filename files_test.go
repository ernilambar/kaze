package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"kaze/internal/cli"
)

func TestAppListFilesReadsSidebarFolders(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeTempFile(t, root, "notes.txt", "hello\n")

	if err := writeSessionFile(sessionFile{Sidebar: sidebarRootsByWindow{
		defaultWindowName: {Roots: []string{root}},
	}}); err != nil {
		t.Fatal(err)
	}

	list, err := NewApp().ListFiles(defaultWindowName)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	files := list.Files
	want := []string{filepath.Join(root, "notes.txt")}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("ListFiles() = %q, want %q", files, want)
	}
	if list.Truncated {
		t.Fatal("ListFiles() truncated a tiny tree")
	}
}

func TestListFilesUsesTheCallingWindowsRoots(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	app.registerWindow(defaultWindowName, cli.EmptyLaunchRequest())
	app.registerWindow("window-2", cli.EmptyLaunchRequest())
	first := t.TempDir()
	second := t.TempDir()
	writeTempFile(t, first, "first.txt", "one\n")
	writeTempFile(t, second, "second.txt", "two\n")

	if _, err := app.SetSidebarFolders(defaultWindowName, []string{first}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetSidebarFolders("window-2", []string{second}); err != nil {
		t.Fatal(err)
	}

	list, err := app.ListFiles("window-2")
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if want := []string{filepath.Join(second, "second.txt")}; !reflect.DeepEqual(list.Files, want) {
		t.Fatalf("ListFiles(window-2) = %q, want %q", list.Files, want)
	}
}
