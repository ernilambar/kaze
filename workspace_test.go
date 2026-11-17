package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestRecentFilesMoveToFrontAndDeduplicate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()

	if _, err := app.AddRecentFile("/one.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.AddRecentFile("/two.txt"); err != nil {
		t.Fatal(err)
	}
	files, err := app.AddRecentFile("/one.txt")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"/one.txt", "/two.txt"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("recent files = %q, want %q", files, want)
	}
}

func TestRecentFilesAreCapped(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()

	var files []string
	for index := range maxRecentFiles + 5 {
		var err error
		files, err = app.AddRecentFile(fmt.Sprintf("/tmp/%d.txt", index))
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(files) != maxRecentFiles {
		t.Fatalf("recent files = %d, want %d", len(files), maxRecentFiles)
	}
}

func TestClearRecentFilesEmptiesTheList(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()

	if _, err := app.AddRecentFile("/one.txt"); err != nil {
		t.Fatal(err)
	}
	files, err := app.ClearRecentFiles()
	if err != nil {
		t.Fatal(err)
	}
	if files == nil || len(files) != 0 {
		t.Fatalf("recent files = %#v, want a non-nil empty list", files)
	}
}

func TestSetSessionPersistsNormalizedState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()

	err := app.SetSession("", WindowSession{
		OpenFiles: []string{"/a.txt", " ", "/a.txt", "/b.txt"},
		Active:    "  /b.txt ",
	})
	if err != nil {
		t.Fatal(err)
	}

	path, err := sessionFilePath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var state sessionFile
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	want := []string{"/a.txt", "/b.txt"}
	if !reflect.DeepEqual(state.Window.OpenFiles, want) {
		t.Fatalf("Window.OpenFiles = %q, want %q", state.Window.OpenFiles, want)
	}
	if state.Window.Active != "/b.txt" {
		t.Fatalf("Window.Active = %q, want /b.txt", state.Window.Active)
	}
}

func TestSetSidebarVisibleSurvivesFolderChanges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()

	if err := app.SetSidebarVisible(false); err != nil {
		t.Fatal(err)
	}
	if _, err := app.AddSidebarFolder(folder); err != nil {
		t.Fatal(err)
	}

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Sidebar.Visible {
		t.Fatal("Sidebar.Visible = true after it was hidden")
	}

	if _, err := app.RemoveAllSidebarFolders(); err != nil {
		t.Fatal(err)
	}
	settings, err = app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.Sidebar.Folders) != 0 {
		t.Fatalf("Sidebar.Folders = %q after RemoveAllSidebarFolders", settings.Sidebar.Folders)
	}
}

func TestWatcherReportsExternalChangesOncePerBaseline(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "notes.txt", "one")
	watcher := fileWatcher{}
	watcher.set("main", []string{path})

	if changed := watcher.poll()["main"]; len(changed) != 0 {
		t.Fatalf("first poll = %q, want no changes", changed)
	}

	if err := os.WriteFile(path, []byte("two, longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed := watcher.poll()["main"]
	if len(changed) != 1 || changed[0] != path {
		t.Fatalf("poll after write = %q, want the file", changed)
	}
	if changed := watcher.poll()["main"]; len(changed) != 0 {
		t.Fatalf("second poll = %q, want the change reported once", changed)
	}

	watcher.rebaseline(path)
	if err := os.WriteFile(path, []byte("three"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed := watcher.poll()["main"]; len(changed) != 1 {
		t.Fatalf("poll after a rebaselined write = %q, want the file", changed)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if changed := watcher.poll()["main"]; len(changed) != 1 {
		t.Fatalf("poll after delete = %q, want the file", changed)
	}
}
