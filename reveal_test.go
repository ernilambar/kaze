package main

import (
	"reflect"
	"testing"
)

func TestRevealArgsSelectTheFileInFinder(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		isDir bool
		want  []string
	}{
		{"file", "/tmp/notes.txt", false, []string{"open", "-R", "/tmp/notes.txt"}},
		{"folder", "/tmp", true, []string{"open", "/tmp"}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := revealArgs(test.path, test.isDir)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("revealArgs() = %q, want %q", got, test.want)
			}
		})
	}
}
