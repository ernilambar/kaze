# Kova

A simple text editor built with [Wails](https://wails.io) and
[Monaco Editor](https://microsoft.github.io/monaco-editor/).

## Requirements

- Go 1.26+
- Node.js 20+
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
