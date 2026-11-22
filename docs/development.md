# Development

## Requirements

macOS on Apple silicon, Go 1.26+, Node.js 20+, pnpm, and the Wails v3 CLI:

```sh
go install "github.com/wailsapp/wails/v3/cmd/wails3@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)"
```

Put `$(go env GOPATH)/bin` on `PATH`.

## Setup and run

```sh
go mod download
(cd frontend && pnpm install)
wails3 dev                  # native window with hot reload
./scripts/dev.sh            # debug build to /Applications (Dock, menus)
./scripts/dev.sh --release  # verified release build to /Applications
wails3 task darwin:package  # -> bin/kaze.app
```

`main.go` embeds `frontend/dist`, so build the frontend before a plain
`go build`. Never edit `frontend/bindings/`; add exported `App` methods and run
`wails3 generate bindings -ts -i`.

## Quality gate

Every command must exit 0:

```sh
export MACOSX_DEPLOYMENT_TARGET=15.0 CGO_CFLAGS="-mmacosx-version-min=15.0" CGO_LDFLAGS="-mmacosx-version-min=15.0"
(cd frontend && pnpm run check)
(cd frontend && pnpm test)
(cd frontend && pnpm run build)
go vet ./...
go test -race ./...
test -z "$(gofmt -l .)"
go build ./...
```

Keep the deployment target in sync with `LSMinimumSystemVersion` in both
plists. Startup, Wails or packaging changes also need
`wails3 task darwin:package` and a launch from `bin`.

## Releasing

1. Bump `info.version` in `build/config.yml`.
2. Run `wails3 task common:update:build-assets`, commit, and merge to `main`.
3. Run `./scripts/release.sh vX.Y.Z`.

`release.sh` needs a clean tree and a tag matching `build/config.yml`; it runs
`bundle.sh` (gate + arm64 zip + checksums), then tags and pushes. CI
(`.github/workflows/release.yml`) builds and publishes. Signing is ad hoc.
