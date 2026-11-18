package app

import (
	"regexp"
	"strings"
	"testing"
)

var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func TestLoadAppMetadataReadsBuildConfig(t *testing.T) {
	meta := loadAppMetadata(testBuildConfig)

	// The embedded build/config.yml carries trailing comments, so the values
	// must stop at the closing quote or the start of the comment.
	if meta.ProductName == "" || strings.ContainsAny(meta.ProductName, "#\"'") {
		t.Fatalf("ProductName = %q, want a clean scalar", meta.ProductName)
	}
	if meta.ProductIdentifier == "" || strings.ContainsAny(meta.ProductIdentifier, "#\"' ") {
		t.Fatalf("ProductIdentifier = %q, want a clean scalar", meta.ProductIdentifier)
	}
	if !versionPattern.MatchString(meta.ProductVersion) {
		t.Fatalf("ProductVersion = %q, want a value like 0.1.0", meta.ProductVersion)
	}
}

func TestYAMLScalarStripsQuotesAndComments(t *testing.T) {
	tests := map[string]string{
		`"0.1.0" # The application version`: "0.1.0",
		`'Name'`:                            "Name",
		`bare # comment`:                    "bare",
		`bare`:                              "bare",
		``:                                  "",
	}
	for input, want := range tests {
		if got := yamlScalar(input); got != want {
			t.Errorf("yamlScalar(%q) = %q, want %q", input, got, want)
		}
	}
}
