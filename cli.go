package main

import (
	"fmt"
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
	Folders []string `json:"folders"`
	Missing []string `json:"missing"`
}

// emptyLaunchRequest is the zero request, with the slices the frontend expects
// as arrays rather than null.
func emptyLaunchRequest() LaunchRequest {
	return LaunchRequest{Files: []string{}, Folders: []string{}, Missing: []string{}}
}

// Empty reports whether nothing at all was requested.
func (r LaunchRequest) Empty() bool {
	return len(r.Files) == 0 && len(r.Folders) == 0 && len(r.Missing) == 0
}

// Merge appends another request, keeping command line order. Every folder is
// kept, because each one becomes a root in the sidebar.
func (r *LaunchRequest) Merge(other LaunchRequest) {
	r.Files = append(r.Files, other.Files...)
	r.Folders = append(r.Folders, other.Folders...)
	r.Missing = append(r.Missing, other.Missing...)
}

// normalized replaces nil slices with empty ones and drops repeated folders.
// Wails encodes nil slices as null and the UI reads these fields as arrays; a
// folder named twice is one sidebar root, not two.
func (r LaunchRequest) normalized() LaunchRequest {
	if r.Files == nil {
		r.Files = []string{}
	}
	if r.Missing == nil {
		r.Missing = []string{}
	}
	r.Folders = uniquePaths(r.Folders)
	return r
}

// uniquePaths keeps the first occurrence of each path, preserving order.
func uniquePaths(paths []string) []string {
	unique := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		unique = append(unique, path)
	}
	return unique
}

// cliAction is what a command line asked for before a window exists. Help and
// version must be answered by this process: forwarding them to a running Kova
// would open a window instead of printing anything.
type cliAction int

const (
	cliLaunch cliAction = iota
	cliActionHelp
	cliActionVersion
)

// cliFlags maps every accepted spelling to its action. Go's flag convention
// allows a single dash for long names, and Wails itself uses `-flag`.
var cliFlags = map[string]cliAction{
	"-h":        cliActionHelp,
	"-help":     cliActionHelp,
	"--help":    cliActionHelp,
	"-v":        cliActionVersion,
	"-V":        cliActionVersion,
	"-version":  cliActionVersion,
	"--version": cliActionVersion,
}

// parseCLIAction scans arguments for an informational flag. Help wins over
// version when both are present. `--` ends flag parsing, so a path named
// `--help` after it stays a path.
func parseCLIAction(args []string) cliAction {
	action := cliLaunch
	for _, arg := range args {
		if arg == "--" {
			break
		}
		requested, ok := cliFlags[arg]
		if !ok {
			continue
		}
		if requested == cliActionHelp {
			return cliActionHelp
		}
		action = cliActionVersion
	}
	return action
}

// cliUsageText is the help screen. It lists every spelling so the accepted flags
// are discoverable from the terminal alone.
func cliUsageText(meta appMetadata) string {
	return fmt.Sprintf(`%s %s — a minimal text editor.

Usage:
  %s [path ...]

Arguments:
  path  A file to open, or a folder to add to the sidebar.

Options:
  -h, --help     Show this help and exit.
  -v, --version  Show the version and exit.
  --             Treat every following argument as a path.

Files open in the editor. Folders become sidebar roots and are remembered for
next time. Running %s with paths while Kova is already open hands them to that
window instead of starting a second one.
`, meta.ProductName, meta.ProductVersion, commandName, commandName)
}

// cliVersionText is the one-line answer to -v/--version.
func cliVersionText(meta appMetadata) string {
	return fmt.Sprintf("%s %s\n", commandName, meta.ProductVersion)
}

// parseLaunchRequest turns command line arguments into a LaunchRequest.
// Relative paths resolve against the current directory, flags are ignored, and
// `--` makes everything after it a path. A path that cannot be opened is
// collected in Missing rather than failing the whole launch.
func parseLaunchRequest(args []string) LaunchRequest {
	request := emptyLaunchRequest()
	pathsOnly := false

	for _, arg := range args {
		if !pathsOnly {
			if arg == "--" {
				pathsOnly = true
				continue
			}
			if strings.HasPrefix(arg, "-") {
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
		r.Folders = append(r.Folders, path)
	default:
		r.Files = append(r.Files, path)
	}
}
