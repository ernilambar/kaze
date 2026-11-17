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
// Modified is the file's mtime in Unix milliseconds when the search read it.
// Replace All hands it back so a file changed since the search is skipped.
type SearchMatch struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Text     string `json:"text"`
	Modified int64  `json:"modified"`
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
// results panel shows. ExpectedModified maps a path to the mtime the search
// read it at, in Unix milliseconds; a file that moved since is not rewritten.
// OpenPaths names files open in a tab in the calling window: Replace All edits
// those in the editor's buffer instead of writing them here, so they stay
// unsaved and undoable.
type SearchRequest struct {
	Query            string           `json:"query"`
	Regex            bool             `json:"regex"`
	CaseSensitive    bool             `json:"case_sensitive"`
	Include          string           `json:"include"`
	Exclude          string           `json:"exclude"`
	Replacement      string           `json:"replacement"`
	ExpectedModified map[string]int64 `json:"expected_modified"`
	OpenPaths        []string         `json:"open_paths"`
}

// ReplaceResult reports how much Replace All changed. Skipped lists the files
// whose mtime moved since the search, so the frontend can report them instead
// of silently leaving a stale replacement behind.
//
// Failed names files that could not be read back or written. One failure does
// not stop the rest: by then earlier files are already rewritten, and the
// summary must describe what happened.
type ReplaceResult struct {
	Files   int      `json:"files"`
	Matches int      `json:"matches"`
	Skipped []string `json:"skipped"`
	Failed  []string `json:"failed"`
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
	roots, err := a.LoadSidebarFolders()
	if err != nil {
		return SearchResult{}, err
	}

	files := collectSearchFiles(settings, roots, request)
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
// writing. A file whose mtime moved since the search is skipped rather than
// overwritten: someone may have edited it, and Replace All must not discard
// that change. Skipped paths are reported so the frontend can say so.
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
	roots, err := a.LoadSidebarFolders()
	if err != nil {
		return ReplaceResult{}, err
	}

	open := make(map[string]bool, len(request.OpenPaths))
	for _, path := range request.OpenPaths {
		open[path] = true
	}
	result := ReplaceResult{Skipped: make([]string, 0), Failed: make([]string, 0)}
	for _, path := range collectSearchFiles(settings, roots, request) {
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
			changed, matches, err := replaceInFile(path, expression, request, false)
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
		changed, matches, err := replaceInFile(path, expression, request, true)
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
func collectSearchFiles(settings Settings, roots []string, request SearchRequest) []string {
	excludes := newExcludeMatcher(mergePatterns(
		settings.Files.Exclude,
		settings.Search.Exclude,
		splitPatternList(request.Exclude),
	))
	includes := newExcludeMatcher(splitPatternList(request.Include))

	files := make([]string, 0)
	seen := make(map[string]bool)
	for _, root := range roots {
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
	data, modified, ok := readSearchFile(path)
	if !ok {
		return
	}
	// modified is the mtime of the bytes just read: Replace All uses it to
	// detect a file that changed after this search, so a stale hit never
	// overwrites newer work on disk.
	name := filepath.Base(path)
	for index, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSuffix(line, []byte("\r"))
		for _, location := range expression.FindAllIndex(line, -1) {
			if len(result.Matches) >= maxSearchMatches {
				result.Truncated = true
				return
			}
			result.Matches = append(result.Matches, SearchMatch{
				Path:     path,
				Name:     name,
				Line:     index + 1,
				Column:   utf16Column(line, location[0]),
				Text:     previewLine(line),
				Modified: modified,
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

// readSearchFile reads a file a search may look at, returning its mtime in
// Unix milliseconds so a later Replace All can detect a change. Zero bytes mark
// a binary file, and invalid UTF-8 would make previews unreadable.
func readSearchFile(path string) ([]byte, int64, bool) {
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxSearchFileSize {
		return nil, 0, false
	}
	modified := info.ModTime().UnixMilli()
	data, err := os.ReadFile(path)
	if err != nil || bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return nil, 0, false
	}
	return data, modified, true
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
// When write is false it only counts: an open file is edited in the buffer
// instead, so the bytes on disk stay as they are.
func replaceInFile(
	path string,
	expression *regexp.Regexp,
	request SearchRequest,
	write bool,
) (bool, int, error) {
	data, _, ok := readSearchFile(path)
	if !ok {
		return false, 0, nil
	}

	replaced, matches := replaceLines(data, expression, request.Replacement, request.Regex)
	if matches == 0 || bytes.Equal(replaced, data) {
		return false, matches, nil
	}
	if !write {
		return true, matches, nil
	}
	if err := atomicWriteFile(path, replaced); err != nil {
		return false, 0, fmt.Errorf("replace in %s: %w", filepath.Base(path), err)
	}
	return true, matches, nil
}

// replaceLines replaces matches one line at a time, the way searchFile finds
// them: a pattern never spans lines, ^ and $ hold at each line, and a CRLF
// ending is left alone. replaceText in search.ts does the same to open
// buffers, so the disk and the editor end up with the same text.
func replaceLines(data []byte, expression *regexp.Regexp, replacement string, regex bool) ([]byte, int) {
	lines := bytes.Split(data, []byte("\n"))
	out := make([]byte, 0, len(data))
	total := 0
	for index, line := range lines {
		if index > 0 {
			out = append(out, '\n')
		}
		body := bytes.TrimSuffix(line, []byte("\r"))
		ending := line[len(body):]
		locations := expression.FindAllSubmatchIndex(body, -1)
		total += len(locations)
		last := 0
		for _, location := range locations {
			out = append(out, body[last:location[0]]...)
			if regex {
				out = expandReplacement(out, replacement, expression, body, location)
			} else {
				out = append(out, replacement...)
			}
			last = location[1]
		}
		out = append(out, body[last:]...)
		out = append(out, ending...)
	}
	return out, total
}

// expandReplacement appends one regex replacement using JavaScript's
// String.prototype.replace template, which open buffers are edited with: $$,
// $&, $`, $', $1–$99 and $<name>. A reference that names no group stays text,
// as it does in JavaScript.
func expandReplacement(out []byte, template string, expression *regexp.Regexp, line []byte, match []int) []byte {
	groups := len(match)/2 - 1
	group := func(index int) []byte {
		if match[2*index] < 0 {
			return nil
		}
		return line[match[2*index]:match[2*index+1]]
	}
	for i := 0; i < len(template); i++ {
		if template[i] != '$' || i+1 == len(template) {
			out = append(out, template[i])
			continue
		}
		next := template[i+1]
		switch {
		case next == '$':
			out = append(out, '$')
			i++
		case next == '&':
			out = append(out, group(0)...)
			i++
		case next == '`':
			out = append(out, line[:match[0]]...)
			i++
		case next == '\'':
			out = append(out, line[match[1]:]...)
			i++
		case next >= '0' && next <= '9':
			// Two digits win when they name a group, as in JavaScript.
			if i+2 < len(template) && template[i+2] >= '0' && template[i+2] <= '9' {
				if two := int(next-'0')*10 + int(template[i+2]-'0'); two >= 1 && two <= groups {
					out = append(out, group(two)...)
					i += 2
					continue
				}
			}
			if one := int(next - '0'); one >= 1 && one <= groups {
				out = append(out, group(one)...)
				i++
				continue
			}
			out = append(out, '$')
		case next == '<':
			end := strings.IndexByte(template[i+2:], '>')
			index := -1
			if end >= 0 {
				index = expression.SubexpIndex(template[i+2 : i+2+end])
			}
			if index < 0 {
				out = append(out, '$')
				continue
			}
			out = append(out, group(index)...)
			i += 2 + end
		default:
			out = append(out, '$')
		}
	}
	return out
}
