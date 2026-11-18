package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"kaze/internal/cli"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestQueueLaunchNewWindowBeforeTheFirstWindowIsAdopted(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	app.queueLaunch(cli.ParseLaunchRequest([]string{"--new-window", dir}))
	app.registerWindow("main", cli.EmptyLaunchRequest())

	got := app.GetLaunchRequest("main")
	if want := []string{dir}; !reflect.DeepEqual(got.Folders, want) {
		t.Fatalf("Folders = %q, want %q", got.Folders, want)
	}
	if got.NewWindow {
		t.Fatal("NewWindow = true, want the flag consumed before the frontend sees it")
	}
}

func TestQueueLaunchBeforeListeningKeepsRequest(t *testing.T) {
	app := NewApp()
	app.registerWindow("main", cli.EmptyLaunchRequest())
	app.queueLaunch(cli.ParseLaunchRequest([]string{t.TempDir()}))

	got := app.GetLaunchRequest("main")
	if len(got.Folders) == 0 {
		t.Fatal("GetLaunchRequest() lost a request queued before the UI was listening")
	}
	if !app.GetLaunchRequest("main").Empty() {
		t.Fatal("GetLaunchRequest() must hand a request over only once")
	}
}

func TestCLITextReportsEmbeddedVersion(t *testing.T) {
	meta := loadAppMetadata()
	if meta.ProductVersion == "" {
		t.Fatal("loadAppMetadata() must expose info.productVersion from wails.json")
	}
	want := commandName + " " + meta.ProductVersion + "\n"
	if got := cliVersionText(meta); got != want {
		t.Fatalf("cliVersionText() = %q, want %q", got, want)
	}
}
