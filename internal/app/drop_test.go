package app

import (
	"reflect"
	"testing"

	"kaze/internal/cli"
)

func TestDropLaunchRequestOpensFilesAndAddsFolders(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	got := dropLaunchRequest([]string{file, dir}, false)

	if want := []string{file}; !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
	if want := []string{dir}; !reflect.DeepEqual(got.Folders, want) {
		t.Fatalf("Folders = %q, want %q", got.Folders, want)
	}
	if got.NewWindow || got.ReplaceFolders {
		t.Fatalf("request = %+v, want the dropped window to keep its roots", got)
	}
}

func TestDropLaunchRequestWithOptionOpensNewWindow(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	got := dropLaunchRequest([]string{file, dir}, true)

	if !got.NewWindow {
		t.Fatal("NewWindow = false with Option held")
	}
	if !got.ReplaceFolders {
		t.Fatal("ReplaceFolders = false, want a dropped folder to become the window's root")
	}
	if want := []string{file}; !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
	if want := []string{dir}; !reflect.DeepEqual(got.Folders, want) {
		t.Fatalf("Folders = %q, want %q", got.Folders, want)
	}
}

func TestDropLaunchRequestWithOptionKeepsFilesOnlyWindowRoots(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	got := dropLaunchRequest([]string{file}, true)

	if !got.NewWindow {
		t.Fatal("NewWindow = false with Option held")
	}
	if got.ReplaceFolders {
		t.Fatal("ReplaceFolders = true for a drop with no folder")
	}
}

func TestDropLaunchRequestWithOptionAndNothingDoesNotOpenWindow(t *testing.T) {
	got := dropLaunchRequest(nil, true)

	if !got.Empty() {
		t.Fatalf("request = %+v, want an empty drop to stay empty", got)
	}
	if got.NewWindow {
		t.Fatal("NewWindow = true for a drop with nothing to open")
	}
}

func TestHandleFileDropUsesTheWindowItLandedIn(t *testing.T) {
	app := NewApp()
	first := app.registerWindow("main", cli.EmptyLaunchRequest())
	dropped := app.registerWindow("window-2", cli.EmptyLaunchRequest())
	file := writeTempFile(t, t.TempDir(), "notes.txt", "hello")

	app.handleFileDrop(dropped, []string{file}, false)

	if request := first.takeLaunch(); !request.Empty() {
		t.Fatalf("the first window received %+v, want the drop to stay in its window", request)
	}
	request := dropped.takeLaunch()
	if want := []string{file}; !reflect.DeepEqual(request.Files, want) {
		t.Fatalf("dropped window received %q, want %q", request.Files, want)
	}
}
