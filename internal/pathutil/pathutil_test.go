package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFileKeepsHardLink(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.txt")
	link := filepath.Join(root, "b.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("hard link reads %q, want \"new\"", data)
	}
}

func TestAtomicWriteFileInReadOnlyFolder(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, 0o755) })
	if err := WriteFileAtomic(path, []byte("new")); err != nil {
		t.Fatalf("WriteFileAtomic() error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Fatalf("file reads %q, want \"new\"", data)
	}
}

func TestIsWithinPath(t *testing.T) {
	cases := []struct {
		folder string
		path   string
		want   bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/b", "/a/b/c", true},
		{"/a/b", "/a/bc", false},
		{"/a/b", "/a", false},
	}
	for _, entry := range cases {
		if got := IsWithin(entry.folder, entry.path); got != entry.want {
			t.Errorf("IsWithin(%q, %q) = %v, want %v", entry.folder, entry.path, got, entry.want)
		}
	}
}
