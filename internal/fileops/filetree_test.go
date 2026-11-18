package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUniqueCopyPathDotfile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := UniqueCopyPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != ".env copy" {
		t.Fatalf("UniqueCopyPath(.env) = %q, want \".env copy\"", filepath.Base(got))
	}
}
