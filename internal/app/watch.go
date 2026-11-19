package app

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// watchedChangedEvent tells one window that paths it asked about changed on
// disk. The payload is a []string of absolute paths.
const watchedChangedEvent = "app:watched-changed"

// watchInterval is how often the watcher compares timestamps. One second is
// invisible to a person editing in another program, and a handful of stat
// calls per second costs nothing.
const watchInterval = time.Second

// watchState is what the watcher last saw for one path.
type watchState struct {
	modTime int64
	size    int64
	exists  bool
}

// fileWatcher polls a set of files and directories. Polling rather than
// FSEvents keeps the app free of a native dependency, and the set stays small:
// open buffers plus the folders each window's tree has expanded. Every path
// records which windows asked for it, so a change reaches exactly those windows
// and closing one stops its events.
type fileWatcher struct {
	mu     sync.Mutex
	paths  map[string]watchState
	owners map[string]map[string]bool
}

// set replaces one window's watched set. A path that was already watched keeps
// its old state, so a change that landed between two registrations is still
// reported instead of becoming the new baseline.
func (w *fileWatcher) set(window string, paths []string) {
	owned := make(map[string]bool, len(paths))
	for _, path := range paths {
		if path != "" {
			owned[path] = true
		}
	}
	// Stat before locking: a stalled volume must not block the poll or a save.
	fresh := make(map[string]watchState, len(owned))
	for path := range owned {
		fresh[path] = statWatch(path)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.owners == nil {
		w.owners = make(map[string]map[string]bool)
	}
	if w.paths == nil {
		w.paths = make(map[string]watchState)
	}
	for path, owners := range w.owners {
		if owned[path] {
			continue
		}
		delete(owners, window)
		if len(owners) == 0 {
			delete(w.owners, path)
			delete(w.paths, path)
		}
	}

	for path := range owned {
		if _, watched := w.paths[path]; !watched {
			w.paths[path] = fresh[path]
		}
		owners := w.owners[path]
		if owners == nil {
			owners = make(map[string]bool)
			w.owners[path] = owners
		}
		owners[window] = true
	}
}

// forget drops every path one window watched, so a closed window stops
// receiving change events.
func (w *fileWatcher) forget(window string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for path, owners := range w.owners {
		delete(owners, window)
		if len(owners) == 0 {
			delete(w.owners, path)
			delete(w.paths, path)
		}
	}
}

func statWatch(path string) watchState {
	info, err := os.Stat(path)
	if err != nil {
		return watchState{exists: false}
	}
	return watchState{
		modTime: info.ModTime().UnixMilli(),
		size:    info.Size(),
		exists:  true,
	}
}

// poll compares every watched path with its baseline and returns the paths that
// changed, grouped by the windows that asked about them. A deleted or recreated
// path counts as a change, so a tree refresh sees it.
//
// The stats run without the lock, so a slow volume delays only the poll. A
// path dropped or rebaselined meanwhile is left to the next poll.
func (w *fileWatcher) poll() map[string][]string {
	w.mu.Lock()
	snapshot := make(map[string]watchState, len(w.paths))
	for path, state := range w.paths {
		snapshot[path] = state
	}
	w.mu.Unlock()

	current := make(map[string]watchState, len(snapshot))
	for path, previous := range snapshot {
		if state := statWatch(path); state != previous {
			current[path] = state
		}
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	changed := make(map[string][]string)
	for path, state := range current {
		if baseline, watched := w.paths[path]; !watched || baseline != snapshot[path] {
			continue
		}
		w.paths[path] = state
		for window := range w.owners[path] {
			changed[window] = append(changed[window], path)
		}
	}
	return changed
}

// rebaseline records a path's current state. Saving writes a file itself, so
// the writer tells the watcher its own change is not news.
func (w *fileWatcher) rebaseline(path string) {
	state := statWatch(path)
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, watched := w.paths[path]; !watched {
		return
	}
	w.paths[path] = state
}

// WatchPaths replaces one window's set of files and directories whose external
// changes it wants to hear about. repos are the repositories behind the sidebar
// roots: their git metadata is watched too, because a commit, checkout or stage
// moves HEAD and the index without touching a working-tree file. Calling it the
// first time starts the polling loop, which runs for the life of the process.
func (a *App) WatchPaths(window string, paths []string, repos []string) {
	// A call still in flight when its window closed would register paths
	// after forget, and they would be polled for the rest of the process.
	if window == "" || a.session(window) == nil {
		return
	}
	a.watcher.set(window, watchable(paths, repos))
	a.watchOnce.Do(func() {
		go a.watchLoop()
	})
}

// watchable is the window's own paths plus the git metadata a repository
// rewrites when its state changes. The frontend knows the repositories from the
// status read; their working trees stay still when HEAD moves.
func watchable(paths, repos []string) []string {
	seen := make(map[string]bool, len(paths))
	all := make([]string, 0, len(paths))
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		all = append(all, path)
	}
	for _, path := range paths {
		add(path)
	}
	for _, repo := range repos {
		for _, path := range gitStateFiles(repo) {
			add(path)
		}
	}
	return all
}

// gitStateFiles names the files under a repository's git directory that move
// when HEAD, the index or a conflict state changes. A path may not exist yet;
// the watcher reports its creation like any other change.
func gitStateFiles(repo string) []string {
	dir := gitDir(repo)
	if dir == "" {
		return nil
	}
	names := []string{
		"HEAD",
		"index",
		"COMMIT_EDITMSG",
		"logs/HEAD",
		"packed-refs",
		"MERGE_HEAD",
		"CHERRY_PICK_HEAD",
		"REVERT_HEAD",
	}
	files := make([]string, 0, len(names))
	for _, name := range names {
		files = append(files, filepath.Join(dir, name))
	}
	return files
}

// gitDir is a repository's .git directory. A linked worktree or submodule
// stores a pointer file instead, whose gitdir line names the real directory.
func gitDir(repo string) string {
	dotGit := filepath.Join(repo, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return dotGit
	}
	data, err := os.ReadFile(dotGit)
	if err != nil {
		return ""
	}
	const prefix = "gitdir: "
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	dir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repo, dir)
	}
	return filepath.Clean(dir)
}

// watchLoop polls the watcher and forwards changes to the windows that asked
// for them. It is one goroutine for the whole app, and it never holds the
// watcher lock while emitting.
func (a *App) watchLoop() {
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()
	for range ticker.C {
		for window, changed := range a.watcher.poll() {
			if len(changed) == 0 {
				continue
			}
			if session := a.session(window); session != nil {
				session.emit(watchedChangedEvent, changed)
			}
		}
	}
}
