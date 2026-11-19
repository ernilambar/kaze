package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitignoreRulesHideMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	rules := parseIgnoreRules(dir, "*.log\nbuild/\n/secret.txt\n!keep.log\n")

	if !Ignored(rules, filepath.Join(dir, "debug.log"), false) {
		t.Error("debug.log was not ignored")
	}
	if Ignored(rules, filepath.Join(dir, "keep.log"), false) {
		t.Error("keep.log was ignored despite the negation")
	}
	if !Ignored(rules, filepath.Join(dir, "build"), true) {
		t.Error("build/ was not ignored")
	}
	if Ignored(rules, filepath.Join(dir, "build"), false) {
		t.Error("directory-only pattern matched a file")
	}
	if !Ignored(rules, filepath.Join(dir, "secret.txt"), false) {
		t.Error("anchored pattern did not match at the root")
	}
	if Ignored(rules, filepath.Join(dir, "src", "secret.txt"), false) {
		t.Error("anchored pattern matched below its directory")
	}
}

func TestGitignoreRulesInChildDirectoryOnlyApplyBelowIt(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}

	rules := append(parseIgnoreRules(dir, ""), parseIgnoreRules(child, "*.gen.go")...)
	if !Ignored(rules, filepath.Join(child, "model.gen.go"), false) {
		t.Error("child rule did not match its own directory")
	}
	if Ignored(rules, filepath.Join(dir, "model.gen.go"), false) {
		t.Error("child rule matched the parent directory")
	}
}

func TestParseIgnoreRulesSkipsCommentsAndBlanks(t *testing.T) {
	rules := parseIgnoreRules(t.TempDir(), "# comment\n\n   \n*.tmp\n")
	if len(rules) != 1 {
		t.Fatalf("parsed %d rules, want 1", len(rules))
	}
	if rules[0].pattern.segments[0] != "*.tmp" {
		t.Fatalf("rule = %+v, want *.tmp", rules[0])
	}
}

func TestRepoForReadsRulesFromTheRepositoryRootDown(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, root)
	if err := os.MkdirAll(filepath.Join(root, "pkg", "ui"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".gitignore", "*.log\n")
	writeFile(t, filepath.Join(root, "pkg"), ".gitignore", "build/\n")

	repo := RepoFor(filepath.Join(root, "pkg", "ui"))
	if repo.Root != root {
		t.Fatalf("RepoFor() root = %q, want %q", repo.Root, root)
	}
	if !repo.Ignored(filepath.Join(root, "pkg", "ui", "debug.log"), false) {
		t.Error("a root rule did not reach a nested folder")
	}
	if !repo.Ignored(filepath.Join(root, "pkg", "build"), true) {
		t.Error("a nested rule was not read")
	}
	if repo.Ignored(filepath.Join(root, "pkg", "main.go"), false) {
		t.Error("main.go must not be ignored")
	}
}

func TestRepoForWithoutARepositoryIgnoresNothing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.log\n")

	repo := RepoFor(dir)
	if repo.Ignored(filepath.Join(dir, "debug.log"), false) {
		t.Error("a folder outside a repository must ignore nothing")
	}
}

func TestRepoIgnoredIncludesPathsBelowAnIgnoredDirectory(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, root)
	writeFile(t, root, ".gitignore", "dist/\n!dist/keep.txt\n")

	repo := RepoFor(root)
	if !repo.Ignored(filepath.Join(root, "dist", "bundle.js"), false) {
		t.Error("a file below an ignored folder must be ignored")
	}
	if !repo.Ignored(filepath.Join(root, "dist", "keep.txt"), false) {
		t.Error("git cannot re-include a file below an excluded folder")
	}
}

func makeRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
