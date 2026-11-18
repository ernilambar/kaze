package workspacefile

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
	path := filepath.Join(dir, "project"+Extension)

	info, err := Write(path, []string{inside, outside})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if info.Name != "project" {
		t.Fatalf("workspace name = %q, want project", info.Name)
	}
	wantFolders := []string{inside, outside}
	if !reflect.DeepEqual(info.Folders, wantFolders) {
		t.Fatalf("folders = %q, want %q", info.Folders, wantFolders)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
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
	path := filepath.Join(dir, "team"+Extension)
	writeWorkspaceTestFile(t, path, `{"folders":["nested"]}`)

	info, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []string{nested}
	if !reflect.DeepEqual(info.Folders, want) {
		t.Fatalf("folders = %q, want %q", info.Folders, want)
	}
}

func TestLoadWorkspaceFileRejectsMissingFolder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project"+Extension)
	writeWorkspaceTestFile(t, path, `{"folders":["missing"]}`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want a missing folder error")
	}
}

func TestWriteWorkspaceFileRequiresFolders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty"+Extension)

	if _, err := Write(path, nil); err == nil {
		t.Fatal("Write() error = nil, want an empty workspace error")
	}
}

func writeWorkspaceTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCodeWorkspaceReadsFoldersOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"app", "lib dir"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	outside := t.TempDir()
	path := filepath.Join(dir, "project"+CodeExtension)
	data := `{
	// VS Code writes JSONC.
	"folders": [
		{ "path": "app", "name": "Application" },
		{ "uri": "file://` + filepath.ToSlash(filepath.Join(dir, "lib%20dir")) + `" },
		/* absolute */ { "path": "` + filepath.ToSlash(outside) + `", },
	],
	"settings": { "files.exclude": { "**/,]": true, }, },
	"extensions": { "recommendations": [] },
}
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []string{filepath.Join(dir, "app"), filepath.Join(dir, "lib dir"), outside}
	if !reflect.DeepEqual(info.Folders, want) {
		t.Fatalf("folders = %q, want %q", info.Folders, want)
	}
	if info.Name != "project" {
		t.Fatalf("workspace name = %q, want project", info.Name)
	}
}

func TestCodeWorkspaceRejectsRemoteFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote"+CodeExtension)
	data := `{"folders": [{"uri": "vscode-remote://ssh-remote+host/home/me/app"}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want a remote folder error")
	}
}
