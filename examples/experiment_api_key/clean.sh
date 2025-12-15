#!/usr/bin/env bash
# Clean up terraform state and lock files
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cd "$SCRIPT_DIR"

echo "Cleaning up terraform files..."
rm -rf .terraform .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup

echo "✓ Cleaned up"
