# Development

Building Kaze from source, running it, and cutting releases.

## Requirements

- macOS on Apple silicon
- Go 1.26+
- Node.js 20+
- pnpm
- Wails v3 CLI:

```sh
go install "github.com/wailsapp/wails/v3/cmd/wails3@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)"
```

Ensure `$(go env GOPATH)/bin` is on `PATH`.

## Setup

```sh
go mod download
cd frontend && pnpm install
```

## Run

```sh
wails3 dev                  # native window with hot reload
(cd frontend && pnpm dev)   # frontend only, no native shell
```

To test bundle-only behaviour (Dock icon, app menu, LaunchServices), install
the app to `/Applications`:

```sh
./scripts/dev.sh            # debug build
./scripts/dev.sh --release  # bundle.sh + verified release build
```

## Build

```sh
wails3 task darwin:package  # -> bin/kaze.app
```

`main.go` embeds `frontend/dist`, so a plain `go build` needs the frontend
built first. Generated bindings live in `frontend/bindings/` and must not be
edited by hand; `wails3 dev`, packaging, and
`wails3 generate bindings -ts -i` regenerate them. Add exported `App` methods
instead.

## Quality gate

Run from the repository root; every command must exit 0:

```sh
export MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS="-mmacosx-version-min=13.0" CGO_LDFLAGS="-mmacosx-version-min=13.0"
(cd frontend && pnpm run check)
(cd frontend && pnpm test)
(cd frontend && pnpm run build)
go vet ./...
go test -race ./...
test -z "$(gofmt -l .)"
go build ./...
```

The exports keep cgo objects at the deployment target declared in
`build/darwin/Info.plist`. Keep that value in sync with `LSMinimumSystemVersion`
in both plists. Changes to startup, Wails configuration, or packaging also
need `wails3 task darwin:package` to succeed and the resulting app launched
from `bin`.

Tests are colocated with the code: Go's `testing` and Vitest. Vitest covers
pure TypeScript helpers; Svelte components, Monaco, dialogs, and bindings are
verified by manual smoke testing.

## Releasing

Releases are macOS arm64 only.

```sh
./scripts/bundle.sh          # gate + dist/Kaze-macos-arm64.zip + SHA256SUMS
./scripts/release.sh v0.1.0  # tag and push; CI publishes
```

1. Bump `info.version` in `build/config.yml`.
2. Run `wails3 task common:update:build-assets`, commit, and merge to `main`.
3. Run `./scripts/release.sh vX.Y.Z` with the matching tag.

`release.sh` requires a clean tree, checks the tag against `build/config.yml`,
runs `bundle.sh`, then tags and pushes. The tag starts
`.github/workflows/release.yml`, which rebuilds on an Apple silicon runner and
attaches the zip and checksums. `bundle.sh` also checks the bundle's binary is
arm64. Signing is ad hoc; adding Developer ID signing or notarization means
updating the release workflow.
