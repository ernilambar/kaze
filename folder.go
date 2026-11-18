package main

import (
	"errors"

	"kaze/internal/config"
	"kaze/internal/fileops"
	"kaze/internal/ignore"
)

// ReadFolder lists one level of a folder for the sidebar. The tree loads the
// rest lazily, so only the entries the user can see are read. Entries that
// files.exclude hides are left out, which keeps the tree and Quick Open
// agreeing on what exists.
//
// root is the sidebar folder path belongs to. Patterns match the entry's path
// relative to it, so an anchored `/build` hides only the top-level folder and
// `docs/gen` hides a nested one, the same as Quick Open.
func (a *App) ReadFolder(path, root string) ([]fileops.FolderEntry, error) {
	return fileops.ReadFolder(path, root, a.filesExcludeMatcher())
}

// filesExcludeMatcher loads the files.exclude patterns. A missing or broken
// settings file hides nothing: browsing still works while the user repairs it.
func (a *App) filesExcludeMatcher() ignore.Matcher {
	path, err := settingsFilePath()
	if err != nil {
		return ignore.Matcher{}
	}
	settings, err := config.Load(path)
	if err != nil {
		return ignore.Matcher{}
	}
	return ignore.NewMatcher(settings.Files.Exclude)
}

// OpenFolderDialog asks for a folder to add to the sidebar. An empty string
// means the user cancelled, which is not an error.
func (a *App) OpenFolderDialog() (string, error) {
	if a.wails == nil {
		return "", errors.New("folder dialog is not available before startup")
	}
	return a.wails.Dialog.OpenFile().
		SetTitle("Add Folder to Sidebar").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		PromptForSingleSelection()
}
