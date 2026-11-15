package main

import (
	"os"
	"sync"
	"time"
)

// watchedChangedEvent tells one window that paths it asked about changed on
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
			w.paths[path] = statWatch(path)
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
func (w *fileWatcher) poll() map[string][]string {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := make(map[string][]string)
	for path, previous := range w.paths {
		current := statWatch(path)
		if current == previous {
			continue
		}
		w.paths[path] = current
		for window := range w.owners[path] {
			changed[window] = append(changed[window], path)
		}
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

// WatchPaths replaces one window's set of files and directories whose external
// changes it wants to hear about. Calling it the first time starts the polling
// loop, which runs for the life of the process.
func (a *App) WatchPaths(window string, paths []string) {
	a.watcher.set(window, paths)
	a.watchOnce.Do(func() {
		go a.watchLoop()
	})
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
