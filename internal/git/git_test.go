package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kaze/internal/ignore"
)

func TestParseGitStatus(t *testing.T) {
	root := "/repo"
	record := func(parts ...string) string {
		return strings.Join(parts, "\x00") + "\x00"
	}
	data := record(
		"# branch.oid 1234567890abcdef",
		"# branch.head main",
		"# branch.ab +2 -3",
		"1 .M N... 100644 100644 100644 aaaaaaa bbbbbbb modified.txt",
		"1 A. N... 100644 100644 100644 aaaaaaa bbbbbbb added.txt",
		"1 .D N... 100644 100644 100644 aaaaaaa bbbbbbb deleted.txt",
		"? untracked.txt",
		"2 R. N... 100644 100644 100644 aaaaaaa bbbbbbb R100 renamed.txt",
		"old.txt",
		"u UU N... 100644 100644 100644 100644 aaaaaaa bbbbbbb ccccccc conflict.txt",
	)

	repo, changes, err := parseGitStatus(root, []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if repo.Branch != "main" || repo.Ahead != 2 || repo.Behind != 3 {
		t.Fatalf("repo = %+v", repo)
	}

	statuses := make(map[string]string)
	for _, change := range changes {
		statuses[filepath.Base(change.Path)] = change.Status
	}
	want := map[string]string{
		"modified.txt":  "modified",
		"added.txt":     "added",
		"deleted.txt":   "deleted",
		"untracked.txt": "untracked",
		"renamed.txt":   "renamed",
		"conflict.txt":  "conflicted",
	}
	if len(statuses) != len(want) {
		t.Fatalf("changes = %+v", changes)
	}
	for name, status := range want {
		if statuses[name] != status {
			t.Errorf("%s = %q, want %q", name, statuses[name], status)
		}
	}
}

func TestDiscoverRepos(t *testing.T) {
	root := t.TempDir()
	makeRepo := func(elem ...string) string {
		dir := filepath.Join(append([]string{root}, elem...)...)
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	one := makeRepo("one")
	two := makeRepo("nested", "two")
	// A repository beneath another one belongs to the outer repository, so the
	// walk stops instead of reporting it separately.
	makeRepo("one", "vendor")
	// files.exclude hides a directory from the tree, so it is not searched.
	makeRepo("ignored", "three")

	repos := DiscoverRepos(root, ignore.NewMatcher([]string{"ignored"}))
	got := make(map[string]bool, len(repos))
	for _, repo := range repos {
		got[repo] = true
	}
	if len(got) != 2 || !got[one] || !got[two] {
		t.Fatalf("repos = %+v, want %q and %q", repos, one, two)
	}

	if repos := DiscoverRepos(one, ignore.Matcher{}); len(repos) != 1 || repos[0] != one {
		t.Fatalf("repos for a repository root = %+v, want %q", repos, one)
	}
	if repos := DiscoverRepos(filepath.Join(root, "missing"), ignore.Matcher{}); len(repos) != 0 {
		t.Fatalf("repos for a missing root = %+v, want none", repos)
	}
}
