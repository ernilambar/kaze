#!/usr/bin/env bash
# Build the app bundle, install it to /Applications and relaunch it.
#
#   ./scripts/dev.sh             debug build
#   ./scripts/dev.sh --release   publishable build via scripts/bundle.sh
#
# Installing the bundle is what makes bundle-only behaviour testable: the Dock
# icon, the app menu and LaunchServices registration come from the bundle, not
# from a bare binary. The copy already in /Applications is killed first, so its
# running shell is lost.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
# shellcheck source=scripts/config.sh
. "$root/scripts/config.sh"

app_name="$(read_info productName)"
version="$(read_info version)"
exe_name="$(bundle_executable)"
if [ -z "$app_name" ] || [ -z "$version" ] || [ -z "$exe_name" ]; then
    printf 'error: build/config.yml must define productName and version, and\n' >&2
    printf '       build/darwin/Info.plist must define CFBundleExecutable\n' >&2
    exit 1
fi

usage() {
    cat <<'EOF'
usage: scripts/dev.sh [--release]

  --release  run scripts/bundle.sh and install its verified bundle
EOF
}

release=false
for arg in "$@"; do
    case "$arg" in
        --release) release=true ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            printf 'unknown argument: %s\n\n' "$arg" >&2
            usage >&2
            exit 2
            ;;
    esac
done

if [ "$(uname -s)" != "Darwin" ]; then
    printf 'error: this app targets macOS only (this host is %s)\n' "$(uname -s)" >&2
    exit 1
fi

# hw.optional.arm64 stays 1 on Apple silicon even under Rosetta, where uname -m
# reports x86_64; checking the hardware keeps an Intel build from being tested
# as if it were a supported target.
if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" != "1" ]; then
    printf 'error: Apple silicon required; this is an Intel Mac\n' >&2
    exit 1
fi

if $release; then
    "$root/scripts/bundle.sh"
    app="$root/dist/$app_name.app"
else
    wails_version="$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)"
    wails3="${WAILS3:-$(command -v wails3 || true)}"
    if [ -z "$wails3" ]; then
        wails3="$(go env GOPATH)/bin/wails3"
    fi
    if [ ! -x "$wails3" ]; then
        printf 'error: wails3 CLI not found; install with:\n' >&2
        printf '  go install github.com/wailsapp/wails/v3/cmd/wails3@%s\n' "$wails_version" >&2
        exit 1
    fi

    printf '==> wails3 package (debug)\n'
    "$wails3" task darwin:package DEV=true
    app="$root/bin/$exe_name.app"
fi

if [ ! -d "$app" ]; then
    printf 'error: expected bundle not found: %s\n' "$app" >&2
    exit 1
fi

dest="/Applications/$app_name.app"
lsregister="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
if pkill -f "$dest/Contents/MacOS/"; then
    while pgrep -f "$dest/Contents/MacOS/" >/dev/null; do
        sleep 0.1
    done
fi

rm -rf "$dest"
ditto "$app" "$dest"
# Finder only notices a replaced bundle once LaunchServices forgets the old one.
"$lsregister" -f "$dest"

printf 'Installed %s (%s) to %s\n' "$app" "$version" "$dest"
open "$dest"
