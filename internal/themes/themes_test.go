package themes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kaze/internal/config"
)

func TestEnsureThemesDirSeedsOnce(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	path := filepath.Join(dir, "kaze.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("built-in theme was not written: %v", err)
	}

	if err := os.WriteFile(path, []byte(`{"name":"Edited","author":"me","themes":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir() second call error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Edited") {
		t.Fatalf("existing theme file was overwritten: %s", data)
	}
}

func TestEnsureThemesDirRefreshesAnUntouchedSeed(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	// An older build wrote a different built-in copy and recorded its hash.
	previous := []byte(`{"name":"Previous","themes":[]}`)
	path := filepath.Join(dir, seedFile)
	if err := os.WriteFile(path, previous, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, seedHashFile), []byte(seedHash(previous)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir() refresh error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(builtinThemeJSON) {
		t.Fatalf("untouched seed was not refreshed: %s", data)
	}
	recorded, err := os.ReadFile(filepath.Join(dir, seedHashFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(recorded) != seedHash(builtinThemeJSON) {
		t.Fatalf("seed hash = %q, want the current palette's", recorded)
	}
}

func TestPartialThemeFallsBackToBuiltin(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"name": "Mine",
		"author": "me",
		"themes": [{
			"name": "Partial",
			"appearance": "light",
			"style": {
				"background": "#fafafa",
				"text": "#101010"
			}
		}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "mine.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, problems := LoadEntries(dir)
	if len(problems) != 0 {
		t.Fatalf("LoadEntries() problems = %q", problems)
	}
	entry, ok := FindVariant(entries, "Partial")
	if !ok {
		t.Fatal("Partial theme was not loaded")
	}
	palette := PaletteFromEntry(entry)
	if palette.UI["bg"] != "#fafafa" {
		t.Errorf("bg = %q, want #fafafa", palette.UI["bg"])
	}
	if palette.UI["fg"] != "#101010" {
		t.Errorf("fg = %q, want #101010", palette.UI["fg"])
	}
	// Missing keys come from the built-in light theme, not from an empty string.
	if palette.UI["bg-raised"] != "#f3f3f3" {
		t.Errorf("bg-raised = %q, want built-in light fallback", palette.UI["bg-raised"])
	}
	if palette.UI["editor-bg"] != "#ffffff" {
		t.Errorf("editor-bg = %q, want built-in light editor fallback", palette.UI["editor-bg"])
	}
	if palette.Syntax["comment"].Color != "#008000" {
		t.Errorf("comment = %q, want built-in light fallback", palette.Syntax["comment"].Color)
	}
}

func TestUserThemeOverridesBuiltinName(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"name": "Mine",
		"author": "me",
		"themes": [{
			"name": "` + config.DefaultThemeDark + `",
			"appearance": "dark",
			"style": {"background": "#000001"}
		}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "override.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, _ := LoadEntries(dir)
	entry, ok := FindVariant(entries, config.DefaultThemeDark)
	if !ok {
		t.Fatalf("%s was not loaded", config.DefaultThemeDark)
	}
	if entry.Variant.Style.Background != "#000001" {
		t.Fatalf("background = %q, want the user override", entry.Variant.Style.Background)
	}
	palette := PaletteFromEntry(entry)
	if palette.UI["bg"] != "#000001" {
		t.Errorf("bg = %q, want #000001", palette.UI["bg"])
	}
}

func TestBrokenThemeFileIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`{"themes":`), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, problems := LoadEntries(dir)
	if len(problems) != 1 || !strings.Contains(problems[0], "broken.json") {
		t.Fatalf("problems = %q, want a broken.json error", problems)
	}
	if _, ok := FindVariant(entries, config.DefaultThemeDark); !ok {
		t.Fatal("built-in themes disappeared after one broken file")
	}
}

func TestFirstColorRejectsMalformedValues(t *testing.T) {
	if got := firstColor("#abc", "#123456"); got != "#abc" {
		t.Fatalf("firstColor() = %q, want #abc", got)
	}
	if got := firstColor("not-a-color", "#123456"); got != "#123456" {
		t.Fatalf("firstColor() = %q, want fallback", got)
	}
	if got := firstColor("#gggggg", ""); got != "" {
		t.Fatalf("firstColor() = %q, want empty fallback", got)
	}
}
