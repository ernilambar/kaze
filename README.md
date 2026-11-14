# Kova

A simple text editor built with [Wails](https://wails.io) and
[Monaco Editor](https://microsoft.github.io/monaco-editor/).

## Requirements

- Go 1.26+
- Node.js 20+
- pnpm
- Wails CLI v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

## Development

```sh
wails dev
```

This starts the Vite dev server and opens the app with hot reload.

## Build

```sh
wails build
```

The packaged app is written to `build/bin`.

## Settings

Kova reads editor settings from `settings.json` in the platform config
directory and creates it with defaults on first launch. See
[docs/settings.md](docs/settings.md) for the file locations, supported fields,
and validation rules.

## Releases

```sh
./scripts/dev.sh             # debug build, install to /Applications, launch
./scripts/dev.sh --release   # same, with the verified release bundle
./scripts/bundle.sh          # verify + build dist/Kova-macos-arm64.zip
./scripts/release.sh v0.1.0  # tag and push; GitHub Actions publishes
```

Releases support macOS on Apple silicon only. Bump `info.productVersion` in
`wails.json` before tagging.

## License

[MIT](LICENSE) © [Nilambar Sharma](https://www.nilambar.net)
