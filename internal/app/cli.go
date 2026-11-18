package app

import "fmt"

// launchEvent is the Wails event the frontend listens on to pick up paths
// passed to an instance that is already running.
const launchEvent = "app:launch"

// cliUsageText is the help screen. It lists every spelling so the accepted flags
// are discoverable from the terminal alone.
func cliUsageText(meta appMetadata) string {
	return fmt.Sprintf(`%s %s — a lightweight macOS editor for code and plain text.

Usage:
  %s [path ...]

Arguments:
  path  A file to open, or a folder to add to the sidebar.

Options:
  -h, --help        Show this help and exit.
  -v, --version     Show the version and exit.
  -n, --new-window  Open a new window instead of the focused one.
  --                Treat every following argument as a path.

Files open in the editor. Folders become sidebar roots and are remembered for
next time. Running %s with paths while %s is already open hands them to that
window instead of starting a second one; --new-window opens a window of its own
for them instead.
`, meta.ProductName, meta.ProductVersion, commandName, commandName, meta.ProductName)
}

// cliVersionText is the one-line answer to -v/--version.
func cliVersionText(meta appMetadata) string {
	return fmt.Sprintf("%s %s\n", commandName, meta.ProductVersion)
}
