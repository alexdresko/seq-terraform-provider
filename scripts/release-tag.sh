#!/usr/bin/env bash
set -euo pipefail

if [[ ${1:-} == "" ]]; then
  echo "Usage: $0 <version>" >&2
  echo "Examples: $0 0.1.0   OR   $0 v0.1.0" >&2
  exit 2
fi

version="$1"
if [[ "$version" != v* ]]; then
  version="v${version}"
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Working tree not clean. Commit or stash changes before tagging." >&2
  exit 1
fi

if git rev-parse "$version" >/dev/null 2>&1; then
  echo "Tag already exists: $version" >&2
  exit 1
fi

echo "Running tests..."
go test ./...

echo "Generating docs..."
"$(dirname "$0")/generate-docs.sh"

if ! git diff --quiet docs; then
  echo "Docs changed. Commit updated docs before tagging." >&2
  git --no-pager diff -- docs >&2 || true
  exit 1
fi

echo "Creating annotated tag: $version"
git tag -a "$version" -m "Release $version"

echo "Tag created. Push it to trigger GitHub Actions release:"
echo "  git push origin $version"
