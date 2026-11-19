package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"kaze/internal/cli"
	"kaze/internal/pathutil"
	"kaze/internal/workspacefile"
)

func TestWorkspaceSessionParksTabsUnderCanonicalPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	workspace := writeWorkspaceTestFileAt(t, folder, "work"+workspacefile.Extension)

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

func TestSessionKeepsSidebarWidthPerWorkspace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	workspace := writeWorkspaceTestFileAt(t, folder, "work"+workspacefile.Extension)

	if err := app.SetSession("", WindowSession{Workspace: workspace, SidebarWidth: 320}); err != nil {
		t.Fatal(err)
	}
	if err := app.SetSession("", WindowSession{SidebarWidth: -5}); err != nil {
		t.Fatal(err)
	}

	info, err := app.LoadWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if info.Session.SidebarWidth != 320 {
		t.Fatalf("parked width = %d, want 320", info.Session.SidebarWidth)
	}
	window, err := app.GetWindowSession("")
	if err != nil {
		t.Fatal(err)
	}
	if window.SidebarWidth != 0 {
		t.Fatalf("window width = %d, want 0", window.SidebarWidth)
	}
}

func TestWorkspaceSessionFollowsRenamedWorkspaceByFolderSet(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	oldPath := filepath.Join(folder, "old"+workspacefile.Extension)
	newPath := writeWorkspaceTestFileAt(t, folder, "new"+workspacefile.Extension)

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
	if _, ok := state.Workspaces[pathutil.CanonicalPath(oldPath)]; ok {
		t.Error("the old workspace key survived adoption")
	}
	if _, ok := state.Workspaces[pathutil.CanonicalPath(newPath)]; !ok {
		t.Error("the renamed workspace has no parked session")
	}
}

func TestWorkspaceSessionDoesNotGuessBetweenDuplicateFolderSets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	first := writeWorkspaceTestFileAt(t, folder, "first"+workspacefile.Extension)
	second := writeWorkspaceTestFileAt(t, folder, "second"+workspacefile.Extension)
	target := writeWorkspaceTestFileAt(t, folder, "target"+workspacefile.Extension)
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
	app.registerWindow("main", cli.EmptyLaunchRequest())
	app.registerWindow("window-2", cli.EmptyLaunchRequest())
	folder := t.TempDir()
	workspace := writeWorkspaceTestFileAt(t, folder, "work"+workspacefile.Extension)

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
	original := writeWorkspaceTestFileAt(t, folder, "original"+workspacefile.Extension)
	copied := writeWorkspaceTestFileAt(t, folder, "copy"+workspacefile.Extension)

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
	if _, ok := state.Workspaces[pathutil.CanonicalPath(original)]; !ok {
		t.Error("the original workspace lost its parked session")
	}
}

func TestSessionStoresSidebarRoots(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	folder := t.TempDir()

	app := NewApp()
	if _, err := app.AddSidebarFolder(defaultWindowName, folder); err != nil {
		t.Fatal(err)
	}

	roots, err := NewApp().LoadSidebarFolders(defaultWindowName)
	if err != nil {
		t.Fatalf("LoadSidebarFolders() error = %v", err)
	}
	want := []string{folder}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("roots = %q, want %q", roots, want)
	}
}

// A session written before roots were keyed by window must still restore its
// sidebar; otherwise the upgrade silently empties the folder tree.
func TestSessionAdoptsLegacySidebarRoots(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	var state sessionFile
	if err := json.Unmarshal([]byte(`{"version":2,"sidebar":{"roots":["`+root+`"]}}`), &state); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if err := writeSessionFile(normalizeSession(state)); err != nil {
		t.Fatal(err)
	}

	roots, err := NewApp().LoadSidebarFolders(defaultWindowName)
	if err != nil {
		t.Fatalf("LoadSidebarFolders() error = %v", err)
	}
	if want := []string{root}; !reflect.DeepEqual(roots, want) {
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

func TestSessionStoresMaximisedWithoutNormalFrame(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	NewApp().storeWindowGeometry(WindowFrame{Maximised: true})

	got := NewApp().savedWindowGeometry()
	if want := (WindowFrame{Maximised: true}); got != want {
		t.Fatalf("frame = %+v, want %+v", got, want)
	}
}

func TestNormalizeWindowFrameDropsUnusable(t *testing.T) {
	if got := normalizeWindowFrame(WindowFrame{Width: minWindowWidth - 1, Height: 700}); got != (WindowFrame{}) {
		t.Fatalf("tiny frame = %+v, want the zero frame", got)
	}
	zoomed := WindowFrame{Maximised: true}
	if got := normalizeWindowFrame(zoomed); got != zoomed {
		t.Fatalf("sizeless zoomed frame = %+v, want the flag kept", got)
	}
	frame := WindowFrame{Width: minWindowWidth, Height: minWindowHeight}
	if got := normalizeWindowFrame(frame); got != frame {
		t.Fatalf("valid frame = %+v, want it kept", got)
	}
}

func writeWorkspaceTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
