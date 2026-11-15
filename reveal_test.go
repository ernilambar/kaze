package main

import (
	"reflect"
	"testing"
)

func TestRevealArgsSelectTheFileOnEachPlatform(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		isDir bool
		goos  string
		want  []string
	}{
		{"darwin file", "/tmp/notes.txt", false, "darwin", []string{"open", "-R", "/tmp/notes.txt"}},
		{"darwin folder", "/tmp", true, "darwin", []string{"open", "/tmp"}},
		{"windows file", `C:\tmp\notes.txt`, false, "windows", []string{"explorer", `/select,C:\tmp\notes.txt`}},
		{"windows folder", `C:\tmp`, true, "windows", []string{"explorer", `C:\tmp`}},
		{"linux file", "/tmp/notes.txt", false, "linux", []string{"xdg-open", "/tmp"}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := revealArgs(test.path, test.isDir, test.goos)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("revealArgs() = %q, want %q", got, test.want)
			}
		})
	}
}
