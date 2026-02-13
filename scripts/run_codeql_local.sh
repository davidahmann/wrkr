#!/usr/bin/env bash
set -euo pipefail

if ! command -v codeql >/dev/null 2>&1; then
  echo "[wrkr] codeql CLI not installed; skipping local deep scan"
  exit 0
fi

echo "[wrkr] codeql detected; local wrapper is intentionally lightweight in Epic 0"
echo "[wrkr] run codeql database/analyze commands here once workflow inputs are finalized"
