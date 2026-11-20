package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadSettingsCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "settings.json")

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Default()
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("Load() = %+v, want %+v", settings, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings file was not created: %v", err)
	}

	var written map[string]map[string]any
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("settings file is not valid JSON: %v", err)
	}
	editor := written["editor"]
	if editor["font_family"] != "Menlo" {
		t.Errorf("font_family = %v, want Menlo", editor["font_family"])
	}
	if editor["font_size"] != 13.0 {
		t.Errorf("font_size = %v, want 13", editor["font_size"])
	}
	if editor["font_weight"] != 400.0 {
		t.Errorf("font_weight = %v, want 400", editor["font_weight"])
	}
}

func TestLoadSettingsReadsUserValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{
		"editor": {
			"font_family": "JetBrains Mono",
			"font_size": 16.5,
			"font_weight": 300
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Default()
	want.Editor.FontFamily = "JetBrains Mono"
	want.Editor.FontSize = 16.5
	want.Editor.FontWeight = 300
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("Load() = %+v, want %+v", settings, want)
	}
}

func TestLoadSettingsFillsMissingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"editor":{"font_size":18}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if settings.Editor.FontFamily != DefaultFontFamily {
		t.Errorf("font_family = %q, want %q", settings.Editor.FontFamily, DefaultFontFamily)
	}
	if settings.Editor.FontSize != 18 {
		t.Errorf("font_size = %v, want 18", settings.Editor.FontSize)
	}
	if settings.Editor.FontWeight != defaultFontWeight {
		t.Errorf("font_weight = %v, want %v", settings.Editor.FontWeight, defaultFontWeight)
	}
}

func TestLoadSettingsRejectsInvalidValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"editor":{"font_family":"  ","font_size":-4,"font_weight":0}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !reflect.DeepEqual(settings, Default()) {
		t.Fatalf("Load() = %+v, want defaults %+v", settings, Default())
	}
}

func TestLoadSettingsMalformedFileKeepsContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"editor":`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want parse error")
	}
	if !reflect.DeepEqual(settings, Default()) {
		t.Fatalf("Load() = %+v, want defaults %+v", settings, Default())
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != content {
		t.Fatalf("malformed file was modified: %q", data)
	}
}

func TestLoadSettingsReadsEmptyFileAsDefaultsWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(" \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.Editor.FontFamily != Default().Editor.FontFamily {
		t.Fatalf("font family = %q, want the default", settings.Editor.FontFamily)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != " \n" {
		t.Fatalf("empty settings file was rewritten: %q", data)
	}
}

func TestLoadSettingsDefaultsToAutoReveal(t *testing.T) {
	settings, err := Load(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !settings.Sidebar.AutoReveal {
		t.Fatal("Sidebar.AutoReveal = false, want true by default")
	}
}

func TestLoadSettingsReadsAutoReveal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"sidebar":{"auto_reveal":false}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.Sidebar.AutoReveal {
		t.Fatal("Sidebar.AutoReveal = true, want the saved false")
	}
}

func TestNormalizeFolderListRepairsUserContent(t *testing.T) {
	got := NormalizeFolderList([]string{" /one ", "", "/two", "/one", "   "})

	want := []string{"/one", "/two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeFolderList() = %q, want %q", got, want)
	}
}

func TestNormalizeFolderListReturnsEmptySliceForNil(t *testing.T) {
	got := NormalizeFolderList(nil)

	if got == nil {
		t.Fatal("NormalizeFolderList(nil) = nil, want an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("NormalizeFolderList(nil) = %q, want no folders", got)
	}
}

func TestLoadSettingsKeepsViewOptionDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"editor":{"font_size":18}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Absent keys keep the defaults the editor was designed around, so an old
	// settings file does not silently turn off word wrap or line numbers.
	defaults := Default().Editor
	if !settings.Editor.WordWrap || !settings.Editor.LineNumbers ||
		!settings.Editor.IndentGuides || !settings.Editor.CodeFolding ||
		!settings.Editor.AutoCloseBrackets || !settings.Editor.SmartIndent {
		t.Errorf("missing view options did not keep their defaults: %+v", settings.Editor)
	}
	if settings.Editor.CursorBlink != defaults.CursorBlink {
		t.Errorf("cursor_blink = %v, want %v", settings.Editor.CursorBlink, defaults.CursorBlink)
	}
}

func TestLoadSettingsReadsEditorOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"editor":{
		"tab_size":2,
		"hard_tabs":true,
		"ligatures":true,
		"cursor_width":3,
		"cursor_blink":false,
		"cursor_surrounding_lines":2,
		"scroll_beyond_last_line":true,
		"ensure_final_newline":true,
		"word_wrap":false,
		"invisible_characters":true,
		"line_numbers":false,
		"indent_guides":false,
		"code_folding":false,
		"auto_close_brackets":false,
		"smart_indent":false
	}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	editor := settings.Editor
	if editor.TabSize != 2 || !editor.HardTabs || !editor.Ligatures {
		t.Errorf("indentation settings = %+v", editor)
	}
	if editor.CursorWidth != 3 || editor.CursorBlink || editor.CursorSurroundingLines != 2 {
		t.Errorf("cursor settings = %+v", editor)
	}
	if editor.ScrollBeyondLastLine != ScrollBeyondOnePage || !editor.EnsureFinalNewline {
		t.Errorf("save/scroll settings = %+v", editor)
	}
	if editor.WordWrap || !editor.InvisibleCharacters || editor.LineNumbers ||
		editor.IndentGuides || editor.CodeFolding || editor.AutoCloseBrackets || editor.SmartIndent {
		t.Errorf("view settings = %+v", editor)
	}
}

func TestLoadSettingsReadsScrollBeyondModes(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  ScrollBeyondMode
	}{
		{name: "off", value: `"off"`, want: ScrollBeyondOff},
		{name: "half page", value: `"half_page"`, want: ScrollBeyondHalfPage},
		{name: "one_page", value: `"one_page"`, want: ScrollBeyondOnePage},
		{name: "legacy true", value: `true`, want: ScrollBeyondOnePage},
		{name: "legacy false", value: `false`, want: ScrollBeyondOff},
		{name: "unknown", value: `"page"`, want: ScrollBeyondOnePage},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			content := `{"editor":{"scroll_beyond_last_line":` + test.value + `}}`
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			settings, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if settings.Editor.ScrollBeyondLastLine != test.want {
				t.Errorf(
					"scroll_beyond_last_line = %q, want %q",
					settings.Editor.ScrollBeyondLastLine,
					test.want,
				)
			}
		})
	}
}

func TestNormalizeRulers(t *testing.T) {
	settings := Default()
	settings.Editor.Rulers = []int{120, 0, 80, 120, -4, 100}
	settings.Normalize()

	if !reflect.DeepEqual(settings.Editor.Rulers, []int{80, 100, 120}) {
		t.Errorf("rulers = %v, want [80 100 120]", settings.Editor.Rulers)
	}
}

func TestNormalizeThemeAndUI(t *testing.T) {
	settings := Default()
	settings.Theme = ThemeSettings{Mode: "LIGHT", Light: "  ", Dark: "  Custom Dark  "}
	settings.UI = UISettings{
		FontFamily:      "  Fira Sans  ",
		FontSize:        0,
		TabIcons:        false,
		IconStyle:       "neon",
		SidebarPosition: "RIGHT",
	}
	settings.Normalize()

	if settings.Theme.Mode != "light" {
		t.Errorf("theme mode = %q, want light", settings.Theme.Mode)
	}
	if settings.Theme.Light != DefaultThemeLight {
		t.Errorf("light theme = %q, want %q", settings.Theme.Light, DefaultThemeLight)
	}
	if settings.Theme.Dark != "Custom Dark" {
		t.Errorf("dark theme = %q, want Custom Dark", settings.Theme.Dark)
	}
	if settings.UI.FontFamily != "Fira Sans" {
		t.Errorf("ui font = %q, want Fira Sans", settings.UI.FontFamily)
	}
	if settings.UI.FontSize != defaultUIFontSize {
		t.Errorf("ui font size = %v, want %v", settings.UI.FontSize, defaultUIFontSize)
	}
	if settings.UI.TabIcons {
		t.Error("tab icons = true, want the explicit false")
	}
	if settings.UI.IconStyle != defaultUIIconStyle {
		t.Errorf("icon style = %q, want %q", settings.UI.IconStyle, defaultUIIconStyle)
	}
	if settings.UI.SidebarPosition != "right" {
		t.Errorf("sidebar position = %q, want right", settings.UI.SidebarPosition)
	}
}

func TestNormalizeThemeModeKeepsSystem(t *testing.T) {
	if got := NormalizeThemeMode(" System "); got != "system" {
		t.Errorf("theme mode = %q, want system", got)
	}
	if got := NormalizeThemeMode("neon"); got != defaultThemeMode {
		t.Errorf("theme mode = %q, want %q", got, defaultThemeMode)
	}
}
