#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] validating schema contracts and model mappings"
go test ./core/schema/... ./core/jcs ./core/sign ./core/errors
