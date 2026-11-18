package main

import (
	"errors"
	"strings"

	"kaze/internal/config"
	"kaze/internal/find"
)

// SearchProject finds query across one window's sidebar roots. files.exclude
// and search.exclude decide what is read, and every .gitignore file on the way
// is honoured, so generated and ignored trees stay out of the results.
func (a *App) SearchProject(window string, request find.SearchRequest) (find.SearchResult, error) {
	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" {
		return find.SearchResult{Matches: []find.SearchMatch{}}, nil
	}

	expression, err := find.Expression(request)
	if err != nil {
		return find.SearchResult{}, err
	}
	settings, err := a.searchSettings()
	if err != nil {
		return find.SearchResult{}, err
	}
	roots, err := a.LoadSidebarFolders(window)
	if err != nil {
		return find.SearchResult{}, err
	}

	files := find.CollectFiles(settings, roots, request)
	result := find.SearchResult{Matches: make([]find.SearchMatch, 0), Files: len(files)}
	for _, path := range files {
		find.ScanFile(path, expression, &result)
		if result.Truncated {
			break
		}
	}
	return result, nil
}

// ReplaceInProject rewrites every match in every searched file. It searches
// the same file set first, so a file with no match is never opened for
// writing. A file whose mtime moved since the search is skipped rather than
// overwritten: someone may have edited it, and Replace All must not discard
// that change. Skipped paths are reported so the frontend can say so.
func (a *App) ReplaceInProject(window string, request find.SearchRequest) (find.ReplaceResult, error) {
	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" {
		return find.ReplaceResult{}, errors.New("nothing to replace")
	}

	expression, err := find.Expression(request)
	if err != nil {
		return find.ReplaceResult{}, err
	}
	settings, err := a.searchSettings()
	if err != nil {
		return find.ReplaceResult{}, err
	}
	roots, err := a.LoadSidebarFolders(window)
	if err != nil {
		return find.ReplaceResult{}, err
	}

	open := make(map[string]bool, len(request.OpenPaths))
	for _, path := range request.OpenPaths {
		open[path] = true
	}
	result := find.ReplaceResult{Skipped: make([]string, 0), Failed: make([]string, 0)}
	for _, path := range find.CollectFiles(settings, roots, request) {
		// Only files the search reported are candidates. Without an expected
		// mtime the frontend did not offer this file for replacement, so a
		// search run earlier cannot turn into a write here.
		expected, searched := request.ExpectedModified[path]
		if !searched {
			continue
		}
		// An open file is edited in the editor's buffer instead. Go still
		// counts its matches so the result describes everything replaced, but
		// it leaves the file on disk untouched for the user to save.
		if open[path] {
			changed, matches, err := find.ReplaceInFile(path, expression, request, false)
			if err != nil {
				result.Failed = append(result.Failed, path)
				continue
			}
			if changed {
				result.Files++
			}
			result.Matches += matches
			continue
		}
		conflict, err := changedOnDisk(path, expected)
		if err != nil {
			result.Failed = append(result.Failed, path)
			continue
		}
		if conflict {
			result.Skipped = append(result.Skipped, path)
			continue
		}
		// The watcher is not rebaselined: this file is not open here, and
		// another window that has it open must see the change.
		changed, matches, err := find.ReplaceInFile(path, expression, request, true)
		if err != nil {
			result.Failed = append(result.Failed, path)
			continue
		}
		if changed {
			result.Files++
		}
		result.Matches += matches
	}
	return result, nil
}

// searchSettings loads settings.json for a search. A missing or broken file
// means no roots, which produces an empty result rather than an error.
func (a *App) searchSettings() (config.Settings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return config.Default(), err
	}
	return config.Load(path)
}
