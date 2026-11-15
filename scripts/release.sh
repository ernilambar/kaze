#!/usr/bin/env bash
# Tag and push a release after building and verifying the macOS bundle.
#
#   ./scripts/release.sh vX.Y.Z
#
# The tag must match info.version in build/config.yml; GitHub Actions then
# publishes the artifacts built by scripts/bundle.sh.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/config.sh
. scripts/config.sh

version="$(read_info version)"
if [ -z "$version" ]; then
    echo "error: build/config.yml has no info.version" >&2
    exit 1
fi

tag="${1:-}"
if [ -z "$tag" ]; then
    echo "usage: scripts/release.sh vX.Y.Z (current version: v$version)" >&2
    exit 1
fi
if [[ "$tag" != v* ]]; then
    tag="v$tag"
fi

expected="v$version"
if [[ "$tag" != "$expected" ]]; then
    echo "error: $tag does not match build/config.yml info.version $version (expected $expected)" >&2
    exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
    echo "error: working tree is not clean" >&2
    exit 1
fi

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
    echo "error: tag $tag already exists" >&2
    exit 1
fi

scripts/bundle.sh

# The quality gate may regenerate bindings. Tag only committed sources.
if [[ -n "$(git status --porcelain)" ]]; then
    echo "error: the build changed tracked files; commit them and retry" >&2
    exit 1
fi

git tag -a "$tag" -m "$tag"
git push origin "$tag"
echo "pushed $tag; GitHub Actions will publish the release"
