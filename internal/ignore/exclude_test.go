package ignore

import "testing"

func TestExcludeMatcherHidesNamesAtAnyDepth(t *testing.T) {
	matcher := NewMatcher([]string{"node_modules", "*.log", ".DS_Store"})

	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"node_modules", true, true},
		{"packages/app/node_modules", true, true},
		{"debug.log", false, true},
		{"logs/today/debug.log", false, true},
		{".DS_Store", false, true},
		{"src/main.go", false, false},
		{"node_modules.txt", false, false},
	}
	for _, test := range cases {
		if got := matcher.Matches(test.path, test.isDir); got != test.want {
			t.Errorf("matches(%q, %v) = %v, want %v", test.path, test.isDir, got, test.want)
		}
	}
}

func TestExcludeMatcherAnchorsPatternsWithSlash(t *testing.T) {
	matcher := NewMatcher([]string{"build/output", "**/generated", "dist/"})

	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"build/output", true, true},
		{"build/output", false, true},
		{"other/build/output", false, false},
		{"src/generated", true, true},
		{"generated", true, true},
		{"dist", true, true},
		{"dist", false, false},
		{"src/dist", true, true},
	}
	for _, test := range cases {
		if got := matcher.Matches(test.path, test.isDir); got != test.want {
			t.Errorf("matches(%q, %v) = %v, want %v", test.path, test.isDir, got, test.want)
		}
	}
}

func TestExcludeMatcherIgnoresBlankPatterns(t *testing.T) {
	matcher := NewMatcher([]string{"", "   ", "/"})
	if !matcher.Empty() {
		t.Fatalf("empty patterns produced %d rules", len(matcher.patterns))
	}
	if matcher.Matches("src/main.go", false) {
		t.Fatal("empty matcher hid a file")
	}
}
