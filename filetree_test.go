package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateFileCreatesAnEmptyFile(t *testing.T) {
	dir := t.TempDir()

	result, err := NewApp().CreateFile(dir, "notes.txt")
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}

	path := filepath.Join(dir, "notes.txt")
	if result.Path != path {
		t.Fatalf("CreateFile() path = %q, want %q", result.Path, path)
	}
	if result.Name != "notes.txt" || result.IsDir {
		t.Fatalf("CreateFile() = %+v, want a file named notes.txt", result)
	}
	info, err := os.Stat(result.RealPath)
	if err != nil {
		t.Fatalf("stat %s: %v", result.RealPath, err)
	}
	if info.Size() != 0 {
		t.Fatalf("CreateFile() wrote %d bytes, want an empty file", info.Size())
	}
}

func TestCreateFileRefusesAnExistingName(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "notes.txt", "hello")

	if _, err := NewApp().CreateFile(dir, "notes.txt"); err == nil {
		t.Fatal("CreateFile() error = nil, want it to refuse an existing file")
	}
	// The refusal must not have truncated what was there.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "hello" {
		t.Fatalf("existing file = %q, want it untouched", data)
	}
}

func TestCreateFolderCreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := NewApp().CreateFolder(dir, "src")
	if err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	if !result.IsDir {
		t.Fatalf("CreateFolder() = %+v, want a directory", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "src")); err != nil {
		t.Fatalf("stat new folder: %v", err)
	}
}

func TestTreeNamesCannotWalkUpTheTree(t *testing.T) {
	dir := t.TempDir()
	app := NewApp()

	for _, name := range []string{"", "  ", "..", "../escape.txt", "a/b.txt"} {
		if _, err := app.CreateFile(dir, name); err == nil {
			t.Fatalf("CreateFile(%q) error = nil, want a rejected name", name)
		}
		if _, err := app.CreateFolder(dir, name); err == nil {
			t.Fatalf("CreateFolder(%q) error = nil, want a rejected name", name)
		}
		if _, err := app.RenamePath(filepath.Join(dir, "missing.txt"), name); err == nil {
			t.Fatalf("RenamePath(%q) error = nil, want a rejected name", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "escape.txt")); err == nil {
		t.Fatal("a rejected name created a file outside the folder")
	}
}

func TestCreateFileRejectsAFileAsParent(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "notes.txt", "hello")

	_, err := NewApp().CreateFile(path, "child.txt")
	if err == nil {
		t.Fatal("CreateFile() error = nil, want a not a folder error")
	}
	if !strings.Contains(err.Error(), "not a folder") {
		t.Fatalf("CreateFile() error = %v, want it to say the parent is not a folder", err)
	}
}

func TestRenamePathKeepsTheFileInItsFolder(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")

	result, err := NewApp().RenamePath(source, "journal.txt")
	if err != nil {
		t.Fatalf("RenamePath() error = %v", err)
	}

	want := filepath.Join(dir, "journal.txt")
	if result.Path != want {
		t.Fatalf("RenamePath() path = %q, want %q", result.Path, want)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("old path still exists: %v", err)
	}
	data, err := os.ReadFile(want)
	if err != nil || string(data) != "hello" {
		t.Fatalf("renamed file = %q, %v; want its content kept", data, err)
	}
}

func TestRenamePathRefusesATakenName(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")
	writeTempFile(t, dir, "taken.txt", "other")

	_, err := NewApp().RenamePath(source, "taken.txt")
	if err == nil {
		t.Fatal("RenamePath() error = nil, want it to refuse an existing file")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("RenamePath() error = %v, want an already exists error", err)
	}
}

func TestRenamePathAcceptsItsOwnName(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")
	app := NewApp()

	// Submitting the unchanged name is not an "already exists" failure.
	result, err := app.RenamePath(source, "notes.txt")
	if err != nil {
		t.Fatalf("RenamePath() error = %v", err)
	}
	if result.Path != source {
		t.Fatalf("RenamePath() path = %q, want %q", result.Path, source)
	}

	// A case-only rename is how a macOS volume reports the same file.
	if _, err := app.RenamePath(source, "Notes.txt"); err != nil {
		t.Fatalf("RenamePath() case change error = %v", err)
	}
}

func TestRenamePathRefusesATakenSibling(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	for _, path := range []string{first, second} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
	}

	// Renaming a folder to a name its folder already holds is refused, so two
	// folders never merge by accident.
	if _, err := NewApp().RenamePath(first, "second"); err == nil {
		t.Fatal("RenamePath() error = nil, want it to refuse a taken name")
	}
	if _, err := os.Stat(filepath.Join(second, filepath.Base(first))); err == nil {
		t.Fatal("a refused rename moved the folder into the taken one")
	}
}

func TestMovePathRefusesAFolderInsideItself(t *testing.T) {
	dir := t.TempDir()
	inner := filepath.Join(dir, "inner")
	if err := os.MkdirAll(filepath.Join(inner, "deep"), 0o755); err != nil {
		t.Fatalf("create folders: %v", err)
	}

	if _, err := NewApp().MovePath(dir, filepath.Join(inner, "deep")); err == nil {
		t.Fatal("MovePath() error = nil, want it to refuse moving a folder into itself")
	}
	if _, err := os.Stat(filepath.Join(inner, "deep", filepath.Base(dir))); err == nil {
		t.Fatal("a refused move created a folder inside the source")
	}
}

func TestDuplicatePathCopiesNextToTheSource(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")

	result, err := NewApp().DuplicatePath(source)
	if err != nil {
		t.Fatalf("DuplicatePath() error = %v", err)
	}

	want := filepath.Join(dir, "notes copy.txt")
	if result.Path != want {
		t.Fatalf("DuplicatePath() path = %q, want %q", result.Path, want)
	}
	data, err := os.ReadFile(want)
	if err != nil || string(data) != "hello" {
		t.Fatalf("duplicate = %q, %v; want a copy of the file", data, err)
	}
	// The original is untouched, so a duplicate is never a move.
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("original disappeared: %v", err)
	}
}

func TestDuplicatePathUsesItsOwnNameBesideItself(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")
	app := NewApp()

	var paths []string
	for range 3 {
		result, err := app.DuplicatePath(source)
		if err != nil {
			t.Fatalf("DuplicatePath() error = %v", err)
		}
		paths = append(paths, filepath.Base(result.Path))
	}

	// Duplicating beside itself must not collide with the name it already has.
	want := []string{"notes copy.txt", "notes copy 1.txt", "notes copy 2.txt"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("duplicate names = %q, want %q", paths, want)
	}
}

func TestDuplicatePathCopiesADirectoryTree(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("create folders: %v", err)
	}
	writeTempFile(t, source, "main.go", "package main\n")
	writeTempFile(t, filepath.Join(source, "nested"), "notes.txt", "hello")

	result, err := NewApp().DuplicatePath(source)
	if err != nil {
		t.Fatalf("DuplicatePath() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Path, "nested", "notes.txt")); err != nil {
		t.Fatalf("duplicate is missing a nested file: %v", err)
	}
}

func TestCopyPathPutsTheCopyInTheTargetFolder(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")
	target := filepath.Join(dir, "archive")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create target: %v", err)
	}

	result, err := NewApp().CopyPath(source, target)
	if err != nil {
		t.Fatalf("CopyPath() error = %v", err)
	}
	want := filepath.Join(target, "notes.txt")
	if result.Path != want {
		t.Fatalf("CopyPath() path = %q, want %q", result.Path, want)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("original disappeared: %v", err)
	}
}

func TestCopyPathRejectsAFileAsTarget(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")
	target := writeTempFile(t, dir, "archive.txt", "")

	_, err := NewApp().CopyPath(source, target)
	if err == nil {
		t.Fatal("CopyPath() error = nil, want a not a folder error")
	}
	if !strings.Contains(err.Error(), "not a folder") {
		t.Fatalf("CopyPath() error = %v, want it to say the target is not a folder", err)
	}
}

func TestMovePathMovesFileIntoFolder(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")
	target := filepath.Join(dir, "archive")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create target: %v", err)
	}

	result, err := NewApp().MovePath(source, target)
	if err != nil {
		t.Fatalf("MovePath() error = %v", err)
	}
	want := filepath.Join(target, "notes.txt")
	if result.Path != want {
		t.Fatalf("MovePath() path = %q, want %q", result.Path, want)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists after the move: %v", err)
	}
	data, err := os.ReadFile(want)
	if err != nil || string(data) != "hello" {
		t.Fatalf("moved file = %q, %v; want its content kept", data, err)
	}
}

func TestMovePathIntoItsOwnFolderChangesNothing(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")

	result, err := NewApp().MovePath(source, dir)
	if err != nil {
		t.Fatalf("MovePath() error = %v", err)
	}
	if result.Path != source {
		t.Fatalf("MovePath() path = %q, want %q", result.Path, source)
	}
}

func TestMovePathRefusesATakenName(t *testing.T) {
	dir := t.TempDir()
	source := writeTempFile(t, dir, "notes.txt", "hello")
	target := filepath.Join(dir, "archive")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create target: %v", err)
	}
	writeTempFile(t, target, "notes.txt", "other")

	if _, err := NewApp().MovePath(source, target); err == nil {
		t.Fatal("MovePath() error = nil, want it to refuse an existing file")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source disappeared after a refused move: %v", err)
	}
}

func TestTrashPathRejectsAnEmptyPath(t *testing.T) {
	if err := NewApp().TrashPath("  "); err == nil {
		t.Fatal("TrashPath() error = nil, want a rejected path")
	}
}

func TestTrashPathReportsAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.txt")

	err := NewApp().TrashPath(path)
	if err == nil {
		t.Fatal("TrashPath() error = nil, want a missing file error")
	}
	if !strings.Contains(err.Error(), "Trash") {
		t.Fatalf("TrashPath() error = %v, want it to mention the Trash", err)
	}
}

func TestIsWithinPath(t *testing.T) {
	cases := []struct {
		folder string
		path   string
		want   bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/b", "/a/b/c", true},
		{"/a/b", "/a/bc", false},
		{"/a/b", "/a", false},
	}
	for _, entry := range cases {
		if got := isWithinPath(entry.folder, entry.path); got != entry.want {
			t.Errorf("isWithinPath(%q, %q) = %v, want %v", entry.folder, entry.path, got, entry.want)
		}
	}
}
