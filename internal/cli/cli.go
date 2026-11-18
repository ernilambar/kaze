package cli

import (
	"os"
	"path/filepath"
	"strings"

	"kaze/internal/pathutil"
)

// LaunchBuffer is one tab handed to another window. A text buffer carries its
// text, so unsaved edits move with the tab; an image or diff travels by path
// and the receiving window reads it again.
type LaunchBuffer struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// RealPath is Path with symlinks resolved, so the receiving window matches
	// the moved buffer against the tabs it already has.
	RealPath string `json:"real_path"`
	Kind     string `json:"kind"`
	Content  string `json:"content"`
	// SavedContent is the text the tab last had on disk. A moved dirty buffer
	// needs it so the editor knows what "clean" means after the move.
	SavedContent string `json:"saved_content"`
	Modified     int64  `json:"modified"`
	Dirty        bool   `json:"dirty"`
	SourcePath   string `json:"source_path"`
	// BOM, EOL and InvalidUTF8 travel with a text buffer so the receiving
	// window saves it back byte-for-byte the way the first one would.
	BOM         bool   `json:"bom"`
	EOL         string `json:"eol"`
	InvalidUTF8 bool   `json:"invalid_utf8"`
}

// LaunchRequest is what a command line invocation asked the app to open. It travels
// from the invocation that saw the arguments to the running UI, so paths are
// absolute by then and every rejected argument is reported back to the user.
type LaunchRequest struct {
	Files   []string `json:"files"`
	Folders []string `json:"folders"`
	Missing []string `json:"missing"`
	// Buffers carries tabs moved from another window, and ReplaceFolders makes
	// Folders the window's only roots (Open Folder in New Window).
	Buffers        []LaunchBuffer `json:"buffers"`
	ReplaceFolders bool           `json:"replace_folders"`
	// RestoreSession marks the first window of a launch. Only it reopens the
	// saved tabs and writes the session back, so a new window does not clobber
	// the session with its own empty or moved tab list.
	RestoreSession bool `json:"restore_session"`
	// NewWindow asks for a window of its own instead of the focused one. It is
	// consumed by queueLaunch, so a window's frontend never sees it set.
	NewWindow bool `json:"new_window"`
}

// EmptyLaunchRequest is the zero request, with the slices the frontend expects
// as arrays rather than null.
func EmptyLaunchRequest() LaunchRequest {
	return LaunchRequest{
		Files:   []string{},
		Folders: []string{},
		Missing: []string{},
		Buffers: []LaunchBuffer{},
	}
}

// Empty reports whether nothing at all was requested.
func (r LaunchRequest) Empty() bool {
	return len(r.Files) == 0 && len(r.Folders) == 0 && len(r.Missing) == 0 && len(r.Buffers) == 0 && !r.NewWindow
}

// Merge appends another request, keeping command line order. Every folder is
// kept, because each one becomes a root in the sidebar.
func (r *LaunchRequest) Merge(other LaunchRequest) {
	r.Files = append(r.Files, other.Files...)
	r.Folders = append(r.Folders, other.Folders...)
	r.Missing = append(r.Missing, other.Missing...)
	r.Buffers = append(r.Buffers, other.Buffers...)
	r.ReplaceFolders = r.ReplaceFolders || other.ReplaceFolders
	r.RestoreSession = r.RestoreSession || other.RestoreSession
	r.NewWindow = r.NewWindow || other.NewWindow
}

// Normalized replaces nil slices with empty ones and drops repeated folders.
// Wails encodes nil slices as null and the UI reads these fields as arrays; a
// folder named twice is one sidebar root, not two.
func (r LaunchRequest) Normalized() LaunchRequest {
	if r.Files == nil {
		r.Files = []string{}
	}
	if r.Missing == nil {
		r.Missing = []string{}
	}
	if r.Buffers == nil {
		r.Buffers = []LaunchBuffer{}
	}
	r.Folders = pathutil.UniquePaths(r.Folders)
	return r
}

// Action is what a command line asked for before a window exists. Help and
// version must be answered by this process: forwarding them to a running app
// would open a window instead of printing anything.
type Action int

const (
	ActionLaunch Action = iota
	ActionHelp
	ActionVersion
)

// flags maps every accepted spelling to its action. Go's flag convention
// allows a single dash for long names, and Wails itself uses `-flag`.
var flags = map[string]Action{
	"-h":        ActionHelp,
	"-help":     ActionHelp,
	"--help":    ActionHelp,
	"-v":        ActionVersion,
	"-V":        ActionVersion,
	"-version":  ActionVersion,
	"--version": ActionVersion,
}

// ParseAction scans arguments for an informational flag. Help wins over
// version when both are present. `--` ends flag parsing, so a path named
// `--help` after it stays a path.
func ParseAction(args []string) Action {
	action := ActionLaunch
	for _, arg := range args {
		if arg == "--" {
			break
		}
		requested, ok := flags[arg]
		if !ok {
			continue
		}
		if requested == ActionHelp {
			return ActionHelp
		}
		action = ActionVersion
	}
	return action
}

// ParseLaunchRequest turns command line arguments into a LaunchRequest.
// Relative paths resolve against the current directory, flags are ignored, and
// `--` makes everything after it a path. A path that cannot be opened is
// collected in Missing rather than failing the whole launch.
func ParseLaunchRequest(args []string) LaunchRequest {
	request := EmptyLaunchRequest()
	pathsOnly := false

	for _, arg := range args {
		if !pathsOnly {
			if arg == "--" {
				pathsOnly = true
				continue
			}
			if arg == "--new-window" || arg == "-new-window" || arg == "-n" {
				request.NewWindow = true
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
