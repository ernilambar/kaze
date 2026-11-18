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
