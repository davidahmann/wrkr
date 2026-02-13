#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] validating enterprise-consumer contract surfaces"
go test ./core/schema/... ./core/errors ./core/jcs -count=1

echo "[wrkr] validating SDK consumer surface"
if command -v uv >/dev/null 2>&1; then
  (
    cd sdk/python
    uv run --python 3.13 --extra dev pytest -q tests/test_import.py tests/test_cli.py
  )
elif command -v python3 >/dev/null 2>&1 && python3 -c 'import pytest' >/dev/null 2>&1; then
  (
    cd sdk/python
    python3 -m pytest -q tests/test_import.py tests/test_cli.py
  )
else
  echo "[wrkr] uv/pytest unavailable; skipping python consumer checks"
fi
