#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] validating schema contracts and model mappings"
go test ./core/schema/... ./core/jcs ./core/sign ./core/errors

echo "[wrkr] validating contract docs presence"
required_docs=(
  "docs/contracts/primitive_contract.md"
  "docs/contracts/output_layout.md"
  "docs/contracts/checkpoint_protocol.md"
  "docs/contracts/jobpack_verify.md"
  "docs/contracts/acceptance_contract.md"
  "docs/contracts/work_item_bridge_contract.md"
  "docs/contracts/serve_api.md"
  "docs/contracts/wrkr_compatible.md"
)

for file in "${required_docs[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "[wrkr] missing required contract doc: $file"
    exit 1
  fi
done
