//go:build darwin && cgo

package app

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

int nativeTrashPath(const char *path, char **failure);
*/
import "C"

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// trashPath moves path to the Trash. AppKit does it, so the file really lands in
// the user's Trash and can be put back; the call is marshalled onto the main
// thread because the file manager is part of the app's UI.
func trashPath(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var failure *C.char
	var moved C.int
	application.InvokeSync(func() {
		moved = C.nativeTrashPath(cPath, &failure)
	})
	if moved != 0 {
		return nil
	}
	message := "the file manager refused the request"
	if failure != nil {
		message = C.GoString(failure)
		C.free(unsafe.Pointer(failure))
	}
	return fmt.Errorf("move %s to the Trash: %s", filepath.Base(path), message)
}
