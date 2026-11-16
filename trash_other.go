//go:build !darwin || !cgo

package main

import (
	"errors"
	"path/filepath"
)

// errTrashUnsupported is what TrashPath reports where there is no Trash to move
// a file to. The tree never deletes anything for good, so the command fails
// loudly instead of quietly removing the file.
var errTrashUnsupported = errors.New("deleting is not supported on this platform")

func trashPath(path string) error {
	return errors.New("move " + filepath.Base(path) + " to the Trash: " + errTrashUnsupported.Error())
}
