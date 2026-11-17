package main

import (
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

// useTempSettingsDir points the settings file at a fresh directory, so the
// sidebar bindings write somewhere disposable.
func useTempSettingsDir(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

// savedSidebarRoots reads the roots through a fresh App, which is what the
// next launch sees.
func savedSidebarRoots(t *testing.T) []string {
	t.Helper()
	roots, err := NewApp().LoadSidebarFolders()
	if err != nil {
		t.Fatalf("LoadSidebarFolders() error = %v", err)
	}
	return roots
}

func TestAddSidebarFolderSavesFolder(t *testing.T) {
	useTempSettingsDir(t)
	folder := t.TempDir()
	app := NewApp()

	got, err := app.AddSidebarFolder(folder)
	if err != nil {
		t.Fatalf("AddSidebarFolder() error = %v", err)
	}

	want := []string{folder}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AddSidebarFolder() = %q, want %q", got, want)
	}

	// The list must survive a restart, which means it is on disk.
	roots := savedSidebarRoots(t)
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("saved folders = %q, want %q", roots, want)
	}
}

func TestAddSidebarFolderKeepsOrderAndIgnoresRepeats(t *testing.T) {
	useTempSettingsDir(t)
	first := t.TempDir()
	second := t.TempDir()
	app := NewApp()

	for _, folder := range []string{first, second, first} {
		if _, err := app.AddSidebarFolder(folder); err != nil {
			t.Fatalf("AddSidebarFolder(%q) error = %v", folder, err)
		}
	}

	got, err := app.AddSidebarFolder(second)
	if err != nil {
		t.Fatalf("AddSidebarFolder() error = %v", err)
	}

	want := []string{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("folders = %q, want %q", got, want)
	}
}

func TestAddSidebarFolderRejectsFile(t *testing.T) {
	useTempSettingsDir(t)
	path := writeTempFile(t, t.TempDir(), "notes.txt", "hello")

	if _, err := NewApp().AddSidebarFolder(path); err == nil {
		t.Fatal("AddSidebarFolder() error = nil, want a not a folder error")
	}
}

func TestAddSidebarFolderRejectsBlankPath(t *testing.T) {
	useTempSettingsDir(t)

	if _, err := NewApp().AddSidebarFolder("   "); err == nil {
		t.Fatal("AddSidebarFolder() error = nil, want an empty folder error")
	}
}

func TestSetSidebarAutoRevealSavesChoice(t *testing.T) {
	useTempSettingsDir(t)
	app := NewApp()

	if err := app.SetSidebarAutoReveal(false); err != nil {
		t.Fatalf("SetSidebarAutoReveal(false) error = %v", err)
	}

	path, err := settingsFilePath()
	if err != nil {
		t.Fatalf("settingsFilePath() error = %v", err)
	}
	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if settings.Sidebar.AutoReveal {
		t.Fatal("Sidebar.AutoReveal = true, want the saved false")
	}
	if !settings.Sidebar.Visible {
		t.Fatal("Sidebar.Visible = false, want the unrelated setting untouched")
	}
}

func TestRemoveSidebarFolderDropsOneRoot(t *testing.T) {
	useTempSettingsDir(t)
	first := t.TempDir()
	second := t.TempDir()
	app := NewApp()

	if _, err := app.AddSidebarFolder(first); err != nil {
		t.Fatalf("AddSidebarFolder() error = %v", err)
	}
	if _, err := app.AddSidebarFolder(second); err != nil {
		t.Fatalf("AddSidebarFolder() error = %v", err)
	}

	got, err := app.RemoveSidebarFolder(first)
	if err != nil {
		t.Fatalf("RemoveSidebarFolder() error = %v", err)
	}

	want := []string{second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveSidebarFolder() = %q, want %q", got, want)
	}
}

func TestRemoveSidebarFolderIgnoresUnknownRoot(t *testing.T) {
	useTempSettingsDir(t)
	folder := t.TempDir()
	app := NewApp()

	if _, err := app.AddSidebarFolder(folder); err != nil {
		t.Fatalf("AddSidebarFolder() error = %v", err)
	}

	got, err := app.RemoveSidebarFolder(filepath.Join(folder, "elsewhere"))
	if err != nil {
		t.Fatalf("RemoveSidebarFolder() error = %v", err)
	}

	want := []string{folder}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveSidebarFolder() = %q, want %q", got, want)
	}
}

func TestAddSidebarFolderKeepsConcurrentAdditions(t *testing.T) {
	useTempSettingsDir(t)
	app := NewApp()

	folders := make([]string, 4)
	for i := range folders {
		folders[i] = t.TempDir()
	}

	// The sidebar and a launch can add at the same time; a lost update would
	// drop a root the user asked for.
	var wg sync.WaitGroup
	errs := make([]error, len(folders))
	for i, folder := range folders {
		wg.Add(1)
		go func(i int, folder string) {
			defer wg.Done()
			_, errs[i] = app.AddSidebarFolder(folder)
		}(i, folder)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("AddSidebarFolder(%q) error = %v", folders[i], err)
		}
	}

	if roots := savedSidebarRoots(t); len(roots) != len(folders) {
		t.Fatalf("saved folders = %q, want all %d additions", roots, len(folders))
	}
}

func TestSetSidebarFoldersReplacesSavedRoots(t *testing.T) {
	useTempSettingsDir(t)
	first := t.TempDir()
	second := t.TempDir()
	app := NewApp()

	if _, err := app.AddSidebarFolder(first); err != nil {
		t.Fatalf("AddSidebarFolder() error = %v", err)
	}
	got, err := app.SetSidebarFolders([]string{second, second})
	if err != nil {
		t.Fatalf("SetSidebarFolders() error = %v", err)
	}

	want := []string{second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SetSidebarFolders() = %q, want %q", got, want)
	}
}

func TestSetSidebarFoldersKeepsOldListOnMissingFolder(t *testing.T) {
	useTempSettingsDir(t)
	folder := t.TempDir()
	app := NewApp()

	if _, err := app.AddSidebarFolder(folder); err != nil {
		t.Fatalf("AddSidebarFolder() error = %v", err)
	}
	if _, err := app.SetSidebarFolders([]string{filepath.Join(folder, "missing")}); err == nil {
		t.Fatal("SetSidebarFolders() error = nil, want a missing folder error")
	}

	want := []string{folder}
	if roots := savedSidebarRoots(t); !reflect.DeepEqual(roots, want) {
		t.Fatalf("saved folders = %q, want %q", roots, want)
	}
}
