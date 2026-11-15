package main

import (
	"os"
	"sync"
	"time"
)

// watchedChangedEvent tells the frontend that paths it asked about changed on
// disk. The payload is a []string of absolute paths.
const watchedChangedEvent = "kova:watched-changed"

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
// open buffers plus the folders the tree has expanded.
type fileWatcher struct {
	mu    sync.Mutex
	paths map[string]watchState
}

// set replaces the watched set. A path that was already watched keeps its old
// state, so a change that landed between two registrations is still reported
// instead of becoming the new baseline.
func (w *fileWatcher) set(paths []string) {
	next := make(map[string]watchState, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		next[path] = statWatch(path)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	for path := range next {
		if previous, ok := w.paths[path]; ok {
			next[path] = previous
		}
	}
	w.paths = next
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

// poll compares every watched path with its baseline and returns the ones that
// changed. A deleted or recreated path counts as a change, so a tree refresh
// sees it.
func (w *fileWatcher) poll() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := make([]string, 0)
	for path, previous := range w.paths {
		current := statWatch(path)
		if current == previous {
			continue
		}
		w.paths[path] = current
		changed = append(changed, path)
	}
	return changed
}

// rebaseline records a path's current state. Saving writes a file itself, so
// the writer tells the watcher its own change is not news.
func (w *fileWatcher) rebaseline(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, watched := w.paths[path]; !watched {
		return
	}
	w.paths[path] = statWatch(path)
}

// WatchPaths replaces the set of files and directories whose external changes
// the frontend wants to hear about. Calling it the first time starts the
// polling loop, which runs for the life of the process.
func (a *App) WatchPaths(paths []string) {
	a.watcher.set(paths)
	a.watchOnce.Do(func() {
		go a.watchLoop()
	})
}

// watchLoop polls the watcher and forwards changes. It is one goroutine for
// the whole app, and it never holds the watcher lock while emitting.
func (a *App) watchLoop() {
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()
	for range ticker.C {
		changed := a.watcher.poll()
		if len(changed) > 0 {
			a.emitEventWith(watchedChangedEvent, changed)
		}
	}
}
