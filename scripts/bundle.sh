#!/usr/bin/env bash
# Run the quality gate and build the publishable macOS bundle.
#
#   ./scripts/bundle.sh
#
# Produces the same artifacts .github/workflows/release.yml attaches to a
# release: the .app bundle, its -macos-arm64.zip archive and SHA256SUMS, all
# under dist/. Only darwin/arm64 is supported: no Intel build, no universal
# binary.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
# shellcheck source=scripts/config.sh
. "$root/scripts/config.sh"

# Compile and link against the deployment target declared in the bundle's
# Info.plist. Without it, cgo compiles against the host's macOS version while
# the linker targets an older one, and every object file warns about it.
export MACOSX_DEPLOYMENT_TARGET=15.0
export CGO_CFLAGS="-mmacosx-version-min=15.0"
export CGO_LDFLAGS="-mmacosx-version-min=15.0"

app_name="$(read_info productName)"
version="$(read_info version)"
exe_name="$(bundle_executable)"

if [ -z "$app_name" ]; then
    printf 'error: build/config.yml has no info.productName\n' >&2
    exit 1
fi
if [ -z "$version" ]; then
    printf 'error: build/config.yml has no info.version\n' >&2
    exit 1
fi
if [ -z "$exe_name" ]; then
    printf 'error: build/darwin/Info.plist has no CFBundleExecutable\n' >&2
    exit 1
fi

if [ "$(uname -s)" != "Darwin" ]; then
    printf 'error: this app targets macOS only (this host is %s)\n' "$(uname -s)" >&2
    exit 1
fi

# hw.optional.arm64 stays 1 on Apple silicon even under Rosetta, where uname -m
# reports x86_64; checking the hardware keeps an Intel build from passing as
# the supported target.
if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" != "1" ]; then
    printf 'error: Apple silicon required; this is an Intel Mac\n' >&2
    exit 1
fi

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

printf '==> frontend\n'
(
    cd frontend
    pnpm install --frozen-lockfile
    # main.go embeds frontend/dist, and binding generation compiles the main
    # package, so the frontend must exist before bindings are regenerated.
    pnpm run build
)

printf '==> wails bindings\n'
bindings_state() {
    git status --porcelain -- frontend/bindings
    git diff -- frontend/bindings
}
bindings_before="$(bindings_state)"
"$wails3" generate bindings -ts -i
bindings_after="$(bindings_state)"
if [ "$bindings_before" != "$bindings_after" ]; then
    printf 'error: wails bindings changed during generation; commit them and retry:\n' >&2
    git status --short -- frontend/bindings >&2
    exit 1
fi

printf '==> frontend typecheck\n'
(
    cd frontend
    pnpm run check
)

printf '==> frontend tests\n'
(
    cd frontend
    pnpm test
)

printf '==> gofmt\n'
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
    printf 'error: not gofmt-formatted:\n%s\n' "$unformatted" >&2
    exit 1
fi

printf '==> go vet\n'
go vet ./...

printf '==> go test\n'
go test -race ./...

printf '==> wails3 package (release)\n'
rm -rf dist
mkdir -p dist
# The production build task compiles with -tags production -trimpath and strips
# the binary, then bundles it with the generated Info.plist.
"$wails3" task darwin:package

built_app="$root/bin/$exe_name.app"
if [ ! -d "$built_app" ]; then
    printf 'error: expected bundle not found: %s\n' "$built_app" >&2
    exit 1
fi

# The build task regenerates bindings; a change here means committed bindings
# are stale, and the archive would ship a frontend built against them.
if [ "$(bindings_state)" != "$bindings_before" ]; then
    printf 'error: wails bindings changed during the release build; commit them and retry:\n' >&2
    git status --short -- frontend/bindings >&2
    exit 1
fi

dist_app="$root/dist/$app_name.app"
ditto "$built_app" "$dist_app"

app_arch="$(lipo -archs "$dist_app/Contents/MacOS/$exe_name")"
if [ "$app_arch" != "arm64" ]; then
    printf 'error: expected arm64 app binary, got: %s\n' "$app_arch" >&2
    exit 1
fi

printf '==> package\n'
zip="$root/dist/$app_name-macos-arm64.zip"
# --norsrc --noextattr keeps AppleDouble ._* entries out of the archive; the
# signature lives in the Mach-O and CodeResources, not in an xattr.
ditto -c -k --norsrc --noextattr --keepParent "$dist_app" "$zip"
(cd "$root/dist" && shasum -a 256 "$app_name-macos-arm64.zip" >SHA256SUMS)

printf '\nbundle:  %s\narchive: %s\ndigest:  %s\nversion: %s\n' \
    "$dist_app" "$zip" "$(cat "$root/dist/SHA256SUMS")" "$version"
