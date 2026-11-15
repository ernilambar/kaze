#!/usr/bin/env bash
# Helpers for reading the v3 build config from shell scripts.
#
# Called by scripts/dev.sh and scripts/bundle.sh (and sourced by CI), so the
# product name and version exist in exactly one place: build/config.yml, which
# also feeds the bundle's Info.plist. about.go embeds the same file for the app
# menu and `kova --version`.

# read_info prints one scalar from the top-level `info:` block in build/config.yml.
# The file is YAML; only these scalars are read outside the wails3 CLI, so awk
# keeps the scripts free of a YAML dependency.
read_info() {
    local key="$1" file="${2:-build/config.yml}"
    awk -v key="$key" '
        /^info:/ { in_info = 1; next }
        in_info && /^[^ ]/ { exit }
        in_info && $0 ~ "^  " key ":" {
            line = $0
            sub(/^[^:]*:[ ]*/, "", line)
            sub(/[ ]*#.*$/, "", line)
            gsub(/"/, "", line)
            print line
            exit
        }
    ' "$file"
}

# bundle_executable prints the Mach-O name inside the .app bundle, which is also
# the bundle's file name (kova.app).
bundle_executable() {
    plutil -extract CFBundleExecutable raw -o - "${1:-build/darwin/Info.plist}"
}
