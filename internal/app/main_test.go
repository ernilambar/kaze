package app

import (
	"os"
	"testing"
)

// testBuildConfig is build/config.yml, which package main embeds and passes to
// Run in the real app.
var testBuildConfig []byte

func TestMain(m *testing.M) {
	b, err := os.ReadFile("../../build/config.yml")
	if err != nil {
		panic(err)
	}
	testBuildConfig = b
	appMeta = loadAppMetadata(b)
	os.Exit(m.Run())
}
