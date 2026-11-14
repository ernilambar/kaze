package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed wails.json
var wailsConfig []byte

// appMetadata is the part of wails.json the menu needs. Embedding the file at
// compile time keeps the app menu labels on the same product name as the
// bundle, without a second value to maintain.
type appMetadata struct {
	ProductName string `json:"productName"`
}

// loadAppMetadata reads the embedded wails.json. A malformed file is caught by
// the build's own tooling; this falls back rather than failing at startup.
func loadAppMetadata() appMetadata {
	var config struct {
		Info appMetadata `json:"info"`
	}
	if err := json.Unmarshal(wailsConfig, &config); err != nil || config.Info.ProductName == "" {
		return appMetadata{ProductName: "Kova"}
	}
	return config.Info
}
