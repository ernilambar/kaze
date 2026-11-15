//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

void nativePrintText(const char *text, const char *fontName, double fontSize);
int nativePrintImage(const char *path);
*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// printText opens the print panel for text laid out as a monospaced document.
// AppKit owns the panel, so the call is marshalled onto the main thread; the
// panel itself runs as a sheet, which keeps this call from blocking.
func printText(text, fontFamily string) error {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	cFont := C.CString(fontFamily)
	defer C.free(unsafe.Pointer(cFont))

	application.InvokeSync(func() {
		C.nativePrintText(cText, cFont, C.double(defaultFontSize))
	})
	return nil
}

// printImage opens the print panel for a picture, scaled to one page.
func printImage(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var printed C.int
	application.InvokeSync(func() {
		printed = C.nativePrintImage(cPath)
	})
	if printed == 0 {
		return errors.New("cannot read the image to print")
	}
	return nil
}
