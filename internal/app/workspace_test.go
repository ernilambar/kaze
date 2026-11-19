package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"kaze/internal/pathutil"
	"kaze/internal/workspacefile"
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
	stored, err := readSessionState()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored.Recent.Files, want) {
		t.Fatalf("session recent files = %q, want %q", stored.Recent.Files, want)
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
	files, err = app.GetRecentFiles()
	if err != nil {
		t.Fatal(err)
	}
	if files == nil || len(files) != 0 {
		t.Fatalf("GetRecentFiles() = %#v, want a non-nil empty list", files)
	}
}

func TestLoadMigratesRecentFilesOutOfSettings(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	settingsPath, err := settingsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	settingsData := []byte(`{
  "editor": { "font_family": "Menlo" },
  "recent": { "files": ["/old/one.txt", " /old/two.txt ", "/old/one.txt"] },
  "custom": { "keep": true }
}`)
	if err := os.WriteFile(settingsPath, settingsData, 0o644); err != nil {
		t.Fatal(err)
	}

	sessionPath, err := sessionFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte(`{"version":1,"window":{"open_files":["/a.txt"]},"workspaces":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	files, err := app.GetRecentFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/old/one.txt", "/old/two.txt"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("recent files = %q, want %q", files, want)
	}

	updated, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(updated, []byte(`"recent"`)) {
		t.Fatalf("settings.json kept the recent section:\n%s", updated)
	}
	if !bytes.Contains(updated, []byte(`"custom": { "keep": true }`)) {
		t.Fatalf("settings.json lost unrelated content:\n%s", updated)
	}

	state, err := readSessionState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != sessionVersion {
		t.Fatalf("session version = %d, want %d", state.Version, sessionVersion)
	}
	if !reflect.DeepEqual(state.Recent.Files, want) {
		t.Fatalf("session recent files = %q, want %q", state.Recent.Files, want)
	}
	if !reflect.DeepEqual(state.Window.OpenFiles, []string{"/a.txt"}) {
		t.Fatalf("session tabs = %q, want the existing tab to survive", state.Window.OpenFiles)
	}
}

// readSessionState reads the on-disk session file, bypassing the in-memory
// store so a test sees exactly what was written.
func readSessionState() (sessionFile, error) {
	path, err := sessionFilePath()
	if err != nil {
		return sessionFile{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionFile{}, err
	}
	var state sessionFile
	err = json.Unmarshal(data, &state)
	return state, err
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
	if _, err := app.AddSidebarFolder(defaultWindowName, folder); err != nil {
		t.Fatal(err)
	}

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Sidebar.Visible {
		t.Fatal("Sidebar.Visible = true after it was hidden")
	}

	if _, err := app.RemoveAllSidebarFolders(defaultWindowName); err != nil {
		t.Fatal(err)
	}
	if roots := savedSidebarRoots(t); len(roots) != 0 {
		t.Fatalf("sidebar roots = %q after RemoveAllSidebarFolders", roots)
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

func TestRecentWorkspacesAreNewestFirstAndSkipMissingFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	folder := t.TempDir()
	// The names sort in the opposite order to their timestamps, so the test
	// fails if the list is ordered by path instead of by use.
	oldest := writeWorkspaceTestFileAt(t, folder, "a-oldest"+workspacefile.Extension)
	newest := writeWorkspaceTestFileAt(t, folder, "c-newest"+workspacefile.Extension)
	middle := writeWorkspaceTestFileAt(t, folder, "b-middle"+workspacefile.Extension)

	state := emptySession()
	for path, updated := range map[string]int64{
		oldest: 100,
		middle: 200,
		newest: 300,
	} {
		state.Workspaces[pathutil.CanonicalPath(path)] = workspacefile.WorkspaceSession{Updated: updated}
	}
	if err := writeSessionFile(state); err != nil {
		t.Fatal(err)
	}

	// A parked workspace whose file is gone must not be offered: the row could
	// not open.
	if err := os.Remove(middle); err != nil {
		t.Fatal(err)
	}

	paths, err := app.GetRecentWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{pathutil.CanonicalPath(newest), pathutil.CanonicalPath(oldest)}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("recent workspaces = %q, want %q", paths, want)
	}
}

func TestRecentWorkspacesAreEmptyWithoutHistory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	paths, err := NewApp().GetRecentWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("recent workspaces = %q, want none", paths)
	}
}
