package find

import (
	"regexp"
	"strings"
	"testing"
)

func TestPreviewLineTrimsIndentationAndCapsLength(t *testing.T) {
	if got := previewLine([]byte("\t  hello")); got != "hello" {
		t.Fatalf("previewLine() = %q, want hello", got)
	}
	long := strings.Repeat("a", maxSearchPreview+50)
	if got := previewLine([]byte(long)); len([]rune(got)) != maxSearchPreview+1 {
		t.Fatalf("previewLine() length = %d, want %d", len([]rune(got)), maxSearchPreview+1)
	}
}

func TestUTF16ColumnCountsSurrogates(t *testing.T) {
	line := []byte("a😀TODO")
	index := strings.Index(string(line), "TODO")
	if index < 0 {
		t.Fatal("test line has no TODO")
	}
	if got := utf16Column(line, index); got != 4 {
		t.Fatalf("utf16Column() = %d, want 4", got)
	}
	if got := utf16Column(line, 0); got != 1 {
		t.Fatalf("utf16Column() at the start = %d, want 1", got)
	}
}

func TestReplaceLinesMatchesJavaScriptTemplates(t *testing.T) {
	cases := []struct {
		name, data, pattern, replacement, want string
	}{
		{"group then text", "fooOld", `foo(Old)`, "$1New", "OldNew"},
		{"whole match", "abc", `b`, "[$&]", "a[b]c"},
		{"dollar", "abc", `b`, "$$", "a$c"},
		{"named group", "key=value", `(?P<k>\w+)=(?P<v>\w+)`, "$<v>=$<k>", "value=key"},
		{"unknown group stays text", "abc", `b`, "$9", "a$9c"},
		{"per line anchors", "  a\n  b\n", `^\s+`, "", "a\nb\n"},
		{"no cross-line match", "a b\nc d;", `a[^;]*`, "X", "X\nc d;"},
		{"crlf kept", "a \r\nb \r\n", `\s+$`, "", "a\r\nb\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := replaceLines([]byte(tc.data), regexp.MustCompile(tc.pattern), tc.replacement, true)
			if string(got) != tc.want {
				t.Fatalf("replaceLines(%q) = %q, want %q", tc.data, got, tc.want)
			}
		})
	}
}
