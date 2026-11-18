package cli

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseLaunchRequestClassifiesPaths(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	got := ParseLaunchRequest([]string{file, dir, "missing.txt"})

	want := LaunchRequest{
		Files:   []string{file},
		Folders: []string{dir},
		Missing: []string{"missing.txt"},
		Buffers: []LaunchBuffer{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseLaunchRequest() = %+v, want %+v", got, want)
	}
}

func TestParseLaunchRequestKeepsEveryFolder(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()

	got := ParseLaunchRequest([]string{first, second})

	want := []string{first, second}
	if !reflect.DeepEqual(got.Folders, want) {
		t.Fatalf("Folders = %q, want %q", got.Folders, want)
	}
}

func TestParseLaunchRequestResolvesRelativePathsAgainstWorkingDir(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "notes.txt", "hello")
	t.Chdir(dir)

	got := ParseLaunchRequest([]string{"notes.txt"})

	want := []string{filepath.Join(dir, "notes.txt")}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
}

func TestParseLaunchRequestSkipsFlags(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	// v3's dev tooling passes its settings through the environment, so the only
	// argv flags the app sees are ones a user typed. They are ignored, and their
	// values are not mistaken for paths.
	got := ParseLaunchRequest([]string{"--loglevel=Debug", "-dev", "--", file})

	want := []string{file}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
	if len(got.Folders) != 0 {
		t.Fatalf("Folders = %q, want empty; a flag is not a folder argument", got.Folders)
	}
}

func TestParseLaunchRequestRecognizesNewWindow(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "hello")

	for _, flag := range []string{"--new-window", "-new-window", "-n"} {
		t.Run(flag, func(t *testing.T) {
			got := ParseLaunchRequest([]string{flag, file})

			if !got.NewWindow {
				t.Fatalf("NewWindow = false, want %s recognized", flag)
			}
			if want := []string{file}; !reflect.DeepEqual(got.Files, want) {
				t.Fatalf("Files = %q, want %q", got.Files, want)
			}
			if len(got.Missing) != 0 {
				t.Fatalf("Missing = %q, want empty; the flag is not a path", got.Missing)
			}
		})
	}
}

func TestParseLaunchRequestNewWindowAfterDoubleDashIsAPath(t *testing.T) {
	for _, flag := range []string{"--new-window", "-n"} {
		t.Run(flag, func(t *testing.T) {
			got := ParseLaunchRequest([]string{"--", flag})

			if got.NewWindow {
				t.Fatalf("NewWindow = true, want %s after -- to be a path", flag)
			}
			if want := []string{flag}; !reflect.DeepEqual(got.Missing, want) {
				t.Fatalf("Missing = %q, want %q", got.Missing, want)
			}
		})
	}
}

func TestNewWindowOnlyRequestIsNotEmpty(t *testing.T) {
	if (LaunchRequest{NewWindow: true}).Empty() {
		t.Fatal("Empty() = true for a --new-window request, want a window to open")
	}
}

func TestParseLaunchRequestTreatsPathsAfterDoubleDashAsPaths(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "-weird.txt", "hello")

	got := ParseLaunchRequest([]string{"--", file})

	want := []string{file}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
}

func TestLaunchRequestMergeKeepsOrderAndFolders(t *testing.T) {
	request := LaunchRequest{Files: []string{"/a.txt"}}
	request.Merge(LaunchRequest{Files: []string{"/b.txt"}, Folders: []string{"/one"}, Missing: []string{"x"}})
	request.Merge(LaunchRequest{Folders: []string{"/two"}})

	want := LaunchRequest{
		Files:   []string{"/a.txt", "/b.txt"},
		Folders: []string{"/one", "/two"},
		Missing: []string{"x"},
	}
	if !reflect.DeepEqual(request, want) {
		t.Fatalf("Merge() = %+v, want %+v", request, want)
	}
}

func TestEmptyLaunchRequestEncodesSlicesNotEmpty(t *testing.T) {
	request := EmptyLaunchRequest()

	if request.Files == nil || request.Folders == nil || request.Missing == nil {
		t.Fatal("EmptyLaunchRequest() must use non-nil slices so the UI receives arrays")
	}
	if !request.Empty() {
		t.Fatal("EmptyLaunchRequest() must report Empty")
	}
}

func TestNormalizedDropsRepeatedFolders(t *testing.T) {
	request := LaunchRequest{Folders: []string{"/a", "/b", "/a"}}.Normalized()

	want := []string{"/a", "/b"}
	if !reflect.DeepEqual(request.Folders, want) {
		t.Fatalf("Folders = %q, want %q", request.Folders, want)
	}
}

func TestParseCLIActionRecognizesInformationalFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Action
	}{
		{name: "no arguments", args: nil, want: ActionLaunch},
		{name: "paths only", args: []string{"notes.txt", "."}, want: ActionLaunch},
		{name: "unrelated flag", args: []string{"--loglevel=Debug"}, want: ActionLaunch},
		{name: "short help", args: []string{"-h"}, want: ActionHelp},
		{name: "single dash help", args: []string{"-help"}, want: ActionHelp},
		{name: "long help", args: []string{"--help"}, want: ActionHelp},
		{name: "short version", args: []string{"-v"}, want: ActionVersion},
		{name: "capital version", args: []string{"-V"}, want: ActionVersion},
		{name: "single dash version", args: []string{"-version"}, want: ActionVersion},
		{name: "long version", args: []string{"--version"}, want: ActionVersion},
		{name: "help after a path", args: []string{"notes.txt", "--help"}, want: ActionHelp},
		{name: "help wins over version", args: []string{"--version", "--help"}, want: ActionHelp},
		{name: "flag after double dash is a path", args: []string{"--", "--help"}, want: ActionLaunch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ParseAction(test.args); got != test.want {
				t.Fatalf("ParseAction(%q) = %v, want %v", test.args, got, test.want)
			}
		})
	}
}
