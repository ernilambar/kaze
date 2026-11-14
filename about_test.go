package main

import "testing"

func TestLoadAppMetadataReadsWailsConfig(t *testing.T) {
	meta := loadAppMetadata()

	if meta.ProductName == "" {
		t.Fatal("ProductName is empty; embedded wails.json was not parsed")
	}
}
