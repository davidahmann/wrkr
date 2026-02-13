#!/usr/bin/env bash
set -euo pipefail

ran_one=0

can_run() {
  local cmd="$1"
  "$cmd" --help >/dev/null 2>&1
}

if command -v gosec >/dev/null 2>&1 && can_run gosec; then
  echo "[wrkr] running gosec"
  gosec ./...
  ran_one=1
else
  echo "[wrkr] gosec not runnable, skipping"
fi

if command -v govulncheck >/dev/null 2>&1 && can_run govulncheck; then
  echo "[wrkr] running govulncheck"
  govulncheck ./...
  ran_one=1
else
  echo "[wrkr] govulncheck not runnable, skipping"
fi

if [[ "$ran_one" -eq 0 ]]; then
  echo "[wrkr] no SAST tools installed; running go vet fallback"
  go vet ./...
fi
