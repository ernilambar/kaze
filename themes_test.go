package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinThemeFamilyParses(t *testing.T) {
	family, err := builtinThemes()
	if err != nil {
		t.Fatalf("builtinThemes() error = %v", err)
	}
	if family.Name != appMeta.ProductName {
		t.Fatalf("family name = %q, want %q", family.Name, appMeta.ProductName)
	}
	names := map[string]string{}
	for _, variant := range family.Themes {
		names[variant.Name] = variant.Appearance
	}
	if names[defaultThemeDark] != "dark" {
		t.Errorf("%s appearance = %q, want dark", defaultThemeDark, names[defaultThemeDark])
	}
	if names[defaultThemeLight] != "light" {
		t.Errorf("%s appearance = %q, want light", defaultThemeLight, names[defaultThemeLight])
	}
}

func TestEnsureThemesDirSeedsOnce(t *testing.T) {
	dir := t.TempDir()
	if err := ensureThemesDir(dir); err != nil {
		t.Fatalf("ensureThemesDir() error = %v", err)
	}
	path := filepath.Join(dir, "kaze.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("built-in theme was not written: %v", err)
	}

	if err := os.WriteFile(path, []byte(`{"name":"Edited","author":"me","themes":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureThemesDir(dir); err != nil {
		t.Fatalf("ensureThemesDir() second call error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Edited") {
		t.Fatalf("existing theme file was overwritten: %s", data)
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

	entries, problems := loadThemeEntries(dir)
	if len(problems) != 0 {
		t.Fatalf("loadThemeEntries() problems = %q", problems)
	}
	entry, ok := findVariant(entries, "Partial")
	if !ok {
		t.Fatal("Partial theme was not loaded")
	}
	palette := paletteFromEntry(entry)
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
			"name": "` + defaultThemeDark + `",
			"appearance": "dark",
			"style": {"background": "#000001"}
		}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "override.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, _ := loadThemeEntries(dir)
	entry, ok := findVariant(entries, defaultThemeDark)
	if !ok {
		t.Fatalf("%s was not loaded", defaultThemeDark)
	}
	if entry.Variant.Style.Background != "#000001" {
		t.Fatalf("background = %q, want the user override", entry.Variant.Style.Background)
	}
	palette := paletteFromEntry(entry)
	if palette.UI["bg"] != "#000001" {
		t.Errorf("bg = %q, want #000001", palette.UI["bg"])
	}
}

func TestBrokenThemeFileIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`{"themes":`), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, problems := loadThemeEntries(dir)
	if len(problems) != 1 || !strings.Contains(problems[0], "broken.json") {
		t.Fatalf("problems = %q, want a broken.json error", problems)
	}
	if _, ok := findVariant(entries, defaultThemeDark); !ok {
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

func TestLoadThemeAgreesWithCatalog(t *testing.T) {
	// Use a temp config root so the test never touches the real user folder.
	root := t.TempDir()
	t.Setenv("HOME", root)

	catalog := (&App{}).ListThemes()
	if len(catalog.Errors) != 0 {
		t.Fatalf("ListThemes() errors = %q", catalog.Errors)
	}
	if len(catalog.Themes) < 2 {
		t.Fatalf("ListThemes() returned %d themes, want at least the built-in pair", len(catalog.Themes))
	}
	palette, err := (&App{}).LoadTheme(defaultThemeDark)
	if err != nil {
		t.Fatalf("LoadTheme() error = %v", err)
	}
	if palette.Appearance != "dark" || palette.UI["bg"] != "#1e1e1e" {
		t.Fatalf("LoadTheme() = %+v, want %s", palette, defaultThemeDark)
	}
	if _, err := (&App{}).LoadTheme("Missing Theme"); err == nil {
		t.Fatal("LoadTheme() accepted a missing theme")
	}
}
