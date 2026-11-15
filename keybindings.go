package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Keybinding is one entry in keybindings.json, using VS Code's shape. Command
// is a command id; a leading "-" removes that command's default shortcut.
type Keybinding struct {
	Key     string `json:"key"`
	Command string `json:"command"`
	When    string `json:"when,omitempty"`
}

// KeybindingsState is what the frontend receives: the accepted bindings plus a
// human-readable list of entries that were rejected.
type KeybindingsState struct {
	Bindings []Keybinding `json:"bindings"`
	Error    string       `json:"error"`
	// Parsed says the file decoded; entries may still have been rejected. The
	// frontend installs the accepted list when it is true, even if every entry
	// was rejected, and keeps the previous list when a broken file made the
	// whole decode fail.
	Parsed bool `json:"parsed"`
}

// defaultKeybindingsJSON is the file created on first use. The examples stay
// commented out so the native menu keeps its defaults until the user opts in.
const defaultKeybindingsJSON = `[
    // Keybindings use VS Code's shape: { "key": "...", "command": "..." }.
    // Commands are the ids the command palette runs.
    // "when" supports editorFocus, sidebarVisible, hasActiveTab, hasDirtyTab
    // and the !, && and || operators.
    //
    // { "key": "cmd+shift+p", "command": "command-palette" },
    // { "key": "cmd+k cmd+t", "command": "toggle-preview", "when": "editorFocus" }
]
`

// keybindingsFilePath is the per-user keybindings file next to settings.json.
func keybindingsFilePath() (string, error) {
	path, err := settingsFilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "keybindings.json"), nil
}

// ensureKeybindingsFile creates the file with its commented template when it is
// missing. An existing file is never rewritten.
func ensureKeybindingsFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat keybindings file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create keybindings directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(defaultKeybindingsJSON), 0o644); err != nil {
		return fmt.Errorf("write keybindings file: %w", err)
	}
	return nil
}

// readKeybindings loads the file, creating the default template when absent.
// Bad entries are dropped and reported instead of making every binding fail.
func readKeybindings(path string) ([]Keybinding, []string, error) {
	if err := ensureKeybindingsFile(path); err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read keybindings file: %w", err)
	}
	return parseKeybindings(data)
}

func parseKeybindings(data []byte) ([]Keybinding, []string, error) {
	var raw []Keybinding
	if err := json.Unmarshal(stripJSONComments(data), &raw); err != nil {
		return nil, nil, fmt.Errorf("parse keybindings: %w", err)
	}

	bindings := make([]Keybinding, 0, len(raw))
	problems := make([]string, 0)
	for index, binding := range raw {
		key := strings.TrimSpace(binding.Key)
		command := strings.TrimSpace(binding.Command)
		if key == "" {
			problems = append(problems, fmt.Sprintf("entry %d: no key", index+1))
			continue
		}
		if command == "" || command == "-" {
			problems = append(problems, fmt.Sprintf("entry %d: no command", index+1))
			continue
		}
		if !validKeySequence(key) {
			problems = append(problems, fmt.Sprintf("entry %d: invalid key %q", index+1, key))
			continue
		}
		bindings = append(bindings, Keybinding{
			Key:     normalizeKeySequence(key),
			Command: command,
			When:    strings.TrimSpace(binding.When),
		})
	}
	return bindings, problems, nil
}

// stripJSONComments blanks // and /* */ comments while leaving string contents
// alone. VS Code's keybindings.json is JSONC, so users will paste comments.
func stripJSONComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(data) && data[i+1] == '/' {
			i += 2
			for i < len(data) && data[i] != '\n' {
				out = append(out, ' ')
				i++
			}
			if i < len(data) {
				out = append(out, '\n')
			}
			continue
		}
		if c == '/' && i+1 < len(data) && data[i+1] == '*' {
			i += 2
			for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
				if data[i] == '\n' {
					out = append(out, '\n')
				} else {
					out = append(out, ' ')
				}
				i++
			}
			i++
			continue
		}
		out = append(out, c)
	}
	return out
}

// normalizeKeySequence makes two spellings of the same shortcut compare equal.
// A sequence keeps its chords separated by one space.
func normalizeKeySequence(key string) string {
	chords := strings.Fields(key)
	normalized := make([]string, 0, len(chords))
	for _, chord := range chords {
		normalized = append(normalized, normalizeChord(chord))
	}
	return strings.Join(normalized, " ")
}

func firstChord(key string) string {
	chords := strings.Fields(key)
	if len(chords) == 0 {
		return ""
	}
	return chords[0]
}

func normalizeChord(chord string) string {
	parts := strings.Split(strings.TrimSpace(chord), "+")
	modifiers := make([]string, 0, len(parts))
	key := ""
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "":
			continue
		case "cmd", "command", "cmdorctrl":
			modifiers = append(modifiers, "cmd")
		case "ctrl", "control":
			modifiers = append(modifiers, "ctrl")
		case "alt", "option", "optionoralt":
			modifiers = append(modifiers, "alt")
		case "shift":
			modifiers = append(modifiers, "shift")
		case "super", "win", "meta":
			modifiers = append(modifiers, "meta")
		default:
			if key != "" {
				return ""
			}
			key = normalizeKeyName(part)
		}
	}
	if key == "" {
		return ""
	}
	sort.Strings(modifiers)
	return strings.Join(append(modifiers, key), "+")
}

func normalizeKeyName(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "esc":
		return "escape"
	case "return":
		return "enter"
	case "arrowup":
		return "up"
	case "arrowdown":
		return "down"
	case "arrowleft":
		return "left"
	case "arrowright":
		return "right"
	case "del":
		return "delete"
	case "spacebar":
		return "space"
	default:
		return key
	}
}

func validKeySequence(key string) bool {
	chords := strings.Fields(key)
	if len(chords) == 0 {
		return false
	}
	for _, chord := range chords {
		normalized := normalizeChord(chord)
		if normalized == "" {
			return false
		}
		// A plain letter would swallow typing. Require a modifier unless the
		// key never produces text on its own.
		parts := strings.Split(normalized, "+")
		if len(parts) == 1 && !nonTypingKey(parts[0]) {
			return false
		}
	}
	return true
}

func nonTypingKey(key string) bool {
	switch key {
	case "escape", "enter", "tab", "backspace", "delete",
		"up", "down", "left", "right", "home", "end", "pageup", "pagedown":
		return true
	}
	if len(key) < 2 || key[0] != 'f' {
		return false
	}
	number, err := strconv.Atoi(key[1:])
	return err == nil && number >= 1 && number <= 24
}

// normalizeAccelerator gives a native menu accelerator the same spelling as a
// keybinding, so the two can be compared for conflicts.
func normalizeAccelerator(accelerator string) string {
	return normalizeChord(accelerator)
}

// menuSuppression says which default menu shortcuts must be omitted because
// keybindings.json claims the command or the key. The whole binding lives on
// the frontend, so the native item must not fire as well.
func menuSuppression(bindings []Keybinding) (map[string]bool, map[string]bool) {
	suppressed := make(map[string]bool, len(bindings))
	claimed := make(map[string]bool, len(bindings))
	for _, binding := range bindings {
		command := strings.TrimPrefix(binding.Command, "-")
		suppressed[command] = true
		if chord := firstChord(binding.Key); chord != "" {
			claimed[normalizeChord(chord)] = true
		}
	}
	return suppressed, claimed
}

// GetKeybindings returns the accepted bindings and any per-entry problems. The
// frontend installs the list and reports the error without losing the valid
// entries.
func (a *App) GetKeybindings() KeybindingsState {
	path, err := keybindingsFilePath()
	if err != nil {
		return KeybindingsState{Bindings: []Keybinding{}, Error: err.Error(), Parsed: false}
	}
	bindings, problems, err := readKeybindings(path)
	if err != nil {
		return KeybindingsState{Bindings: []Keybinding{}, Error: err.Error(), Parsed: false}
	}
	return KeybindingsState{
		Bindings: bindings,
		Error:    strings.Join(problems, "; "),
		Parsed:   true,
	}
}

// OpenKeybindings opens keybindings.json with the user's editor, creating the
// commented template on first use.
func (a *App) OpenKeybindings() error {
	path, err := keybindingsFilePath()
	if err != nil {
		return err
	}
	if err := ensureKeybindingsFile(path); err != nil {
		return err
	}
	return openInEditor(path)
}

// loadKeybindingsForMenu is the menu's read path. A malformed file leaves the
// native defaults in place; the frontend reports the parse error.
func loadKeybindingsForMenu() []Keybinding {
	path, err := keybindingsFilePath()
	if err != nil {
		return nil
	}
	bindings, _, err := readKeybindings(path)
	if err != nil {
		return nil
	}
	return bindings
}
