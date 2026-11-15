//go:build !darwin || !cgo

package main

import "errors"

// errNoPrinting is what the editor reports where the native print panel is not
// available, so the command fails loudly instead of doing nothing.
var errNoPrinting = errors.New("printing is not supported on this platform")

func printText(string, string) error {
	return errNoPrinting
}

func printImage(string) error {
	return errNoPrinting
}
