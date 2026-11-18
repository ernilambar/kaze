package main

import (
	"kaze/internal/config"
	"kaze/internal/fileops"
	"kaze/internal/ignore"
)

// ListFiles returns every regular file under one window's sidebar roots,
// sorted by path. The frontend filters this list locally, so each keystroke in
// Quick Open costs nothing. files.exclude decides what is walked, so a folder
// hidden in the tree is hidden here too; .gitignore files and VCS directories
// are skipped as well.
func (a *App) ListFiles(window string) (fileops.FileList, error) {
	path, err := settingsFilePath()
	if err != nil {
		return fileops.FileList{}, err
	}
	settings, err := config.Load(path)
	if err != nil {
		return fileops.FileList{}, err
	}
	roots, err := a.LoadSidebarFolders(window)
	if err != nil {
		return fileops.FileList{}, err
	}
	return fileops.ListFiles(roots, ignore.NewMatcher(settings.Files.Exclude)), nil
}
