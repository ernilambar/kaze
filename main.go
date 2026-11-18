package main

import (
	"embed"

	"kaze/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/config.yml
var buildConfig []byte

func main() {
	app.Run(assets, buildConfig)
}
