package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWorkspaceSessionParksTabsUnderCanonicalPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	workspace := writeWorkspaceTestFileAt(t, folder, "work"+workspaceExtension)

	err := app.SetSession("", WindowSession{
		Workspace: workspace,
		Folders:   []string{folder},
		OpenFiles: []string{filepath.Join(folder, "a.txt")},
		Active:    filepath.Join(folder, "a.txt"),
		Expanded:  []string{folder},
	})
	if err != nil {
		t.Fatal(err)
	}

	info, err := app.LoadWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(folder, "a.txt")}
	if !reflect.DeepEqual(info.Session.OpenFiles, want) {
		t.Fatalf("parked tabs = %q, want %q", info.Session.OpenFiles, want)
	}
}

func TestWorkspaceSessionFollowsRenamedWorkspaceByFolderSet(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	oldPath := filepath.Join(folder, "old"+workspaceExtension)
	newPath := writeWorkspaceTestFileAt(t, folder, "new"+workspaceExtension)

	err := app.SetSession("", WindowSession{
		Workspace: oldPath,
		Folders:   []string{folder},
		OpenFiles: []string{filepath.Join(folder, "a.txt")},
		Active:    filepath.Join(folder, "a.txt"),
	})
	if err != nil {
		t.Fatal(err)
	}

	info, err := app.LoadWorkspace(newPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(folder, "a.txt")}
	if !reflect.DeepEqual(info.Session.OpenFiles, want) {
		t.Fatalf("adopted tabs = %q, want %q", info.Session.OpenFiles, want)
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
	if _, ok := state.Workspaces[canonicalPath(oldPath)]; ok {
		t.Error("the old workspace key survived adoption")
	}
	if _, ok := state.Workspaces[canonicalPath(newPath)]; !ok {
		t.Error("the renamed workspace has no parked session")
	}
}

func TestWorkspaceSessionDoesNotGuessBetweenDuplicateFolderSets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	first := writeWorkspaceTestFileAt(t, folder, "first"+workspaceExtension)
	second := writeWorkspaceTestFileAt(t, folder, "second"+workspaceExtension)
	target := writeWorkspaceTestFileAt(t, folder, "target"+workspaceExtension)
	// Both parked sets are orphans, so the folder set alone cannot say which
	// one the new target belongs to.
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}

	for _, workspace := range []string{first, second} {
		if err := app.SetSession("", WindowSession{
			Workspace: workspace,
			Folders:   []string{folder},
			OpenFiles: []string{filepath.Join(folder, "a.txt")},
		}); err != nil {
			t.Fatal(err)
		}
	}

	info, err := app.LoadWorkspace(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Session.OpenFiles) != 0 {
		t.Fatalf("parked tabs = %q, want no guess between two matches", info.Session.OpenFiles)
	}
}

func writeWorkspaceTestFileAt(t *testing.T, folder, name string) string {
	t.Helper()
	path := filepath.Join(folder, name)
	writeWorkspaceTestFile(t, path, `{"folders":["."]}`)
	return path
}

func TestLaterWindowsParkWithoutReplacingTheLaunchSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	app.registerWindow("main", emptyLaunchRequest())
	app.registerWindow("window-2", emptyLaunchRequest())
	folder := t.TempDir()
	workspace := writeWorkspaceTestFileAt(t, folder, "work"+workspaceExtension)

	if err := app.SetSession("main", WindowSession{OpenFiles: []string{"/main.txt"}}); err != nil {
		t.Fatal(err)
	}
	if err := app.SetSession("window-2", WindowSession{
		Workspace: workspace,
		Folders:   []string{folder},
		OpenFiles: []string{"/other.txt"},
	}); err != nil {
		t.Fatal(err)
	}

	main, err := app.GetWindowSession("main")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(main.OpenFiles, []string{"/main.txt"}) {
		t.Fatalf("main tabs = %q, want the owner's tabs", main.OpenFiles)
	}
	other, err := app.GetWindowSession("window-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(other.OpenFiles) != 0 {
		t.Fatalf("window-2 restored %q, want no launch session", other.OpenFiles)
	}
}

func TestWorkspaceSessionDoesNotStealACopiedWorkspace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	original := writeWorkspaceTestFileAt(t, folder, "original"+workspaceExtension)
	copied := writeWorkspaceTestFileAt(t, folder, "copy"+workspaceExtension)

	if err := app.SetSession("", WindowSession{
		Workspace: original,
		Folders:   []string{folder},
		OpenFiles: []string{filepath.Join(folder, "a.txt")},
	}); err != nil {
		t.Fatal(err)
	}

	info, err := app.LoadWorkspace(copied)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Session.OpenFiles) != 0 {
		t.Fatalf("copied workspace adopted %q, want no session", info.Session.OpenFiles)
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
	if _, ok := state.Workspaces[canonicalPath(original)]; !ok {
		t.Error("the original workspace lost its parked session")
	}
}

func TestSessionStoresSidebarRoots(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	folder := t.TempDir()

	app := NewApp()
	if _, err := app.AddSidebarFolder(folder); err != nil {
		t.Fatal(err)
	}

	roots, err := NewApp().LoadSidebarFolders()
	if err != nil {
		t.Fatalf("LoadSidebarFolders() error = %v", err)
	}
	want := []string{folder}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("roots = %q, want %q", roots, want)
	}
}

func TestSessionStoresWindowFrame(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	frame := WindowFrame{Width: 1000, Height: 700, X: 40, Y: 60, Maximised: true}

	NewApp().storeWindowGeometry(frame)

	got := NewApp().savedWindowGeometry()
	if got != frame {
		t.Fatalf("frame = %+v, want %+v", got, frame)
	}
}

func TestNormalizeWindowFrameDropsUnusable(t *testing.T) {
	if got := normalizeWindowFrame(WindowFrame{Width: minWindowWidth - 1, Height: 700}); got != (WindowFrame{}) {
		t.Fatalf("tiny frame = %+v, want the zero frame", got)
	}
	frame := WindowFrame{Width: minWindowWidth, Height: minWindowHeight}
	if got := normalizeWindowFrame(frame); got != frame {
		t.Fatalf("valid frame = %+v, want it kept", got)
	}
}
