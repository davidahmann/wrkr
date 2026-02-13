#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${repo_root}/wrkr-out/reports"
db_dir="${repo_root}/.codeql-db"
sarif_path="${out_dir}/codeql-local.sarif"

if ! command -v codeql >/dev/null 2>&1; then
  echo "[wrkr] codeql CLI not installed; skipping local deep scan"
  exit 0
fi

echo "[wrkr] running local codeql analysis"
mkdir -p "${out_dir}"
rm -rf "${db_dir}"

codeql database create "${db_dir}" \
  --language=go \
  --source-root "${repo_root}" \
  --command "cd ${repo_root} && go build ./cmd/wrkr"

codeql database analyze "${db_dir}" \
  --format=sarifv2.1.0 \
  --output "${sarif_path}" \
  --sarif-category=codeql/go \
  --threads=0 \
  codeql/go-queries:codeql-suites/go-security-and-quality.qls

echo "[wrkr] codeql sarif: ${sarif_path}"
