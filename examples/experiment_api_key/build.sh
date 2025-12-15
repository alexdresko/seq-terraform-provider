#!/usr/bin/env bash
# Build the provider binary from the repo root
set -e

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo "Building provider..."
cd "$REPO_ROOT"
go build -o bin/terraform-provider-seq .

echo "✓ Provider built: $REPO_ROOT/bin/terraform-provider-seq"
