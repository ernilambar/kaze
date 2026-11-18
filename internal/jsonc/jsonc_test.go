package jsonc

import "testing"

func TestStripTrailingCommasKeepsStrings(t *testing.T) {
	got := string(StripTrailingCommas([]byte(`{"a": [1, 2, ], "b": "x, ]", "c": "\", }",}`)))
	want := `{"a": [1, 2 ], "b": "x, ]", "c": "\", }"}`
	if got != want {
		t.Fatalf("StripTrailingCommas() = %s, want %s", got, want)
	}
}
