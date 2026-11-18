package main

import (
	"testing"

	"kaze/internal/config"
	"kaze/internal/themes"
)

func TestBuiltinThemeFamilyParses(t *testing.T) {
	family, err := themes.Builtin()
	if err != nil {
		t.Fatalf("themes.Builtin() error = %v", err)
	}
	if family.Name != appMeta.ProductName {
		t.Fatalf("family name = %q, want %q", family.Name, appMeta.ProductName)
	}
	names := map[string]string{}
	for _, variant := range family.Themes {
		names[variant.Name] = variant.Appearance
	}
	if names[config.DefaultThemeDark] != "dark" {
		t.Errorf("%s appearance = %q, want dark", config.DefaultThemeDark, names[config.DefaultThemeDark])
	}
	if names[config.DefaultThemeLight] != "light" {
		t.Errorf("%s appearance = %q, want light", config.DefaultThemeLight, names[config.DefaultThemeLight])
	}
}

func TestCatalogPreviewMatchesPalette(t *testing.T) {
	// Use a temp config root so a user theme cannot change what is listed.
	root := t.TempDir()
	t.Setenv("HOME", root)

	catalog := (&App{}).ListThemes()
	for _, themeName := range []string{config.DefaultThemeDark, config.DefaultThemeLight} {
		var summary *themes.ThemeSummary
		for index := range catalog.Themes {
			if catalog.Themes[index].Name == themeName {
				summary = &catalog.Themes[index]
				break
			}
		}
		if summary == nil {
			t.Fatalf("ListThemes() did not include %q", themeName)
		}
		palette, err := (&App{}).LoadTheme(themeName)
		if err != nil {
			t.Fatalf("LoadTheme(%q) error = %v", themeName, err)
		}
		if summary.Preview.Background != palette.Editor["editor.background"] {
			t.Errorf("%s preview background = %q, want %q", themeName, summary.Preview.Background, palette.Editor["editor.background"])
		}
		if summary.Preview.Foreground != palette.Editor["editor.foreground"] {
			t.Errorf("%s preview foreground = %q, want %q", themeName, summary.Preview.Foreground, palette.Editor["editor.foreground"])
		}
		if summary.Preview.Accent != palette.UI["link"] {
			t.Errorf("%s preview accent = %q, want %q", themeName, summary.Preview.Accent, palette.UI["link"])
		}
		for capture, got := range map[string]string{
			"keyword": summary.Preview.Keyword,
			"string":  summary.Preview.String,
			"comment": summary.Preview.Comment,
		} {
			if expected := palette.Syntax[capture].Color; got != expected {
				t.Errorf("%s preview %s = %q, want %q", themeName, capture, got, expected)
			}
		}
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
	palette, err := (&App{}).LoadTheme(config.DefaultThemeDark)
	if err != nil {
		t.Fatalf("LoadTheme() error = %v", err)
	}
	if palette.Appearance != "dark" || palette.UI["bg"] != "#1e1e1e" {
		t.Fatalf("LoadTheme() = %+v, want %s", palette, config.DefaultThemeDark)
	}
	if _, err := (&App{}).LoadTheme("Missing Theme"); err == nil {
		t.Fatal("LoadTheme() accepted a missing theme")
	}
}
