package ignore

import (
	"path"
	"strings"
)

// Matcher hides paths that match any of a list of glob patterns. It is
// shared by the explorer, Quick Open and Find in Files, so all three agree on
// what files.exclude and search.exclude mean.
//
// A pattern follows the shape editors use: a leading slash anchors it to the
// root, a pattern with no slash matches a name at any depth, `**` matches any
// number of directories, and a trailing slash applies to folders only.
type Matcher struct {
	patterns []excludePattern
}

type excludePattern struct {
	// segments is the pattern split on "/", so `**` can be handled as a whole
	// directory level instead of as characters inside one name.
	segments []string
	// anchored patterns must match the whole relative path; unanchored ones
	// match at any depth.
	anchored bool
	dirOnly  bool
}

func NewMatcher(patterns []string) Matcher {
	matcher := Matcher{patterns: make([]excludePattern, 0, len(patterns))}
	for _, raw := range patterns {
		pattern, ok := parseExcludePattern(raw)
		if !ok {
			continue
		}
		matcher.patterns = append(matcher.patterns, pattern)
	}
	return matcher
}

// parseExcludePattern normalizes one settings pattern. It reports false for a
// blank entry so a stray comma in settings.json does not hide everything.
func parseExcludePattern(raw string) (excludePattern, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" {
		return excludePattern{}, false
	}

	pattern := excludePattern{}
	if strings.HasSuffix(raw, "/") {
		pattern.dirOnly = true
		raw = strings.TrimSuffix(raw, "/")
	}
	if strings.HasPrefix(raw, "/") {
		pattern.anchored = true
		raw = strings.TrimPrefix(raw, "/")
	}
	raw = strings.TrimPrefix(raw, "./")
	if raw == "" {
		return excludePattern{}, false
	}

	pattern.segments = strings.Split(raw, "/")
	// A slash anywhere but the end makes the pattern root-relative, which is
	// how glob lists in editors behave.
	if len(pattern.segments) > 1 {
		pattern.anchored = true
	}
	return pattern, true
}

// Empty reports whether the matcher has nothing to hide.
func (m Matcher) Empty() bool {
	return len(m.patterns) == 0
}

// Matches reports whether relPath, always slash-separated and relative to a
// sidebar root, is excluded. isDir is needed for directory-only patterns.
func (m Matcher) Matches(relPath string, isDir bool) bool {
	if len(m.patterns) == 0 {
		return false
	}
	relPath = strings.Trim(strings.ReplaceAll(relPath, "\\", "/"), "/")
	if relPath == "" || relPath == "." {
		return false
	}
	segments := strings.Split(relPath, "/")

	for _, pattern := range m.patterns {
		if pattern.dirOnly && !isDir {
			continue
		}
		if globPathMatch(pattern, segments) {
			return true
		}
	}
	return false
}

// globPathMatch tries a pattern against a relative path. An unanchored pattern
// is tried at every directory level, so `node_modules` hides the folder even
// when it lives deep inside a repository.
func globPathMatch(pattern excludePattern, pathSegments []string) bool {
	if pattern.anchored {
		return matchSegments(pattern.segments, pathSegments)
	}
	for start := range len(pathSegments) {
		if matchSegments(pattern.segments, pathSegments[start:]) {
			return true
		}
	}
	return false
}

// matchSegments matches glob segments against path segments. `**` consumes
// zero or more whole segments; every other segment is a single-level match.
func matchSegments(pattern, pathSegments []string) bool {
	if len(pattern) == 0 {
		return len(pathSegments) == 0
	}
	if pattern[0] == "**" {
		if matchSegments(pattern[1:], pathSegments) {
			return true
		}
		for index := range len(pathSegments) {
			if matchSegments(pattern[1:], pathSegments[index+1:]) {
				return true
			}
		}
		return false
	}
	if len(pathSegments) == 0 {
		return false
	}
	matched, err := path.Match(pattern[0], pathSegments[0])
	if err != nil || !matched {
		return false
	}
	return matchSegments(pattern[1:], pathSegments[1:])
}

// SplitPatternList turns the comma or whitespace separated patterns a search
// field carries into a list. A pattern itself never contains a space.
func SplitPatternList(list string) []string {
	return strings.FieldsFunc(list, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

// MergePatterns joins several pattern lists, keeping the first occurrence of
// each so a default repeated by the user is only tested once.
func MergePatterns(lists ...[]string) []string {
	merged := make([]string, 0)
	seen := make(map[string]bool)
	for _, list := range lists {
		for _, pattern := range list {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" || seen[pattern] {
				continue
			}
			seen[pattern] = true
			merged = append(merged, pattern)
		}
	}
	return merged
}

// VCSDirectories are never searched, even when the user removes them from
// search.exclude: version control metadata is machine state, not project text.
var VCSDirectories = map[string]bool{".git": true, ".hg": true, ".svn": true}
