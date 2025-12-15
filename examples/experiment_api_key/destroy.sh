#!/usr/bin/env bash
# Run terraform destroy (no rebuild needed)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cd "$SCRIPT_DIR"
export TF_CLI_CONFIG_FILE="$SCRIPT_DIR/terraform.tfrc"

echo "Running terraform destroy..."
terraform destroy "$@"
