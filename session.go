package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// sessionVersion is the current session.json schema. Version 1 kept the
	// Open Recent list in settings.json; loading an older file moves it here.
	sessionVersion = 2
	// maxWorkspaceSessions caps the parked tab sets in session.json. A user
	// who opens many workspaces would otherwise keep every one forever.
	maxWorkspaceSessions = 50
	// maxSessionFiles caps one tab list, so a hand-edited session cannot make
	// a launch restore an unbounded number of tabs.
	maxSessionFiles = 40
	// maxRecentFiles caps the Open Recent list. More than a screenful is never
	// read.
	maxRecentFiles = 20
)

// WindowSession is the state a relaunch restores for one window. It lives in
// session.json, never in settings.json or a workspace file. Only tabs backed by
// a file are recorded; untitled buffers have no text on disk.
type WindowSession struct {
	// Workspace is the workspace file this window is showing, if any.
	Workspace string   `json:"workspace"`
	Folders   []string `json:"folders"`
	OpenFiles []string `json:"open_files"`
	Active    string   `json:"active"`
	Expanded  []string `json:"expanded"`
}

// WorkspaceSession is the tab set parked under a workspace file. Switching
// away parks the current tabs here; switching back restores them. Folders are
// the workspace file's resolved roots, used to adopt a session after the file
// itself was moved or renamed.
type WorkspaceSession struct {
	Folders   []string `json:"folders"`
	OpenFiles []string `json:"open_files"`
	Active    string   `json:"active"`
	Expanded  []string `json:"expanded"`
	Updated   int64    `json:"updated"`
}

// RecentFiles is the Open Recent history, newest first. It is state shared by
// every window, so it lives beside the tab sets instead of in settings.json.
type RecentFiles struct {
	Files []string `json:"files"`
}

// sessionFile is the on-disk shape of session.json. Window belongs to the one
// window that owns the launch session; every window may park a workspace tab
// set in Workspaces. Teak's per-window array is not used because Kaze still
// restores only the first window of a launch.
type sessionFile struct {
	Version    int                         `json:"version"`
	Recent     RecentFiles                 `json:"recent"`
	Window     WindowSession               `json:"window"`
	Workspaces map[string]WorkspaceSession `json:"workspaces"`
}

// sessionFilePath returns the app-written state file beside settings.json.
func sessionFilePath() (string, error) {
	path, err := settingsFilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "session.json"), nil
}

// emptySession is a store with the map the JSON encoder expects and no tabs.
func emptySession() sessionFile {
	return sessionFile{
		Version:    sessionVersion,
		Recent:     RecentFiles{Files: []string{}},
		Window:     WindowSession{},
		Workspaces: map[string]WorkspaceSession{},
	}
}

// loadSessionLocked reads session.json once and keeps it in App. A missing file
// starts empty; a malformed one is moved aside so the next write does not
// destroy whatever it held.
func (a *App) loadSessionLocked() (sessionFile, error) {
	if a.sessionLoaded {
		return a.sessionStore, nil
	}

	path, err := sessionFilePath()
	if err != nil {
		return emptySession(), err
	}
	data, err := os.ReadFile(path)
	migrateRecent := false
	switch {
	case errors.Is(err, os.ErrNotExist), err == nil && len(bytes.TrimSpace(data)) == 0:
		a.sessionStore = emptySession()
		migrateRecent = true
	case err != nil:
		return emptySession(), fmt.Errorf("read session file: %w", err)
	default:
		var state sessionFile
		if json.Unmarshal(data, &state) != nil {
			_ = os.Rename(path, path+".bak")
			state = emptySession()
			migrateRecent = true
		} else if state.Version < sessionVersion {
			migrateRecent = true
		}
		a.sessionStore = normalizeSession(state)
	}
	a.sessionLoaded = true
	if migrateRecent {
		a.adoptLegacyRecentLocked()
	}
	return a.sessionStore, nil
}

// adoptLegacyRecentLocked moves the Open Recent list that older builds kept in
// settings.json into session.json and drops the old key. A failed read or write
// leaves both files alone, so nothing is lost.
func (a *App) adoptLegacyRecentLocked() {
	settingsPath, err := settingsFilePath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}
	var legacy struct {
		Recent RecentFiles `json:"recent"`
	}
	if json.Unmarshal(data, &legacy) != nil {
		return
	}

	if len(a.sessionStore.Recent.Files) == 0 && len(legacy.Recent.Files) > 0 {
		a.sessionStore.Recent.Files = normalizeRecentFiles(legacy.Recent.Files)
		if err := writeSessionFile(a.sessionStore); err != nil {
			return
		}
	}

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	_ = dropSettingsSection(settingsPath, "recent")
}

// normalizeSession repairs a file written by hand or by a future version that
// added fields this build does not understand.
func normalizeSession(state sessionFile) sessionFile {
	state.Version = sessionVersion
	state.Recent.Files = normalizeRecentFiles(state.Recent.Files)
	state.Window = normalizeWindowSession(state.Window)
	if state.Workspaces == nil {
		state.Workspaces = map[string]WorkspaceSession{}
	}
	for key, workspace := range state.Workspaces {
		state.Workspaces[key] = normalizeWorkspaceSession(workspace)
	}
	return state
}

// normalizeRecentFiles trims entries, drops repeats past the cap and always
// returns a non-nil list.
func normalizeRecentFiles(files []string) []string {
	normalized := normalizeFileList(files)
	if len(normalized) > maxRecentFiles {
		normalized = normalized[:maxRecentFiles]
	}
	if normalized == nil {
		return []string{}
	}
	return normalized
}

// GetRecentFiles returns the Open Recent list, newest first. Every window reads
// the same list.
func (a *App) GetRecentFiles() ([]string, error) {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	state, err := a.loadSessionLocked()
	if err != nil {
		return nil, err
	}
	return state.Recent.Files, nil
}

func normalizeWindowSession(session WindowSession) WindowSession {
	session.Workspace = strings.TrimSpace(session.Workspace)
	if session.Workspace != "" {
		session.Workspace = canonicalPath(session.Workspace)
	}
	session.Folders = canonicalFolderList(session.Folders)
	session.OpenFiles = normalizeSessionFiles(session.OpenFiles)
	session.Active = strings.TrimSpace(session.Active)
	session.Expanded = canonicalFolderList(session.Expanded)
	return session
}

func normalizeWorkspaceSession(session WorkspaceSession) WorkspaceSession {
	session.Folders = canonicalFolderList(session.Folders)
	session.OpenFiles = normalizeSessionFiles(session.OpenFiles)
	session.Active = strings.TrimSpace(session.Active)
	session.Expanded = canonicalFolderList(session.Expanded)
	return session
}

// normalizeSessionFiles trims paths, drops repeats and applies the cap.
func normalizeSessionFiles(files []string) []string {
	normalized := normalizeFileList(files)
	if len(normalized) > maxSessionFiles {
		normalized = normalized[:maxSessionFiles]
	}
	if normalized == nil {
		return []string{}
	}
	return normalized
}

// canonicalFolderList resolves symlinks and drops repeats while keeping order.
// It is the identity a parked workspace is matched by after its file moved.
func canonicalFolderList(folders []string) []string {
	normalized := make([]string, 0, len(folders))
	seen := make(map[string]bool, len(folders))
	for _, folder := range folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}
		folder = canonicalPath(folder)
		if seen[folder] {
			continue
		}
		seen[folder] = true
		normalized = append(normalized, folder)
	}
	return normalized
}

// GetWindowSession returns the tabs this window should restore. Only the first
// window of a launch owns the session; later windows start with their own
// launch request instead of overwriting it.
func (a *App) GetWindowSession(window string) (WindowSession, error) {
	if owner := a.session(window); owner != nil && !owner.owner {
		return WindowSession{}, nil
	}

	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	state, err := a.loadSessionLocked()
	if err != nil {
		return WindowSession{}, err
	}
	return state.Window, nil
}

// SetSession records this window's tabs and the workspace they belong to. The
// owner window also updates the launch session; every window parks its current
// workspace set, so switching workspaces restores the tabs in any window.
func (a *App) SetSession(window string, session WindowSession) error {
	session = normalizeWindowSession(session)
	owner := a.session(window)
	updateWindow := owner == nil || owner.owner

	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	state, err := a.loadSessionLocked()
	if err != nil {
		return err
	}
	if updateWindow {
		state.Window = session
	}
	if session.Workspace != "" {
		key := canonicalPath(session.Workspace)
		state.Workspaces[key] = WorkspaceSession{
			Folders:   session.Folders,
			OpenFiles: session.OpenFiles,
			Active:    session.Active,
			Expanded:  session.Expanded,
			Updated:   time.Now().Unix(),
		}
		pruneWorkspaceSessions(&state, key)
	}
	a.sessionStore = state
	return writeSessionFile(state)
}

// parkedWorkspace returns the tab set saved for a workspace. The exact
// canonical path wins; otherwise a unique entry with the same resolved folders
// is adopted, which is what makes moving or renaming a workspace file keep its
// tabs. Two possible matches are left alone rather than guessed between.
func (a *App) parkedWorkspace(info WorkspaceInfo) WorkspaceSession {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	state, err := a.loadSessionLocked()
	if err != nil {
		return WorkspaceSession{}
	}
	key := canonicalPath(info.Path)
	if parked, ok := state.Workspaces[key]; ok {
		return parked
	}

	folders := canonicalFolderList(info.Folders)
	var matchKey string
	var match WorkspaceSession
	for candidateKey, parked := range state.Workspaces {
		if len(parked.Folders) == 0 || !sameFolderSet(parked.Folders, folders) {
			continue
		}
		// A workspace file that still exists is a copy, not a move: its
		// session must not be stolen by the new path.
		if _, err := os.Stat(candidateKey); err == nil {
			continue
		}
		if matchKey != "" {
			return WorkspaceSession{}
		}
		matchKey = candidateKey
		match = parked
	}
	if matchKey == "" {
		return WorkspaceSession{}
	}

	// Adopt the orphan under the new path. The old key no longer names a
	// workspace file, so keeping both would only duplicate the tab set.
	delete(state.Workspaces, matchKey)
	match.Updated = time.Now().Unix()
	state.Workspaces[key] = match
	a.sessionStore = state
	_ = writeSessionFile(state)
	return match
}

func sameFolderSet(a, b []string) bool {
	left := canonicalFolderList(a)
	right := canonicalFolderList(b)
	if len(left) != len(right) {
		return false
	}
	sort.Strings(left)
	sort.Strings(right)
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// pruneWorkspaceSessions keeps the most recently used entries. The workspace
// being written is always kept, even when its timestamp is the oldest.
func pruneWorkspaceSessions(state *sessionFile, keep string) {
	if len(state.Workspaces) <= maxWorkspaceSessions {
		return
	}
	keys := make([]string, 0, len(state.Workspaces))
	for key := range state.Workspaces {
		if key != keep {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return state.Workspaces[keys[i]].Updated < state.Workspaces[keys[j]].Updated
	})
	remove := len(state.Workspaces) - maxWorkspaceSessions
	for _, key := range keys {
		if remove == 0 {
			break
		}
		delete(state.Workspaces, key)
		remove--
	}
}

// writeSessionFile replaces session.json atomically. The file is private:
// workspace paths and open files are the user's, not project configuration.
func writeSessionFile(state sessionFile) error {
	path, err := sessionFilePath()
	if err != nil {
		return err
	}
	state = normalizeSession(state)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".session-*.json")
	if err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write session file: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("write session file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	return nil
}
