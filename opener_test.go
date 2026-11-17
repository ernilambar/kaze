package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEditorArgsPrefersVisual(t *testing.T) {
	got, err := editorArgs("/tmp/settings.json", "code -w", "vim")
	if err != nil {
		t.Fatalf("editorArgs() error = %v", err)
	}

	want := []string{"code", "-w", "/tmp/settings.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editorArgs() = %q, want %q", got, want)
	}
}

func TestEditorArgsFallsBackToEditor(t *testing.T) {
	got, err := editorArgs("/tmp/settings.json", "  ", "nano")
	if err != nil {
		t.Fatalf("editorArgs() error = %v", err)
	}

	want := []string{"nano", "/tmp/settings.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editorArgs() = %q, want %q", got, want)
	}
}

func TestEditorArgsQuotedPathKeepsSpaces(t *testing.T) {
	command := `"/Applications/My Editor.app/Contents/MacOS/editor" -n`
	got, err := editorArgs("/tmp/settings.json", command, "")
	if err != nil {
		t.Fatalf("editorArgs() error = %v", err)
	}

	want := []string{"/Applications/My Editor.app/Contents/MacOS/editor", "-n", "/tmp/settings.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editorArgs() = %q, want %q", got, want)
	}
}

func TestEditorArgsDefaultOpener(t *testing.T) {
	got, err := editorArgs("/tmp/settings.json", "", "")
	if err != nil {
		t.Fatalf("editorArgs() error = %v", err)
	}

	want := []string{"open", "/tmp/settings.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editorArgs() = %q, want %q", got, want)
	}
}

func TestSplitCommandRejectsUnbalancedQuote(t *testing.T) {
	if _, err := splitCommand(`code "unterminated`); err == nil {
		t.Fatal("splitCommand() error = nil, want unbalanced quote error")
	}
}

func TestOpenSettingsFileCreatesAndLaunches(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VISUAL", "true")
	t.Setenv("EDITOR", "")

	app := NewApp()
	if err := app.openSettingsFile(); err != nil {
		t.Fatalf("openSettingsFile() error = %v", err)
	}

	path := filepath.Join(home, "Library", "Application Support", appMeta.ProductName, "settings.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file was not created: %v", err)
	}
}
