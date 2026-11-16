//go:build darwin && cgo

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// TestTrashPathMovesTheFileToTheTrash exercises the real AppKit call, because
// the whole point of the binding is that a deleted file can be put back. The
// file is removed from the Trash again so a test run leaves nothing behind.
// AppKit owns the Trash, and the call is marshalled through the Wails app, so
// this only runs where one exists.
func TestTrashPathMovesTheFileToTheTrash(t *testing.T) {
	if application.Get() == nil {
		t.Skip("the Trash is only reachable from a running application")
	}
	dir := t.TempDir()
	name := fmt.Sprintf("kova-trash-test-%d.txt", time.Now().UnixNano())
	path := writeTempFile(t, dir, name, "hello")

	if err := NewApp().TrashPath(path); err != nil {
		t.Fatalf("TrashPath() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path still exists after TrashPath(): %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot find the home directory: %v", err)
	}
	trashed := filepath.Join(home, ".Trash", name)
	if _, err := os.Stat(trashed); err != nil {
		t.Fatalf("trashed file is not in the Trash: %v", err)
	}
	if err := os.Remove(trashed); err != nil {
		t.Fatalf("clean up the trashed file: %v", err)
	}
}
