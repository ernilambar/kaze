package git

import (
	"path/filepath"
	"strings"
	"testing"
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
