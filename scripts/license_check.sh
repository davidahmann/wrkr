#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${repo_root}/wrkr-out/reports"
mkdir -p "${out_dir}"

echo "[wrkr] collecting dependency inventory"
deps_file="${out_dir}/license_inventory_go.txt"
go list -m all | LC_ALL=C sort > "${deps_file}"

if grep -E '\.(local|internal)$' "${deps_file}" >/dev/null 2>&1; then
  echo "[wrkr] unexpected local/internal modules found in dependency inventory"
  exit 1
fi

echo "[wrkr] license inventory written: ${deps_file}"
