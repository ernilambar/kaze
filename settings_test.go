package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadSettingsCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Kova", "settings.json")

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	want := defaultSettings()
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("loadSettings() = %+v, want %+v", settings, want)
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

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	want := Settings{
		Editor: EditorSettings{
			FontFamily: "JetBrains Mono",
			FontSize:   16.5,
			FontWeight: 300,
		},
		Sidebar: SidebarSettings{Folders: []string{}},
	}
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("loadSettings() = %+v, want %+v", settings, want)
	}
}

func TestLoadSettingsFillsMissingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"editor":{"font_size":18}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	if settings.Editor.FontFamily != defaultFontFamily {
		t.Errorf("font_family = %q, want %q", settings.Editor.FontFamily, defaultFontFamily)
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

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	if !reflect.DeepEqual(settings, defaultSettings()) {
		t.Fatalf("loadSettings() = %+v, want defaults %+v", settings, defaultSettings())
	}
}

func TestLoadSettingsMalformedFileKeepsContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"editor":`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err == nil {
		t.Fatal("loadSettings() error = nil, want parse error")
	}
	if !reflect.DeepEqual(settings, defaultSettings()) {
		t.Fatalf("loadSettings() = %+v, want defaults %+v", settings, defaultSettings())
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != content {
		t.Fatalf("malformed file was modified: %q", data)
	}
}

func TestLoadSettingsRewritesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(" \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := loadSettings(path); err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"font_family": "Menlo"`) {
		t.Fatalf("empty settings file was not rewritten with defaults: %q", data)
	}
}

func TestSettingsFilePathUsesConfigDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)

	path, err := settingsFilePath()
	if err != nil {
		t.Fatalf("settingsFilePath() error = %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("Kova", "settings.json")) {
		t.Fatalf("settingsFilePath() = %q, want path under Kova", path)
	}
}

func TestLoadSettingsDefaultsToNoSidebarFolders(t *testing.T) {
	settings, err := loadSettings(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	if settings.Sidebar.Folders == nil {
		t.Fatal("Sidebar.Folders = nil, want an empty slice so the UI receives an array")
	}
	if len(settings.Sidebar.Folders) != 0 {
		t.Fatalf("Sidebar.Folders = %q, want no folders", settings.Sidebar.Folders)
	}
}

func TestLoadSettingsReadsSidebarFolders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"sidebar":{"folders":["/one","/two"]}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	want := []string{"/one", "/two"}
	if !reflect.DeepEqual(settings.Sidebar.Folders, want) {
		t.Fatalf("Sidebar.Folders = %q, want %q", settings.Sidebar.Folders, want)
	}
}

func TestNormalizeFolderListRepairsUserContent(t *testing.T) {
	got := normalizeFolderList([]string{" /one ", "", "/two", "/one", "   "})

	want := []string{"/one", "/two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeFolderList() = %q, want %q", got, want)
	}
}

func TestNormalizeFolderListReturnsEmptySliceForNil(t *testing.T) {
	got := normalizeFolderList(nil)

	if got == nil {
		t.Fatal("normalizeFolderList(nil) = nil, want an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("normalizeFolderList(nil) = %q, want no folders", got)
	}
}
