package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	// maxSearchMatches caps what one search returns. The results list is meant
	// to narrow a search, not to page through a whole repository.
	maxSearchMatches = 500
	// maxSearchFileSize skips files that are too large to be source: reading
	// them would slow the walk far more than any match could help.
	maxSearchFileSize = 2 << 20
	// maxSearchPreview is how much of a matching line the results list shows.
	maxSearchPreview = 240
)

// vcsDirectories are never searched, even when the user removes them from
// search.exclude: version control metadata is machine state, not project text.
var vcsDirectories = map[string]bool{".git": true, ".hg": true, ".svn": true}

// SearchMatch is one hit: where it is and the trimmed line it was found on.
type SearchMatch struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Text   string `json:"text"`
}

// SearchResult is every hit a project search found, plus whether the cap cut
// the list short.
type SearchResult struct {
	Matches   []SearchMatch `json:"matches"`
	Truncated bool          `json:"truncated"`
	Files     int           `json:"files"`
}

// SearchRequest is one Find in Files query. The literal/regular-expression
// switch, the case option and the include and exclude globs mirror what the
// results panel shows.
type SearchRequest struct {
	Query         string `json:"query"`
	Regex         bool   `json:"regex"`
	CaseSensitive bool   `json:"case_sensitive"`
	Include       string `json:"include"`
	Exclude       string `json:"exclude"`
	Replacement   string `json:"replacement"`
}

// ReplaceResult reports how much Replace All changed.
type ReplaceResult struct {
	Files   int `json:"files"`
	Matches int `json:"matches"`
}

// SearchProject finds query across the sidebar roots. files.exclude and
// search.exclude decide what is read, and every .gitignore file on the way is
// honoured, so generated and ignored trees stay out of the results.
func (a *App) SearchProject(request SearchRequest) (SearchResult, error) {
	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" {
		return SearchResult{Matches: []SearchMatch{}}, nil
	}

	expression, err := searchExpression(request)
	if err != nil {
		return SearchResult{}, err
	}
	settings, err := a.searchSettings()
	if err != nil {
		return SearchResult{}, err
	}

	files := collectSearchFiles(settings, request)
	result := SearchResult{Matches: make([]SearchMatch, 0), Files: len(files)}
	for _, path := range files {
		searchFile(path, expression, &result)
		if result.Truncated {
			break
		}
	}
	return result, nil
}

// ReplaceInProject rewrites every match in every searched file. It searches
// the same file set first, so a file with no match is never opened for
// writing.
func (a *App) ReplaceInProject(request SearchRequest) (ReplaceResult, error) {
	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" {
		return ReplaceResult{}, errors.New("nothing to replace")
	}

	expression, err := searchExpression(request)
	if err != nil {
		return ReplaceResult{}, err
	}
	settings, err := a.searchSettings()
	if err != nil {
		return ReplaceResult{}, err
	}

	result := ReplaceResult{}
	for _, path := range collectSearchFiles(settings, request) {
		changed, matches, err := replaceInFile(path, expression, request)
		if err != nil {
			return result, err
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
func (a *App) searchSettings() (Settings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return defaultSettings(), err
	}
	return loadSettings(path)
}

// searchExpression compiles the request. A literal query is quoted, so a
// period is a period; the case option becomes a regexp flag because it applies
// to both modes.
func searchExpression(request SearchRequest) (*regexp.Regexp, error) {
	expression := request.Query
	if !request.Regex {
		expression = regexp.QuoteMeta(expression)
	}
	if !request.CaseSensitive {
		expression = "(?i)" + expression
	}
	compiled, err := regexp.Compile(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", err)
	}
	return compiled, nil
}

// collectSearchFiles walks every root and returns the files a search should
// read. The filters are applied while walking, so an excluded tree is never
// descended into.
func collectSearchFiles(settings Settings, request SearchRequest) []string {
	excludes := newExcludeMatcher(mergePatterns(
		settings.Files.Exclude,
		settings.Search.Exclude,
		splitPatternList(request.Exclude),
	))
	includes := newExcludeMatcher(splitPatternList(request.Include))

	files := make([]string, 0)
	seen := make(map[string]bool)
	for _, root := range settings.Sidebar.Folders {
		root = filepath.Clean(root)
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			continue
		}
		walkSearchRoot(root, "", excludes, includes, nil, func(path string) {
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
		})
	}
	// A stable order makes results repeatable and lets the frontend group them
	// by file without sorting again.
	sort.Strings(files)
	return files
}

// walkSearchRoot visits one directory and descends into the children that
// survive the filters. rules carries the .gitignore rules of the ancestors,
// because a pattern in a parent directory applies to everything below it.
func walkSearchRoot(
	dir string,
	rel string,
	excludes excludeMatcher,
	includes excludeMatcher,
	rules []ignoreRule,
	visit func(path string),
) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	if local := readIgnoreRules(dir); len(local) > 0 {
		merged := make([]ignoreRule, 0, len(rules)+len(local))
		merged = append(merged, rules...)
		merged = append(merged, local...)
		rules = merged
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		path := filepath.Join(dir, name)

		if entry.IsDir() {
			if vcsDirectories[strings.ToLower(name)] {
				continue
			}
			if excludes.matches(childRel, true) || ignored(rules, path, true) {
				continue
			}
			walkSearchRoot(path, childRel, excludes, includes, rules, visit)
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}
		if excludes.matches(childRel, false) || ignored(rules, path, false) {
			continue
		}
		if !includes.empty() && !includes.matches(childRel, false) {
			continue
		}
		visit(path)
	}
}

// searchFile appends every match in one file to result. Binary and oversized
// files are skipped silently: they are not source, and reporting them would
// bury the hits the user wants.
func searchFile(path string, expression *regexp.Regexp, result *SearchResult) {
	data, ok := readSearchFile(path)
	if !ok {
		return
	}

	name := filepath.Base(path)
	for index, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSuffix(line, []byte("\r"))
		for _, location := range expression.FindAllIndex(line, -1) {
			if len(result.Matches) >= maxSearchMatches {
				result.Truncated = true
				return
			}
			result.Matches = append(result.Matches, SearchMatch{
				Path:   path,
				Name:   name,
				Line:   index + 1,
				Column: utf16Column(line, location[0]),
				Text:   previewLine(line),
			})
		}
	}
}

// utf16Column converts a byte offset to the 1-based UTF-16 column Monaco
// places its cursors with, so a result on a line containing an emoji still
// lands on the right character.
func utf16Column(line []byte, byteIndex int) int {
	if byteIndex <= 0 {
		return 1
	}
	return len(utf16.Encode([]rune(string(line[:byteIndex])))) + 1
}

// readSearchFile reads a file a search may look at. Zero bytes mark a binary
// file, and invalid UTF-8 would make previews unreadable.
func readSearchFile(path string) ([]byte, bool) {
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxSearchFileSize {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil || bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return nil, false
	}
	return data, true
}

// previewLine is the line shown beside a match: leading indentation removed
// and the text capped so one long line cannot stretch the results list.
func previewLine(line []byte) string {
	text := strings.TrimLeft(string(line), " \t")
	if utf8.RuneCountInString(text) <= maxSearchPreview {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxSearchPreview]) + "…"
}

// replaceInFile rewrites one file's matches. It reports whether the file
// changed and how many matches were replaced, so Replace All can summarize.
func replaceInFile(path string, expression *regexp.Regexp, request SearchRequest) (bool, int, error) {
	data, ok := readSearchFile(path)
	if !ok {
		return false, 0, nil
	}

	matches := len(expression.FindAllIndex(data, -1))
	if matches == 0 {
		return false, 0, nil
	}

	var replaced []byte
	if request.Regex {
		replaced = expression.ReplaceAll(data, []byte(request.Replacement))
	} else {
		// A literal query must treat $ in the replacement as text, not as a
		// capture-group reference.
		replaced = expression.ReplaceAllLiteral(data, []byte(request.Replacement))
	}
	if bytes.Equal(replaced, data) {
		return false, matches, nil
	}
	if err := os.WriteFile(path, replaced, 0o644); err != nil {
		return false, 0, fmt.Errorf("replace in %s: %w", filepath.Base(path), err)
	}
	return true, matches, nil
}
