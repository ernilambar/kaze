package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	commandName = "kova"
	// installDir is on the default macOS PATH, so installing here makes `kova`
	// work in a fresh shell without touching the user's profile.
	installDir = "/usr/local/bin"
	// bundleExecutable is the Mach-O inside the .app bundle. Running it
	// directly keeps the caller's working directory, which is what makes
	// relative arguments like `kova notes.txt` resolve.
	bundleExecutable = "Contents/MacOS/" + commandName
)

// commandInstallPath is the file the menu item writes.
func commandInstallPath() string {
	return filepath.Join(installDir, commandName)
}

// launcherScript returns the shell script installed as `kova`. Arguments are
// made absolute before Kova sees them: when an instance is already running the
// launch is forwarded to it, and that instance resolves relative paths against
// its own working directory rather than the shell's.
func launcherScript(binaryPath string) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("# Installed by Kova's \"Install Command Line Tool…\" menu item. Re-run\n")
	b.WriteString("# that item after moving Kova.app; the path below is baked in.\n")
	b.WriteString("app=" + shellQuote(binaryPath) + "\n")
	b.WriteString("count=$#\n")
	b.WriteString("while [ \"$count\" -gt 0 ]; do\n")
	b.WriteString("    arg=\"$1\"\n")
	b.WriteString("    shift\n")
	b.WriteString("    case \"$arg\" in\n")
	b.WriteString("        -*) ;;\n")
	b.WriteString("        /*) ;;\n")
	b.WriteString("        *) arg=\"$PWD/$arg\" ;;\n")
	b.WriteString("    esac\n")
	b.WriteString("    set -- \"$@\" \"$arg\"\n")
	b.WriteString("    count=$((count - 1))\n")
	b.WriteString("done\n")
	b.WriteString("exec \"$app\" \"$@\"\n")
	return b.String()
}

// shellQuote wraps s in single quotes for POSIX sh.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// installTarget returns the executable the launcher should run. Kova must be a
// bundled app: a bare `go run` binary has no stable path to point at.
func installTarget() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve running executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve running executable: %w", err)
	}
	if !strings.Contains(exe, ".app/"+bundleExecutable) {
		return "", errors.New("the command line tool can only be installed from the Kova.app bundle")
	}
	return exe, nil
}

// installCommandLineTool writes the `kova` launcher. A writable install
// directory is used directly; otherwise macOS asks for an administrator
// password, which is the normal case for /usr/local/bin on a fresh machine.
func installCommandLineTool() error {
	target, err := installTarget()
	if err != nil {
		return err
	}
	return installLauncher(commandInstallPath(), target)
}

// installLauncher is installCommandLineTool with the install path and target
// spelled out, so the writable and privileged paths are both testable.
func installLauncher(path, target string) error {
	script := []byte(launcherScript(target))
	if err := writeFileAtomic(path, script); err == nil {
		return nil
	}
	return writeFileWithPrivileges(path, script)
}

// writeFileAtomic writes data to path through a temporary file, so an existing
// launcher is replaced in one step instead of being truncated.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), commandName+"-")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return fmt.Errorf("make %s executable: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}

// writeFileWithPrivileges installs through `do shell script … with
// administrator privileges`, which is macOS' own password prompt.
func writeFileWithPrivileges(path string, data []byte) error {
	tmp, err := os.CreateTemp("", "kova-command-")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", tmpName, err)
	}

	command := privilegedInstallCommand(path, tmpName)
	script := fmt.Sprintf(
		"do shell script %s with administrator privileges",
		appleScriptString(command),
	)

	out, err := exec.Command("/usr/bin/osascript", "-e", script).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		// Cancelling the password prompt is a decision, not a failure to report
		// as a bug.
		if strings.Contains(message, "User canceled") || strings.Contains(message, "(-128)") {
			return errors.New("installation cancelled")
		}
		if message == "" {
			return fmt.Errorf("install %s: %w", path, err)
		}
		return fmt.Errorf("install %s: %s", path, message)
	}
	return nil
}

// privilegedInstallCommand is the shell command run as root: it creates the
// install directory and moves the staged copy into place with mode 755.
func privilegedInstallCommand(path, staged string) string {
	return fmt.Sprintf(
		"/usr/bin/install -d -m 755 %s && /usr/bin/install -m 755 %s %s",
		shellQuote(filepath.Dir(path)),
		shellQuote(staged),
		shellQuote(path),
	)
}

// appleScriptString renders s as an AppleScript string literal.
func appleScriptString(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(s) + `"`
}
