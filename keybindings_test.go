package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStripJSONCommentsKeepsStrings(t *testing.T) {
	input := `[
		// one line
		{ "key": "cmd+b", "command": "toggle-sidebar" }, /* block
		comment */
		{ "key": "cmd+shift+p", "command": "settings", "when": "editorFocus" },
		{ "key": "cmd+l", "command": "https://example.com" }
	]`
	bindings, problems, err := parseKeybindings([]byte(input))
	if err != nil {
		t.Fatalf("parseKeybindings() error = %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("parseKeybindings() problems = %q", problems)
	}
	if len(bindings) != 3 {
		t.Fatalf("parseKeybindings() = %d bindings, want 3", len(bindings))
	}
	if bindings[1].When != "editorFocus" {
		t.Errorf("when = %q, want editorFocus", bindings[1].When)
	}
	if bindings[2].Command != "https://example.com" {
		t.Errorf("command = %q, want the URL intact", bindings[2].Command)
	}
}

func TestParseKeybindingsRejectsPlainTypingKeys(t *testing.T) {
	input := `[{"key":"b","command":"toggle-sidebar"}]`
	bindings, problems, err := parseKeybindings([]byte(input))
	if err != nil {
		t.Fatalf("parseKeybindings() error = %v", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("plain key was accepted: %+v", bindings)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "invalid key") {
		t.Fatalf("problems = %q, want an invalid key error", problems)
	}
}

func TestParseKeybindingsAllowsNonTypingKeys(t *testing.T) {
	input := `[{"key":"escape","command":"settings"},{"key":"f5","command":"refresh"}]`
	bindings, problems, err := parseKeybindings([]byte(input))
	if err != nil {
		t.Fatalf("parseKeybindings() error = %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %q", problems)
	}
	if len(bindings) != 2 {
		t.Fatalf("bindings = %+v, want two", bindings)
	}
}

func TestParseKeybindingsMalformedJSON(t *testing.T) {
	if _, _, err := parseKeybindings([]byte(`[{"key":`)); err == nil {
		t.Fatal("parseKeybindings() accepted malformed JSON")
	}
}

func TestNormalizeKeySequence(t *testing.T) {
	cases := map[string]string{
		"CmdOrCtrl+Shift+P": "cmd+shift+p",
		"cmd+shift+p":       "cmd+shift+p",
		"Ctrl+Tab":          "ctrl+tab",
		"ctrl+shift+tab":    "ctrl+shift+tab",
		"option+cmd+k":      "alt+cmd+k",
		"cmd+k cmd+t":       "cmd+k cmd+t",
		"esc":               "escape",
	}
	for input, want := range cases {
		if got := normalizeKeySequence(input); got != want {
			t.Errorf("normalizeKeySequence(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMenuSuppressionUsesCommandAndFirstChord(t *testing.T) {
	bindings := []Keybinding{
		{Key: "cmd+b", Command: "toggle-sidebar"},
		{Key: "cmd+shift+p", Command: "command-palette"},
		{Key: "cmd+k cmd+t", Command: "toggle-preview"},
		{Key: "cmd+s", Command: "-save"},
	}
	suppressed, claimed := menuSuppression(bindings)
	for _, command := range []string{"toggle-sidebar", "command-palette", "toggle-preview", "save"} {
		if !suppressed[command] {
			t.Errorf("command %q was not suppressed", command)
		}
	}
	for _, key := range []string{"cmd+b", "cmd+shift+p", "cmd+k", "cmd+s"} {
		if !claimed[key] {
			t.Errorf("key %q was not claimed", key)
		}
	}
}

func TestEnsureKeybindingsFileWritesTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keybindings.json")
	if err := ensureKeybindingsFile(path); err != nil {
		t.Fatalf("ensureKeybindingsFile() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Kova keybindings") {
		t.Fatalf("template = %q, want the explanatory comment", data)
	}
	// The template is JSONC and must parse to an empty list.
	bindings, problems, err := parseKeybindings(data)
	if err != nil {
		t.Fatalf("template does not parse: %v", err)
	}
	if len(bindings) != 0 || len(problems) != 0 {
		t.Fatalf("template = %+v problems = %q, want empty", bindings, problems)
	}
}

func TestReadKeybindingsCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keybindings.json")
	bindings, problems, err := readKeybindings(path)
	if err != nil {
		t.Fatalf("readKeybindings() error = %v", err)
	}
	if !reflect.DeepEqual(bindings, []Keybinding{}) {
		t.Fatalf("bindings = %+v, want empty", bindings)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %q", problems)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("keybindings file was not created: %v", err)
	}
}
