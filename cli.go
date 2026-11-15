package main

import (
	"os"
	"path/filepath"
	"strings"
)

// launchEvent is the Wails event the frontend listens on to pick up paths
// passed to a Kova that is already running.
const launchEvent = "kova:launch"

// LaunchRequest is what a `kova` command line asked the app to open. It travels
// from the invocation that saw the arguments to the running UI, so paths are
// absolute by then and every rejected argument is reported back to the user.
type LaunchRequest struct {
	Files   []string `json:"files"`
	Folder  string   `json:"folder"`
	Missing []string `json:"missing"`
}

// emptyLaunchRequest is the zero request, with the slices the frontend expects
// as arrays rather than null.
func emptyLaunchRequest() LaunchRequest {
	return LaunchRequest{Files: []string{}, Missing: []string{}}
}

// Empty reports whether nothing at all was requested.
func (r LaunchRequest) Empty() bool {
	return len(r.Files) == 0 && r.Folder == "" && len(r.Missing) == 0
}

// Merge appends another request, keeping command line order. A later folder
// wins, because the last directory named is the one the user meant.
func (r *LaunchRequest) Merge(other LaunchRequest) {
	r.Files = append(r.Files, other.Files...)
	r.Missing = append(r.Missing, other.Missing...)
	if other.Folder != "" {
		r.Folder = other.Folder
	}
}

// normalized replaces nil slices with empty ones. Wails encodes nil slices as
// null, and the UI reads these fields as arrays.
func (r LaunchRequest) normalized() LaunchRequest {
	if r.Files == nil {
		r.Files = []string{}
	}
	if r.Missing == nil {
		r.Missing = []string{}
	}
	return r
}

// wailsValueFlags are the flags the Wails runtime parses out of os.Args itself.
// Their values are skipped too, so `wails dev` does not look like a request to
// open the asset directory.
var wailsValueFlags = map[string]bool{
	"-assetdir":              true,
	"--assetdir":             true,
	"-devserver":             true,
	"--devserver":            true,
	"-frontenddevserverurl":  true,
	"--frontenddevserverurl": true,
	"-loglevel":              true,
	"--loglevel":             true,
	"-appargs":               true,
	"--appargs":              true,
}

// parseLaunchRequest turns command line arguments into a LaunchRequest.
// Relative paths resolve against the current directory, flags are ignored, and
// `--` makes everything after it a path. A path that cannot be opened is
// collected in Missing rather than failing the whole launch.
func parseLaunchRequest(args []string) LaunchRequest {
	request := emptyLaunchRequest()
	pathsOnly := false
	skipValue := false

	for _, arg := range args {
		if skipValue {
			skipValue = false
			continue
		}
		if !pathsOnly {
			if arg == "--" {
				pathsOnly = true
				continue
			}
			if strings.HasPrefix(arg, "-") {
				name, _, hasValue := strings.Cut(arg, "=")
				skipValue = !hasValue && wailsValueFlags[name]
				continue
			}
		}

		request.addPath(arg)
	}

	return request
}

// addPath classifies one argument as a file or a folder and appends it.
func (r *LaunchRequest) addPath(arg string) {
	path, err := filepath.Abs(arg)
	if err != nil {
		r.Missing = append(r.Missing, arg)
		return
	}

	info, err := os.Stat(path)
	switch {
	case err != nil:
		r.Missing = append(r.Missing, arg)
	case info.IsDir():
		r.Folder = path
	default:
		r.Files = append(r.Files, path)
	}
}
