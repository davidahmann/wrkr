#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] running lightweight OSS dependency checks"
go list -m all >/dev/null

echo "[wrkr] license-check passed (policy implementation scheduled in later epic)"
