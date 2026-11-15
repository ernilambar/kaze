package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// openInEditor launches the user's editor on path. $VISUAL is preferred over
// $EDITOR, following the usual Unix convention; with neither set the platform's
// default file handler opens the file. The editor is not waited for, so it
// outlives the app.
func openInEditor(path string) error {
	argv, err := editorArgs(path, os.Getenv("VISUAL"), os.Getenv("EDITOR"), runtime.GOOS)
	if err != nil {
		return err
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch editor %q: %w", argv[0], err)
	}
	return cmd.Process.Release()
}

// editorArgs builds the argv that opens path. visual and editor are the raw
// environment values; they may carry arguments ("code -w"), so they are split
// with shell-like quoting. Empty values count as unset.
func editorArgs(path, visual, editor, goos string) ([]string, error) {
	command := strings.TrimSpace(visual)
	if command == "" {
		command = strings.TrimSpace(editor)
	}
	if command == "" {
		return defaultOpener(path, goos), nil
	}

	argv, err := splitCommand(command)
	if err != nil {
		return nil, err
	}
	return append(argv, path), nil
}

// defaultOpener is the platform's file handler, used when no editor is
// configured in the environment.
func defaultOpener(path, goos string) []string {
	switch goos {
	case "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler", path}
	case "darwin":
		return []string{"open", path}
	default:
		return []string{"xdg-open", path}
	}
}

// splitCommand splits a command line on whitespace, honouring single quotes,
// double quotes and backslash escapes so an editor path containing spaces
// survives intact.
func splitCommand(line string) ([]string, error) {
	runes := []rune(line)
	var (
		argv    []string
		current strings.Builder
		quote   rune
		started bool
	)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			started = true
		case r == '\\' && i+1 < len(runes):
			i++
			current.WriteRune(runes[i])
			started = true
		case r == ' ' || r == '\t':
			if started {
				argv = append(argv, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(r)
			started = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unbalanced quote in editor command %q", line)
	}
	if started {
		argv = append(argv, current.String())
	}
	if len(argv) == 0 {
		return nil, errors.New("empty editor command")
	}
	return argv, nil
}
