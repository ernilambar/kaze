package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeWatchFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A repository's working tree does not move when HEAD does, so the watcher
// needs the metadata files that a commit or a stage rewrites.
func TestWatchableAddsGitState(t *testing.T) {
	repo := t.TempDir()
	writeWatchFile(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	plain := t.TempDir()

	paths := watchable([]string{plain}, []string{repo, plain})
	got := make(map[string]bool, len(paths))
	for _, path := range paths {
		got[path] = true
	}

	for _, name := range []string{"HEAD", "index", "COMMIT_EDITMSG", filepath.Join("logs", "HEAD")} {
		want := filepath.Join(repo, ".git", name)
		if !got[want] {
			t.Errorf("watchable is missing %s: %v", want, paths)
		}
	}
	if !got[plain] {
		t.Errorf("watchable dropped the window path %s", plain)
	}
	if len(paths) != 1+len(gitStateFiles(repo)) {
		t.Errorf("watchable = %v, want one window path plus the git state", paths)
	}
}

func TestWatcherReportsCommitMetadata(t *testing.T) {
	repo := t.TempDir()
	index := filepath.Join(repo, ".git", "index")
	writeWatchFile(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeWatchFile(t, index, "staged")

	w := &fileWatcher{}
	w.set("main", watchable(nil, []string{repo}))

	// Force a timestamp the baseline cannot already hold, so the test does not
	// depend on filesystem timestamp resolution.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(index, future, future); err != nil {
		t.Fatal(err)
	}

	changed := w.poll()
	if len(changed["main"]) != 1 || changed["main"][0] != index {
		t.Fatalf("poll = %v, want the index change for main", changed)
	}
	if next := w.poll(); len(next) != 0 {
		t.Fatalf("second poll = %v, want the new state as the baseline", next)
	}
}

func TestGitDirFollowsWorktreePointer(t *testing.T) {
	worktree := t.TempDir()
	dir := filepath.Join(t.TempDir(), "worktrees", "wt")
	writeWatchFile(t, filepath.Join(dir, "HEAD"), "ref: refs/heads/main\n")
	writeWatchFile(t, filepath.Join(worktree, ".git"), "gitdir: "+dir+"\n")

	if got := gitDir(worktree); got != dir {
		t.Fatalf("gitDir = %q, want %q", got, dir)
	}
}
