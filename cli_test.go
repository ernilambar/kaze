package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestParseLaunchRequestClassifiesPaths(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	got := parseLaunchRequest([]string{file, dir, "missing.txt"})

	want := LaunchRequest{
		Files:   []string{file},
		Folder:  dir,
		Missing: []string{"missing.txt"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseLaunchRequest() = %+v, want %+v", got, want)
	}
}

func TestParseLaunchRequestResolvesRelativePathsAgainstWorkingDir(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "notes.txt", "hello")
	t.Chdir(dir)

	got := parseLaunchRequest([]string{"notes.txt"})

	want := []string{filepath.Join(dir, "notes.txt")}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
}

func TestParseLaunchRequestSkipsFlagsAndTheirValues(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	args := []string{"-assetdir", dir, "--loglevel=Debug", "-appargs", "-- -x", "--", file}
	got := parseLaunchRequest(args)

	want := []string{file}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
	if got.Folder != "" {
		t.Fatalf("Folder = %q, want empty; a flag value is not a folder argument", got.Folder)
	}
}

func TestParseLaunchRequestTreatsPathsAfterDoubleDashAsPaths(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "-weird.txt", "hello")

	got := parseLaunchRequest([]string{"--", file})

	want := []string{file}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
}

func TestParseLaunchRequestLastFolderWins(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()

	got := parseLaunchRequest([]string{first, second})

	if got.Folder != second {
		t.Fatalf("Folder = %q, want %q", got.Folder, second)
	}
}

func TestLaunchRequestMergeKeepsOrderAndLatestFolder(t *testing.T) {
	request := LaunchRequest{Files: []string{"/a.txt"}}
	request.Merge(LaunchRequest{Files: []string{"/b.txt"}, Folder: "/one", Missing: []string{"x"}})
	request.Merge(LaunchRequest{Folder: "/two"})

	want := LaunchRequest{
		Files:   []string{"/a.txt", "/b.txt"},
		Folder:  "/two",
		Missing: []string{"x"},
	}
	if !reflect.DeepEqual(request, want) {
		t.Fatalf("Merge() = %+v, want %+v", request, want)
	}
}

func TestEmptyLaunchRequestEncodesSlicesNotEmpty(t *testing.T) {
	request := emptyLaunchRequest()

	if request.Files == nil || request.Missing == nil {
		t.Fatal("emptyLaunchRequest() must use non-nil slices so the UI receives arrays")
	}
	if !request.Empty() {
		t.Fatal("emptyLaunchRequest() must report Empty")
	}
}

func TestQueueLaunchBeforeListeningKeepsRequest(t *testing.T) {
	app := NewApp()
	app.queueLaunch(parseLaunchRequest([]string{t.TempDir()}))

	got := app.GetLaunchRequest()
	if got.Folder == "" {
		t.Fatal("GetLaunchRequest() lost a request queued before the UI was listening")
	}
	if !app.GetLaunchRequest().Empty() {
		t.Fatal("GetLaunchRequest() must hand a request over only once")
	}
}
