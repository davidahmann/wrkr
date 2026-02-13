#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] chaos/serve: auth, request-size, traversal guardrails"
./scripts/test_serve_hardening.sh
