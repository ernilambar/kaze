package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// revealFile shows path in Finder with the file selected.
func revealFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("no file to reveal")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("reveal %s: %w", filepath.Base(path), err)
	}

	argv := revealArgs(path, info.IsDir())
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch Finder: %w", err)
	}
	return cmd.Process.Release()
}

// revealArgs builds the Finder command that reveals path. It is separate from
// revealFile so the shape is testable without launching anything.
func revealArgs(path string, isDir bool) []string {
	if isDir {
		return []string{"open", path}
	}
	return []string{"open", "-R", path}
}
