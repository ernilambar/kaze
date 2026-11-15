package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// revealFile shows path in the platform's file manager. On macOS that is
// Finder with the file selected; the other platforms get their own equivalent
// so a shared command keeps working outside the bundle.
func revealFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("no file to reveal")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("reveal %s: %w", filepath.Base(path), err)
	}

	argv := revealArgs(path, info.IsDir(), runtime.GOOS)
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch file manager: %w", err)
	}
	return cmd.Process.Release()
}

// revealArgs builds the command that reveals path. It is separate from
// revealFile so the per-platform shapes are testable without launching
// anything.
func revealArgs(path string, isDir bool, goos string) []string {
	switch goos {
	case "windows":
		if isDir {
			return []string{"explorer", path}
		}
		return []string{"explorer", "/select," + path}
	case "darwin":
		if isDir {
			return []string{"open", path}
		}
		return []string{"open", "-R", path}
	default:
		target := path
		if !isDir {
			target = filepath.Dir(path)
		}
		return []string{"xdg-open", target}
	}
}
