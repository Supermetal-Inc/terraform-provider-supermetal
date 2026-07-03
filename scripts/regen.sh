#!/bin/bash
# Regenerate provider code from OpenAPI spec using sibling generator repo
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROVIDER_ROOT="$(dirname "$SCRIPT_DIR")"
GENERATOR_ROOT="${PROVIDER_ROOT}/../terraform-provider-gen"

if [[ ! -d "$GENERATOR_ROOT" ]]; then
    echo "Error: Generator repo not found at $GENERATOR_ROOT"
    echo "Clone it with: git clone git@github.com:supermetal-inc/terraform-provider-gen.git"
    exit 1
fi

cd "$GENERATOR_ROOT"
go run . supermetal_openapi.json --generate "$PROVIDER_ROOT/internal/provider"

echo "Regeneration complete. Run 'go build ./...' to verify."
