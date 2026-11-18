package app

import "strings"

// appMetadata is the part of the build config runtime code needs. Embedding the
// file at compile time keeps the app menu labels, the config directory and
// `--version` on the same values as the bundle, without a second value to
// maintain.
type appMetadata struct {
	ProductName       string
	ProductIdentifier string
	ProductVersion    string
}

// appMeta is set once by Run; the build config cannot change while the app
// runs.
var appMeta appMetadata

// loadAppMetadata reads productName, productIdentifier and version from the embedded v3 build
// config. The file is YAML and the two scalars we need sit in the top-level
// `info:` block, so they are read directly instead of pulling in a YAML parser
// for three strings. A malformed file falls back rather than failing at startup.
func loadAppMetadata(buildConfig []byte) appMetadata {
	meta := appMetadata{ProductVersion: "unknown"}

	inInfo := false
	for _, line := range strings.Split(string(buildConfig), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			inInfo = line == "info:"
			continue
		}
		if !inInfo {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		value = yamlScalar(value)
		switch key {
		case "productName":
			meta.ProductName = value
		case "productIdentifier":
			meta.ProductIdentifier = value
		case "version":
			if value != "" {
				meta.ProductVersion = value
			}
		}
	}
	return meta
}

// yamlScalar reads one scalar from "value # comment" YAML, keeping a quoted
// value intact when the comment follows it.
func yamlScalar(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if quote := value[0]; quote == '"' || quote == '\'' {
		if end := strings.IndexByte(value[1:], quote); end >= 0 {
			return value[1 : end+1]
		}
		return ""
	}
	before, _, _ := strings.Cut(value, " #")
	return strings.TrimSpace(before)
}
