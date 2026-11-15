# Development

Building Kova from source, running it with hot reload, and cutting releases.

## Prerequisites

- Go 1.26 or newer
- Node.js 20 or newer
- pnpm (the frontend package manager)
- Wails CLI v3:

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH` so `wails3` is found.

## Setup

```sh
go mod download
cd frontend && pnpm install
```

This installs Go modules and the frontend dependencies. pnpm stores packages
in a global content-addressable store and links them into `node_modules`, so
several projects share one copy.

## Development

Run the app with hot reload:

```sh
wails3 dev
```

This starts the Vite dev server and opens the native window. Frontend-only
iteration can also use `cd frontend && pnpm dev`, but the full app requires
`wails3 dev`.

To test the real bundle (Dock icon, app menu, LaunchServices registration),
install it to `/Applications`:

```sh
./scripts/dev.sh             # debug build
./scripts/dev.sh --release   # verified release build via scripts/bundle.sh
```

Installing the bundle is what makes bundle-only behaviour testable; it comes
from the `.app` bundle, not from a bare binary.

## Build

```sh
wails3 task darwin:package
```

The packaged app is written to `bin/kova.app`. `frontend/dist` must exist before
a plain `go build` because `main.go` embeds it.

## Quality gate

Run all of the following from the repository root and confirm each exits 0:

```sh
(cd frontend && pnpm run check)
(cd frontend && pnpm test)
(cd frontend && pnpm run build)
go vet ./...
go test -race ./...
test -z "$(gofmt -l .)"
go build ./...
```

If your change affects app startup, Wails configuration, or platform
packaging, also run `wails3 task darwin:package` and launch the resulting app
from `bin`.

Tests are colocated with the code they cover: Go tests use `testing`
(`app_test.go`) and frontend tests use Vitest (`src/*.test.ts`). TypeScript
tests cover pure helpers only; Svelte components, Monaco, native dialogs, and
generated bindings are verified by manual smoke testing.

## Releasing

Releases are macOS on Apple silicon only; Intel Macs are not supported.

```sh
./scripts/bundle.sh          # quality gate + verified bundle and dist/Kova-macos-arm64.zip
./scripts/release.sh v0.1.0  # tag and push; GitHub Actions publishes the release
```

1. Bump `info.version` in `build/config.yml`, then run
   `wails3 task common:update:build-assets` so the plists match.
2. Commit the bump and merge it to `main`.
3. Run `./scripts/release.sh vX.Y.Z` with the matching tag.

`release.sh` requires a clean tree, verifies the tag against
`build/config.yml`, runs `scripts/bundle.sh` (quality gate plus `wails3 task
darwin:package`), then tags and pushes. The tag triggers
`.github/workflows/release.yml`, which rebuilds on an Apple silicon runner and
attaches `Kova-macos-arm64.zip` and `SHA256SUMS` to the GitHub release.

`bundle.sh` fails on non-macOS hosts and on Intel Macs, and verifies the built
binary is `arm64`. The app is signed ad hoc; if Developer ID signing and
notarization are added later, update the release workflow accordingly.
