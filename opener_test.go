package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestEditorArgsPrefersVisual(t *testing.T) {
	got, err := editorArgs("/tmp/settings.json", "code -w", "vim", "darwin")
	if err != nil {
		t.Fatalf("editorArgs() error = %v", err)
	}

	want := []string{"code", "-w", "/tmp/settings.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editorArgs() = %q, want %q", got, want)
	}
}

func TestEditorArgsFallsBackToEditor(t *testing.T) {
	got, err := editorArgs("/tmp/settings.json", "  ", "nano", "darwin")
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
	got, err := editorArgs("/tmp/settings.json", command, "", "darwin")
	if err != nil {
		t.Fatalf("editorArgs() error = %v", err)
	}

	want := []string{"/Applications/My Editor.app/Contents/MacOS/editor", "-n", "/tmp/settings.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editorArgs() = %q, want %q", got, want)
	}
}

func TestEditorArgsDefaultOpenerPerPlatform(t *testing.T) {
	for _, tc := range []struct {
		goos string
		want []string
	}{
		{"darwin", []string{"open", "/tmp/settings.json"}},
		{"windows", []string{"rundll32", "url.dll,FileProtocolHandler", "/tmp/settings.json"}},
		{"linux", []string{"xdg-open", "/tmp/settings.json"}},
	} {
		got, err := editorArgs("/tmp/settings.json", "", "", tc.goos)
		if err != nil {
			t.Fatalf("editorArgs(%s) error = %v", tc.goos, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("editorArgs(%s) = %q, want %q", tc.goos, got, tc.want)
		}
	}
}

func TestSplitCommandRejectsUnbalancedQuote(t *testing.T) {
	if _, err := splitCommand(`code "unterminated`); err == nil {
		t.Fatal("splitCommand() error = nil, want unbalanced quote error")
	}
}

func TestOpenSettingsFileCreatesAndLaunches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test relies on the POSIX true command")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VISUAL", "true")
	t.Setenv("EDITOR", "")

	app := NewApp()
	if err := app.openSettingsFile(); err != nil {
		t.Fatalf("openSettingsFile() error = %v", err)
	}

	path := filepath.Join(home, "Library", "Application Support", "Kova", "settings.json")
	if runtime.GOOS != "darwin" {
		path = filepath.Join(home, ".config", "Kova", "settings.json")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file was not created: %v", err)
	}
}
