#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ $# -gt 1 ]]; then
  echo "usage: $0 [path-to-wrkr-binary]" >&2
  exit 2
fi

if [[ $# -eq 1 ]]; then
  if [[ "$1" = /* ]]; then
    BIN_PATH="$1"
  else
    BIN_PATH="$(pwd)/$1"
  fi
else
  BIN_PATH="$(mktemp "${TMPDIR:-/tmp}/wrkr-install-smoke.XXXXXX")"
  go build -o "${BIN_PATH}" ./cmd/wrkr
fi

if [[ ! -x "${BIN_PATH}" ]]; then
  echo "binary is not executable: ${BIN_PATH}" >&2
  exit 2
fi

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "unsupported OS for install smoke: $(uname -s)" >&2
      exit 2
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture for install smoke: $(uname -m)" >&2
      exit 2
      ;;
  esac
}

sha256_file() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return
  fi
  shasum -a 256 "$path" | awk '{print $1}'
}

os="$(detect_os)"
arch="$(detect_arch)"
version="v0.0.0-ci"

work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"; [[ $# -eq 0 ]] && rm -f "${BIN_PATH}"' EXIT

release_dir="${work_dir}/release"
install_dir="${work_dir}/bin"
extract_dir="${work_dir}/extract"
mkdir -p "${release_dir}" "${install_dir}" "${extract_dir}"

asset_name="wrkr_${version}_${os}_${arch}.tar.gz"
cp "${BIN_PATH}" "${extract_dir}/wrkr"
tar -czf "${release_dir}/${asset_name}" -C "${extract_dir}" wrkr
checksum="$(sha256_file "${release_dir}/${asset_name}")"
printf '%s  %s\n' "${checksum}" "${asset_name}" > "${release_dir}/checksums.txt"

echo "==> install script smoke"
WRKR_RELEASE_BASE_URL="file://${release_dir}" \
  bash "${REPO_ROOT}/scripts/install.sh" \
    --version "${version}" \
    --install-dir "${install_dir}"

if [[ ! -x "${install_dir}/wrkr" ]]; then
  echo "installed binary missing: ${install_dir}/wrkr" >&2
  exit 1
fi

"${install_dir}/wrkr" --json demo > "${work_dir}/demo.json"
python3 - <<'PY' "${work_dir}/demo.json"
import json
import sys
from pathlib import Path
payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if not payload.get("job_id"):
    raise SystemExit("missing job_id in demo output")
if not payload.get("jobpack"):
    raise SystemExit("missing jobpack in demo output")
PY

echo "install smoke: pass"
