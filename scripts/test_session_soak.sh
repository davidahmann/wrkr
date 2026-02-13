#!/usr/bin/env bash
set -euo pipefail

iters=${1:-25}

echo "[wrkr] session soak: ${iters} iterations"
for i in $(seq 1 "$iters"); do
  echo "[wrkr] soak iteration ${i}/${iters}"
  go test ./internal/integration -run TestStoreConcurrentAppendNoCorruption -count=1

done

echo "[wrkr] session soak completed"
