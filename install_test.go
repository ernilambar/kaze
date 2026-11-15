package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLauncherScriptQuotesTheBundlePath(t *testing.T) {
	script := launcherScript("/Applications/Editor.app/Contents/MacOS/editor")

	if !strings.Contains(script, "app='/Applications/Editor.app/Contents/MacOS/editor'") {
		t.Fatalf("launcherScript() does not point at the bundle executable:\n%s", script)
	}
	if !strings.HasPrefix(script, "#!/bin/sh\n") {
		t.Fatalf("launcherScript() is not a shell script:\n%s", script)
	}
	if !strings.Contains(script, `exec "$app" "$@"`) {
		t.Fatalf("launcherScript() does not exec the app:\n%s", script)
	}
}

func TestLauncherScriptMakesArgumentsAbsolute(t *testing.T) {
	// Relative paths have to be resolved in the caller's shell: a forwarded
	// launch runs in the running instance's working directory, not the user's.
	script := launcherScript("/Applications/Editor.app/Contents/MacOS/editor")

	if !strings.Contains(script, `*) arg="$PWD/$arg" ;;`) {
		t.Fatalf("launcherScript() does not resolve relative arguments:\n%s", script)
	}
}

func TestShellQuoteEscapesSingleQuotes(t *testing.T) {
	got := shellQuote("it's")

	want := `'it'\''s'`
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}

func TestAppleScriptStringEscapesQuotesAndBackslashes(t *testing.T) {
	got := appleScriptString(`say "hi" \ bye`)

	want := `"say \"hi\" \\ bye"`
	if got != want {
		t.Fatalf("appleScriptString() = %q, want %q", got, want)
	}
}

func TestInstallTargetRejectsUnbundledBinary(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	if strings.Contains(exe, ".app/"+bundleExecutable) {
		t.Skip("test binary runs from a bundle; it cannot exercise the rejection path")
	}

	if _, err := installTarget(); err == nil {
		t.Fatal("installTarget() error = nil, want a bundled-app requirement")
	}
}

func TestWriteFileAtomicReplacesContentAndKeepsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bin", "editor")

	if err := writeFileAtomic(path, []byte("first\n")); err != nil {
		t.Fatalf("writeFileAtomic() error = %v", err)
	}
	if err := writeFileAtomic(path, []byte("second\n")); err != nil {
		t.Fatalf("writeFileAtomic() replace error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read installed file: %v", err)
	}
	if string(data) != "second\n" {
		t.Fatalf("installed content = %q, want %q", data, "second\n")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat installed file: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("installed mode = %v, want 755", info.Mode().Perm())
	}
}

func TestInstallLauncherWritesRunnableScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the launcher is a POSIX shell script")
	}

	path := filepath.Join(t.TempDir(), "bin", "editor")
	target := "/Applications/Editor.app/Contents/MacOS/editor"

	if err := installLauncher(path, target); err != nil {
		t.Fatalf("installLauncher() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat installed launcher: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("installed mode = %v, want 755", info.Mode().Perm())
	}

	// The launcher must survive `sh -n`, since a quoting mistake would only
	// surface for the user as a broken command.
	if out, err := exec.Command("sh", "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("installed launcher is not valid sh: %v\n%s", err, out)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read installed launcher: %v", err)
	}
	if !strings.Contains(string(data), target) {
		t.Fatalf("installed launcher does not reference %q:\n%s", target, data)
	}
}

func TestInstalledLauncherRunsTargetWithAbsolutePaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the launcher is a POSIX shell script")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "Editor.app", "Contents", "MacOS", "editor")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	// The fake app prints each argument it receives, on its own line.
	recorder := "#!/bin/sh\nfor arg in \"$@\"; do printf '%s\\n' \"$arg\"; done\n"
	if err := os.WriteFile(target, []byte(recorder), 0o755); err != nil {
		t.Fatalf("write fake app: %v", err)
	}

	launcher := filepath.Join(dir, "bin", "editor")
	if err := installLauncher(launcher, target); err != nil {
		t.Fatalf("installLauncher() error = %v", err)
	}

	work := filepath.Join(dir, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatalf("create working directory: %v", err)
	}

	cmd := exec.Command(launcher, "notes.txt", "/tmp/absolute.txt", "-v")
	cmd.Dir = work
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run launcher: %v\n%s", err, out)
	}

	// Relative paths are resolved in the caller's directory, absolute paths are
	// left alone, and flags are not files to rewrite.
	want := filepath.Join(work, "notes.txt") + "\n/tmp/absolute.txt\n-v\n"
	if string(out) != want {
		t.Fatalf("launcher output = %q, want %q", out, want)
	}
}

func TestCommandInstallPathUsesInstallDir(t *testing.T) {
	want := installDir + "/" + commandName
	if got := commandInstallPath(); got != want {
		t.Fatalf("commandInstallPath() = %q, want %q", got, want)
	}
}

func TestPrivilegedInstallCommandInstallsBothIntermediates(t *testing.T) {
	got := privilegedInstallCommand("/usr/local/bin/editor", "/tmp/editor-command-1")

	want := "/usr/bin/install -d -m 755 '/usr/local/bin' && /usr/bin/install -m 755 '/tmp/editor-command-1' '/usr/local/bin/editor'"
	if got != want {
		t.Fatalf("privilegedInstallCommand() = %q, want %q", got, want)
	}
}
