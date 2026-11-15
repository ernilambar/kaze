package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWorkspaceFileStoresInsideFoldersRelative(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "src")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	path := filepath.Join(dir, "project"+workspaceExtension)

	info, err := writeWorkspaceFile(path, []string{inside, outside})
	if err != nil {
		t.Fatalf("writeWorkspaceFile() error = %v", err)
	}
	if info.Name != "project" {
		t.Fatalf("workspace name = %q, want project", info.Name)
	}
	wantFolders := []string{inside, outside}
	if !reflect.DeepEqual(info.Folders, wantFolders) {
		t.Fatalf("folders = %q, want %q", info.Folders, wantFolders)
	}

	loaded, err := loadWorkspaceFile(path)
	if err != nil {
		t.Fatalf("loadWorkspaceFile() error = %v", err)
	}
	if !reflect.DeepEqual(loaded.Folders, wantFolders) {
		t.Fatalf("loaded folders = %q, want %q", loaded.Folders, wantFolders)
	}

	var file workspaceFile
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	wantEntries := []string{"src", filepath.ToSlash(outside)}
	if !reflect.DeepEqual(file.Folders, wantEntries) {
		t.Fatalf("stored folders = %q, want %q", file.Folders, wantEntries)
	}
}

func TestLoadWorkspaceFileResolvesRelativeFolders(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "team"+workspaceExtension)
	writeWorkspaceTestFile(t, path, `{"folders":["nested"]}`)

	info, err := loadWorkspaceFile(path)
	if err != nil {
		t.Fatalf("loadWorkspaceFile() error = %v", err)
	}
	want := []string{nested}
	if !reflect.DeepEqual(info.Folders, want) {
		t.Fatalf("folders = %q, want %q", info.Folders, want)
	}
}

func TestLoadWorkspaceFileRejectsMissingFolder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project"+workspaceExtension)
	writeWorkspaceTestFile(t, path, `{"folders":["missing"]}`)

	if _, err := loadWorkspaceFile(path); err == nil {
		t.Fatal("loadWorkspaceFile() error = nil, want a missing folder error")
	}
}

func TestWriteWorkspaceFileRequiresFolders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty"+workspaceExtension)

	if _, err := writeWorkspaceFile(path, nil); err == nil {
		t.Fatal("writeWorkspaceFile() error = nil, want an empty workspace error")
	}
}

func writeWorkspaceTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
