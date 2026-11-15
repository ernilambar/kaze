package main

import (
	"os"
	"path/filepath"
	"strings"
)

// ignoreRule is one line of a .gitignore file. Rules keep the directory they
// were read from, because a pattern is relative to that directory.
type ignoreRule struct {
	base    string
	pattern excludePattern
	negate  bool
}

// readIgnoreRules loads dir/.gitignore. A missing or unreadable file simply
// contributes no rules, which is also what git does for a file it cannot read.
func readIgnoreRules(dir string) []ignoreRule {
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return nil
	}
	return parseIgnoreRules(dir, string(data))
}

// parseIgnoreRules turns .gitignore content into rules. The parser covers the
// shapes people write in practice: comments, negations, directory-only
// patterns, anchoring and `**`. Escaped leading characters are left as-is.
func parseIgnoreRules(base, content string) []ignoreRule {
	rules := make([]ignoreRule, 0)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		rule := ignoreRule{base: base}
		if strings.HasPrefix(trimmed, "!") {
			rule.negate = true
			trimmed = strings.TrimPrefix(trimmed, "!")
		}
		// Unescaped trailing spaces are insignificant in gitignore.
		trimmed = strings.TrimRight(trimmed, " ")

		pattern, ok := parseExcludePattern(trimmed)
		if !ok {
			continue
		}
		rule.pattern = pattern
		rules = append(rules, rule)
	}
	return rules
}

// ignored reports whether path is ignored by rules. Later rules win, so a
// deeper .gitignore overrides an ancestor's and a negation can bring a file
// back. The rules slice is ordered from the root downwards.
func ignored(rules []ignoreRule, path string, isDir bool) bool {
	ignore := false
	for _, rule := range rules {
		if rule.pattern.dirOnly && !isDir {
			continue
		}
		rel, ok := relativeTo(rule.base, path)
		if !ok {
			continue
		}
		if globPathMatch(rule.pattern, strings.Split(rel, "/")) {
			ignore = !rule.negate
		}
	}
	return ignore
}

// relativeTo returns path relative to base, slash-separated, reporting false
// when path is not inside base.
func relativeTo(base, path string) (string, bool) {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
