#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] validating schema contracts and model mappings"
go test ./core/schema/... ./core/jcs ./core/sign ./core/errors -count=1

echo "[wrkr] validating contract docs presence"
required_docs=(
  "docs/contracts/primitive_contract.md"
  "docs/contracts/output_layout.md"
  "docs/contracts/checkpoint_protocol.md"
  "docs/contracts/lease_heartbeat.md"
  "docs/contracts/environment_fingerprint.md"
  "docs/contracts/jobpack_verify.md"
  "docs/contracts/acceptance_contract.md"
  "docs/contracts/ticket_footer_conformance.md"
  "docs/contracts/github_summary_conformance.md"
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

echo "[wrkr] validating consumer compatibility contract checks"
./scripts/test_ent_consumer_contract.sh
