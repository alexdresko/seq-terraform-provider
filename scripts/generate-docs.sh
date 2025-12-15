#!/usr/bin/env bash
set -euo pipefail

# Generates provider docs into ./docs using ./templates and .tfplugindocs.hcl.

go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate \
	-provider-dir . \
	-provider-name seq \
	-rendered-provider-name Seq \
	-website-source-dir templates \
	-rendered-website-dir docs
