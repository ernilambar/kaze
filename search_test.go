package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// searchFixture writes a small project plus settings that point at it, and
// returns the root. HOME is redirected so settings.json lands in the test.
func searchFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, root, "main.go", "package main\n// TODO fix this\n")
	writeTempFile(t, root, "notes.txt", "TODO in text\n")
	writeTempFile(t, root, "ignored.go", "// TODO ignored\n")
	writeTempFile(t, filepath.Join(root, "node_modules"), "dep.js", "// TODO dependency\n")
	writeTempFile(t, filepath.Join(root, "src"), "helper.go", "// TODO helper\n")
	writeTempFile(t, root, ".gitignore", "ignored.go\n")

	path, err := settingsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	settings := defaultSettings()
	settings.Sidebar.Folders = []string{root}
	if err := writeSettings(path, settings); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSearchProjectRespectsExcludesAndGitignore(t *testing.T) {
	root := searchFixture(t)

	result, err := NewApp().SearchProject(SearchRequest{Query: "TODO"})
	if err != nil {
		t.Fatalf("SearchProject() error = %v", err)
	}

	var paths []string
	for _, match := range result.Matches {
		paths = append(paths, filepath.Base(match.Path))
	}
	want := []string{"main.go", "notes.txt", "helper.go"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("matches = %q, want %q (root %s)", paths, want, root)
	}
	if result.Matches[0].Line == 0 || result.Matches[0].Column == 0 {
		t.Fatalf("match position = %+v, want 1-based line and column", result.Matches[0])
	}
}

func TestSearchProjectHonoursIncludeGlobs(t *testing.T) {
	searchFixture(t)

	result, err := NewApp().SearchProject(SearchRequest{Query: "TODO", Include: "*.go"})
	if err != nil {
		t.Fatalf("SearchProject() error = %v", err)
	}
	for _, match := range result.Matches {
		if filepath.Ext(match.Path) != ".go" {
			t.Fatalf("match outside the include glob: %s", match.Path)
		}
	}
	if len(result.Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(result.Matches))
	}
}

func TestSearchProjectLiteralQueryIsNotARegex(t *testing.T) {
	searchFixture(t)

	result, err := NewApp().SearchProject(SearchRequest{Query: "TODO f"})
	if err != nil {
		t.Fatalf("SearchProject() error = %v", err)
	}
	if len(result.Matches) != 1 || result.Matches[0].Line != 2 {
		t.Fatalf("matches = %+v, want the literal TODO in main.go", result.Matches)
	}

	if _, err := NewApp().SearchProject(SearchRequest{Query: "(", Regex: true}); err == nil {
		t.Fatal("SearchProject() error = nil for an invalid regex")
	}
}

func TestReplaceInProjectRewritesMatches(t *testing.T) {
	root := searchFixture(t)

	result, err := NewApp().ReplaceInProject(SearchRequest{
		Query:       "TODO",
		Include:     "*.go",
		Replacement: "DONE",
	})
	if err != nil {
		t.Fatalf("ReplaceInProject() error = %v", err)
	}
	if result.Files != 2 || result.Matches != 2 {
		t.Fatalf("ReplaceInProject() = %+v, want 2 files and 2 matches", result)
	}

	data, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "TODO") || !strings.Contains(string(data), "DONE") {
		t.Fatalf("main.go = %q, want TODO replaced with DONE", data)
	}

	// A file ignored by .gitignore is outside the search set, so it is left
	// alone even though it matches.
	ignored, err := os.ReadFile(filepath.Join(root, "ignored.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ignored), "TODO") {
		t.Fatalf("ignored.go = %q, want its match untouched", ignored)
	}
}

func TestPreviewLineTrimsIndentationAndCapsLength(t *testing.T) {
	if got := previewLine([]byte("\t  hello")); got != "hello" {
		t.Fatalf("previewLine() = %q, want hello", got)
	}
	long := strings.Repeat("a", maxSearchPreview+50)
	if got := previewLine([]byte(long)); len([]rune(got)) != maxSearchPreview+1 {
		t.Fatalf("previewLine() length = %d, want %d", len([]rune(got)), maxSearchPreview+1)
	}
}

func TestUTF16ColumnCountsSurrogates(t *testing.T) {
	line := []byte("a😀TODO")
	index := strings.Index(string(line), "TODO")
	if index < 0 {
		t.Fatal("test line has no TODO")
	}
	if got := utf16Column(line, index); got != 4 {
		t.Fatalf("utf16Column() = %d, want 4", got)
	}
	if got := utf16Column(line, 0); got != 1 {
		t.Fatalf("utf16Column() at the start = %d, want 1", got)
	}
}
