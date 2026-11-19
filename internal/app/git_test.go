package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs one git command in dir and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

// initGitRepo initializes a repository in dir. Tests skip when git is not
// installed, so a machine without it still runs the rest of the suite.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	runGit(t, dir, "init", "-q", "-b", "main", ".")
}

// gitRepo makes a throwaway repository with one commit.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	initGitRepo(t, dir)
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGitStatusReportsWorkingTree(t *testing.T) {
	dir := gitRepo(t)

	tracked := filepath.Join(dir, "tracked.txt")
	deleted := filepath.Join(dir, "deleted.txt")
	added := filepath.Join(dir, "added.txt")
	untracked := filepath.Join(dir, "untracked.txt")
	writeFile(t, tracked, "one\n")
	writeFile(t, deleted, "gone\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-qm", "init")

	writeFile(t, tracked, "one\ntwo\n")
	os.Remove(deleted)
	writeFile(t, added, "added\n")
	runGit(t, dir, "add", "added.txt")
	writeFile(t, untracked, "untracked\n")

	snapshot, err := (&App{}).GetGitStatus([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Repos) != 1 {
		t.Fatalf("repos = %+v", snapshot.Repos)
	}
	if snapshot.Repos[0].Branch != "main" {
		t.Errorf("branch = %q", snapshot.Repos[0].Branch)
	}
	if snapshot.Repos[0].Root != dir {
		t.Errorf("repo root = %q, want %q", snapshot.Repos[0].Root, dir)
	}

	statuses := make(map[string]string)
	for _, change := range snapshot.Changes {
		statuses[change.Path] = change.Status
		if change.Root != dir {
			t.Errorf("change root = %q, want %q", change.Root, dir)
		}
	}
	for path, want := range map[string]string{
		tracked:   "modified",
		added:     "added",
		deleted:   "deleted",
		untracked: "untracked",
	} {
		if statuses[path] != want {
			t.Errorf("%s = %q, want %q", path, statuses[path], want)
		}
	}
}

func TestGitBaselineAndDiff(t *testing.T) {
	dir := gitRepo(t)

	path := filepath.Join(dir, "file.txt")
	writeFile(t, path, "one\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-qm", "init")

	baseline, err := (&App{}).GetGitBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if !baseline.InRepo || !baseline.Tracked || baseline.Content != "one\n" {
		t.Fatalf("baseline = %+v", baseline)
	}

	writeFile(t, path, "one\ntwo\n")
	diff, err := (&App{}).GetGitDiff(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "+two") || !strings.Contains(diff, "+++ b/file.txt") {
		t.Fatalf("diff = %q", diff)
	}

	untracked := filepath.Join(dir, "untracked.txt")
	writeFile(t, untracked, "new\n")
	baseline, err = (&App{}).GetGitBaseline(untracked)
	if err != nil {
		t.Fatal(err)
	}
	if !baseline.InRepo || baseline.Tracked || baseline.Content != "" {
		t.Fatalf("untracked baseline = %+v", baseline)
	}
	diff, err = (&App{}).GetGitDiff(untracked)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "+new") {
		t.Fatalf("untracked diff = %q", diff)
	}

	writeFile(t, filepath.Join(dir, ".gitignore"), "ignored.env\n")
	ignored := filepath.Join(dir, "ignored.env")
	writeFile(t, ignored, "secret\n")
	baseline, err = (&App{}).GetGitBaseline(ignored)
	if err != nil {
		t.Fatal(err)
	}
	if !baseline.InRepo || !baseline.Ignored || baseline.Tracked || baseline.Content != "" {
		t.Fatalf("ignored baseline = %+v", baseline)
	}
}

func TestGitOutsideRepository(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	writeFile(t, path, "text\n")

	baseline, err := (&App{}).GetGitBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.InRepo {
		t.Fatalf("baseline = %+v", baseline)
	}

	if _, err := (&App{}).GetGitDiff(path); err == nil {
		t.Fatal("expected an error for a file outside a repository")
	}
	snapshot, err := (&App{}).GetGitStatus([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Repos) != 0 || len(snapshot.Changes) != 0 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

// A sidebar root can be a container of repositories. Discovery walks the
// container, so its nested repositories report changes whether or not the tree
// has expanded them.
func TestGitStatusFindsRepositoryNestedUnderRoot(t *testing.T) {
	container := t.TempDir()
	dir := filepath.Join(container, "nested")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, dir)

	path := filepath.Join(dir, "notes.txt")
	writeFile(t, path, "one\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-qm", "init")
	writeFile(t, path, "two\n")

	snapshot, err := (&App{}).GetGitStatus([]string{container})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Repos) != 1 {
		t.Fatalf("repos = %+v, want the nested repository", snapshot.Repos)
	}
	if snapshot.Repos[0].Root != dir {
		t.Errorf("repo root = %q, want %q", snapshot.Repos[0].Root, dir)
	}
	if len(snapshot.Changes) != 1 {
		t.Fatalf("changes = %+v, want the modified file", snapshot.Changes)
	}
	change := snapshot.Changes[0]
	if change.Path != path || change.Root != dir || change.Status != "modified" {
		t.Errorf("change = %+v, want %q under %q as modified", change, path, dir)
	}
}
