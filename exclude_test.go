package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExcludeMatcherHidesNamesAtAnyDepth(t *testing.T) {
	matcher := newExcludeMatcher([]string{"node_modules", "*.log", ".DS_Store"})

	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"node_modules", true, true},
		{"packages/app/node_modules", true, true},
		{"debug.log", false, true},
		{"logs/today/debug.log", false, true},
		{".DS_Store", false, true},
		{"src/main.go", false, false},
		{"node_modules.txt", false, false},
	}
	for _, test := range cases {
		if got := matcher.matches(test.path, test.isDir); got != test.want {
			t.Errorf("matches(%q, %v) = %v, want %v", test.path, test.isDir, got, test.want)
		}
	}
}

func TestExcludeMatcherAnchorsPatternsWithSlash(t *testing.T) {
	matcher := newExcludeMatcher([]string{"build/output", "**/generated", "dist/"})

	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"build/output", true, true},
		{"build/output", false, true},
		{"other/build/output", false, false},
		{"src/generated", true, true},
		{"generated", true, true},
		{"dist", true, true},
		{"dist", false, false},
		{"src/dist", true, true},
	}
	for _, test := range cases {
		if got := matcher.matches(test.path, test.isDir); got != test.want {
			t.Errorf("matches(%q, %v) = %v, want %v", test.path, test.isDir, got, test.want)
		}
	}
}

func TestExcludeMatcherIgnoresBlankPatterns(t *testing.T) {
	matcher := newExcludeMatcher([]string{"", "   ", "/"})
	if !matcher.empty() {
		t.Fatalf("empty patterns produced %d rules", len(matcher.patterns))
	}
	if matcher.matches("src/main.go", false) {
		t.Fatal("empty matcher hid a file")
	}
}

func TestReadFolderExcludingDropsExcludedEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, dir, "debug.log", "noise\n")
	writeTempFile(t, dir, "main.go", "package main\n")

	entries, err := readFolderExcluding(dir, newExcludeMatcher([]string{"node_modules", "*.log"}))
	if err != nil {
		t.Fatalf("readFolderExcluding() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "main.go" {
		t.Fatalf("readFolderExcluding() = %+v, want only main.go", entries)
	}
}
