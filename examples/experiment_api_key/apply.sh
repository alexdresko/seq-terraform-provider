#!/usr/bin/env bash
# Build provider and run terraform apply
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Build the provider
"$SCRIPT_DIR/build.sh"

# Run terraform apply with the correct config
cd "$SCRIPT_DIR"
export TF_CLI_CONFIG_FILE="$SCRIPT_DIR/terraform.tfrc"

echo ""
echo "Running terraform apply..."
terraform apply "$@"
