package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// searchFixture writes a small project plus session state that points at it,
// and returns the root. HOME is redirected so session.json lands in the test.
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

	if err := writeSessionFile(sessionFile{Sidebar: sidebarRoots{Roots: []string{root}}}); err != nil {
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
	app := NewApp()
	search, err := app.SearchProject(SearchRequest{Query: "TODO", Include: "*.go"})
	if err != nil {
		t.Fatal(err)
	}
	expected := searchExpected(search.Matches)

	result, err := app.ReplaceInProject(SearchRequest{
		Query:            "TODO",
		Include:          "*.go",
		Replacement:      "DONE",
		ExpectedModified: expected,
	})
	if err != nil {
		t.Fatalf("ReplaceInProject() error = %v", err)
	}
	if result.Files != 2 || result.Matches != 2 {
		t.Fatalf("ReplaceInProject() = %+v, want 2 files and 2 matches", result)
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("ReplaceInProject() skipped = %v, want none", result.Skipped)
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

// searchExpected mirrors what the frontend builds from search matches: the
// mtime of each file at the moment the search read it.
func searchExpected(matches []SearchMatch) map[string]int64 {
	expected := make(map[string]int64)
	for _, match := range matches {
		expected[match.Path] = match.Modified
	}
	return expected
}

func TestReplaceInProjectSkipsFilesChangedSinceSearch(t *testing.T) {
	root := searchFixture(t)
	app := NewApp()
	search, err := app.SearchProject(SearchRequest{Query: "TODO", Include: "*.go"})
	if err != nil {
		t.Fatal(err)
	}
	expected := searchExpected(search.Matches)

	// Another program edits main.go after the search: its mtime moves, so
	// Replace All must leave it alone and report it as skipped.
	writeTempFile(t, root, "main.go", "package main\n// TODO edited elsewhere\n")

	result, err := app.ReplaceInProject(SearchRequest{
		Query:            "TODO",
		Include:          "*.go",
		Replacement:      "DONE",
		ExpectedModified: expected,
	})
	if err != nil {
		t.Fatalf("ReplaceInProject() error = %v", err)
	}
	if result.Files != 1 || result.Matches != 1 {
		t.Fatalf("ReplaceInProject() = %+v, want 1 file and 1 match", result)
	}
	wantSkipped := filepath.Join(root, "main.go")
	if len(result.Skipped) != 1 || result.Skipped[0] != wantSkipped {
		t.Fatalf("ReplaceInProject() skipped = %v, want [%s]", result.Skipped, wantSkipped)
	}

	changed, err := os.ReadFile(wantSkipped)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changed), "// TODO edited elsewhere") {
		t.Fatalf("main.go = %q, want the external edit left untouched", changed)
	}
	other, err := os.ReadFile(filepath.Join(root, "src", "helper.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(other), "TODO") || !strings.Contains(string(other), "DONE") {
		t.Fatalf("helper.go = %q, want the unchanged file replaced", other)
	}
}

func TestReplaceInProjectLeavesOpenFilesOnDisk(t *testing.T) {
	root := searchFixture(t)
	app := NewApp()
	search, err := app.SearchProject(SearchRequest{Query: "TODO", Include: "*.go"})
	if err != nil {
		t.Fatal(err)
	}
	open := filepath.Join(root, "main.go")

	// main.go is open as a buffer, so Replace All edits it in the editor
	// instead. Go leaves the bytes on disk alone but still counts the match.
	result, err := app.ReplaceInProject(SearchRequest{
		Query:            "TODO",
		Include:          "*.go",
		Replacement:      "DONE",
		ExpectedModified: searchExpected(search.Matches),
		OpenPaths:        []string{open},
	})
	if err != nil {
		t.Fatalf("ReplaceInProject() error = %v", err)
	}
	if result.Files != 2 || result.Matches != 2 {
		t.Fatalf("ReplaceInProject() = %+v, want 2 files and 2 matches", result)
	}

	data, err := os.ReadFile(open)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "DONE") {
		t.Fatalf("main.go = %q, want the open buffer left unsaved on disk", data)
	}
	other, err := os.ReadFile(filepath.Join(root, "src", "helper.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(other), "DONE") {
		t.Fatalf("helper.go = %q, want the file not open in a buffer replaced", other)
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
