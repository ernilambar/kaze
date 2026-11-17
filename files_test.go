package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// listFiles is listFilesExcluding with the first-run patterns, which is what
// an untouched settings file produces.
func listFiles(roots []string) []string {
	return listFilesExcluding(roots, newExcludeMatcher(defaultExcludePatterns())).Files
}

func TestListFilesWalksRootsAndSkipsGeneratedTrees(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "main.go", "package main\n")
	for _, dir := range []string{
		filepath.Join("src", "app"),
		filepath.Join("node_modules", "pkg"),
		".git",
		".hg",
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}
	writeTempFile(t, filepath.Join(root, "src", "app"), "app.ts", "export {};\n")
	writeTempFile(t, filepath.Join(root, "node_modules", "pkg"), "index.js", "\n")
	writeTempFile(t, filepath.Join(root, ".git"), "config", "\n")
	writeTempFile(t, filepath.Join(root, ".hg"), "hgrc", "\n")

	got := listFiles([]string{root})
	want := []string{
		filepath.Join(root, "main.go"),
		filepath.Join(root, "src", "app", "app.ts"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %q, want %q", got, want)
	}
}

func TestListFilesDeduplicatesOverlappingRoots(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, src, "main.go", "package main\n")

	got := listFiles([]string{root, src})
	want := []string{filepath.Join(src, "main.go")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %q, want %q", got, want)
	}
}

func TestListFilesSkipsMissingAndSymlinkedRoots(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "notes.txt", "hello\n")
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got := listFiles([]string{filepath.Join(root, "missing"), root})
	want := []string{filepath.Join(root, "notes.txt")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %q, want %q", got, want)
	}
}

func TestListFilesReturnsEmptySliceWithoutRoots(t *testing.T) {
	got := listFiles(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("listFiles() = %#v, want a non-nil empty slice", got)
	}
}

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

func TestListFilesHonoursGitignore(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, ".gitignore", "dist/\n*.log\n")
	writeTempFile(t, root, "main.go", "package main\n")
	writeTempFile(t, root, "debug.log", "noise\n")
	if err := os.MkdirAll(filepath.Join(root, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, filepath.Join(root, "dist"), "bundle.js", "built\n")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, filepath.Join(root, ".git"), "config", "\n")

	got := listFiles([]string{root})
	want := []string{
		filepath.Join(root, ".gitignore"),
		filepath.Join(root, "main.go"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listFiles() = %q, want %q", got, want)
	}
}

func TestListFilesReportsTruncationAtTheCap(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		writeTempFile(t, root, name, "text\n")
	}

	list := listFilesCapped([]string{root}, newExcludeMatcher(defaultExcludePatterns()), 2)
	if !list.Truncated {
		t.Fatal("Truncated = false, want the cap reported")
	}
	if len(list.Files) != 2 {
		t.Fatalf("files = %q, want exactly the cap", list.Files)
	}
}

func TestListFilesUsesTheCallingWindowsRoots(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	app.registerWindow(defaultWindowName, emptyLaunchRequest())
	app.registerWindow("window-2", emptyLaunchRequest())
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
