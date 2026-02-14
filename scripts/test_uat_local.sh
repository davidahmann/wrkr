#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="${repo_root}/wrkr-out/uat_local"
release_version="${WRKR_UAT_RELEASE_VERSION:-latest}"
skip_brew="false"
skip_release_installer="false"

usage() {
  cat <<'USAGE'
Run local end-to-end UAT across source, release-installer, and Homebrew install paths.

Usage:
  test_uat_local.sh [--output-dir <path>] [--release-version <tag>] [--skip-brew] [--skip-release-installer]

Options:
  --output-dir <path>         UAT artifacts directory (default: wrkr-out/uat_local)
  --release-version <tag>     GitHub release tag for installer path (default: latest)
  --skip-brew                 Skip Homebrew install path checks
  --skip-release-installer    Skip release installer path checks
  -h, --help                  Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-dir)
      [[ $# -ge 2 ]] || { echo "error: --output-dir requires a value" >&2; exit 2; }
      output_dir="$2"
      shift 2
      ;;
    --release-version)
      [[ $# -ge 2 ]] || { echo "error: --release-version requires a value" >&2; exit 2; }
      release_version="$2"
      shift 2
      ;;
    --skip-brew)
      skip_brew="true"
      shift
      ;;
    --skip-release-installer)
      skip_release_installer="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

mkdir -p "${output_dir}/logs"
summary_md="${output_dir}/summary.md"
summary_json="${output_dir}/summary.json"

status="pass"
results_json="[]"

{
  echo "# Wrkr Local UAT Summary"
  echo
  echo "| Step | Status |"
  echo "| --- | --- |"
} > "$summary_md"

append_result() {
  local step_name="$1"
  local step_status="$2"
  local log_path="$3"

  results_json="$(python3 -c 'import json,sys; data=json.loads(sys.argv[1]); data.append({"step": sys.argv[2], "status": sys.argv[3], "log_path": sys.argv[4]}); print(json.dumps(data, separators=(",",":"), sort_keys=True))' "$results_json" "$step_name" "$step_status" "$log_path")"
  echo "| ${step_name} | ${step_status} |" >> "$summary_md"
}

run_step() {
  local step_name="$1"
  shift
  local log_path="${output_dir}/logs/${step_name}.log"

  if "$@" >"$log_path" 2>&1; then
    append_result "$step_name" "pass" "$log_path"
    return 0
  fi

  status="fail"
  append_result "$step_name" "fail" "$log_path"
  return 1
}

run_version_step() {
  local step_name="$1"
  local bin_path="$2"
  local mode="$3"
  local expected="${4:-}"

  run_step "$step_name" bash -c '
set -euo pipefail
bin="$1"
mode="$2"
expected="$3"

if [[ ! -x "$bin" ]]; then
  echo "binary is not executable: $bin" >&2
  exit 1
fi

payload="$("$bin" --json version)"
python3 - "$mode" "$expected" <<PY <<<"$payload"
import json
import sys

mode = sys.argv[1]
expected = sys.argv[2]
payload = json.loads(sys.stdin.read())
version = payload.get("version", "")

if not version:
    raise SystemExit("version field missing")

if mode == "exact":
    if version != expected:
        raise SystemExit(f"expected version {expected}, got {version}")
elif mode == "non-dev":
    if version == "dev":
        raise SystemExit("expected a release version, got dev")
elif mode == "dev":
    if version != "dev":
        raise SystemExit(f"expected version dev, got {version}")
else:
    raise SystemExit(f"unknown mode: {mode}")

print(version)
PY
' _ "$bin_path" "$mode" "$expected"
}

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is required" >&2
  exit 2
fi
if ! command -v python3 >/dev/null 2>&1; then
  echo "error: python3 is required" >&2
  exit 2
fi
if [[ "$skip_brew" != "true" ]] && ! command -v brew >/dev/null 2>&1; then
  echo "error: brew is required unless --skip-brew is set" >&2
  exit 2
fi

cd "$repo_root"

source_bin="${output_dir}/source/wrkr"
release_bin="${output_dir}/release_install/bin/wrkr"

mkdir -p "$(dirname "$source_bin")"

run_step source_build go build -o "$source_bin" ./cmd/wrkr || true
run_version_step source_version "$source_bin" dev || true
run_step source_adoption_smoke bash "$repo_root/scripts/test_adoption_smoke.sh" "$source_bin" || true
run_step adapter_parity bash "$repo_root/scripts/test_adapter_parity.sh" || true

if [[ "$skip_release_installer" == "true" ]]; then
  append_result "release_installer_path" "skip" "requested --skip-release-installer"
else
  mkdir -p "$(dirname "$release_bin")"
  run_step release_install bash "$repo_root/scripts/install.sh" --version "$release_version" --install-dir "$(dirname "$release_bin")" || true
  if [[ "$release_version" == "latest" ]]; then
    run_version_step release_version "$release_bin" non-dev || true
  else
    run_version_step release_version "$release_bin" exact "${release_version#v}" || true
  fi
  run_step release_adoption_smoke bash "$repo_root/scripts/test_adoption_smoke.sh" "$release_bin" || true
fi

if [[ "$skip_brew" == "true" ]]; then
  append_result "brew_path" "skip" "requested --skip-brew"
else
  run_step brew_tap brew tap davidahmann/tap || true
  run_step brew_update brew update || true
  run_step brew_reinstall bash -c 'if brew list --versions davidahmann/tap/wrkr >/dev/null 2>&1; then brew reinstall davidahmann/tap/wrkr; else brew install davidahmann/tap/wrkr; fi' || true
  run_step brew_test_formula brew test davidahmann/tap/wrkr || true

  brew_prefix="$(brew --prefix)"
  brew_bin="${brew_prefix}/bin/wrkr"
  if [[ "$release_version" == "latest" ]]; then
    run_version_step brew_version "$brew_bin" non-dev || true
  else
    run_version_step brew_version "$brew_bin" exact "${release_version#v}" || true
  fi
  run_step brew_adoption_smoke bash "$repo_root/scripts/test_adoption_smoke.sh" "$brew_bin" || true
fi

python3 - <<PY
import json
from datetime import datetime, timezone
payload = {
  "schema_id": "wrkr.uat_summary",
  "schema_version": "v1",
  "created_at": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
  "status": "${status}",
  "release_version": "${release_version}",
  "results": json.loads('''${results_json}'''),
}
with open("${summary_json}", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, indent=2, sort_keys=True)
    fh.write("\n")
PY

echo "[wrkr] UAT summary markdown: ${summary_md}"
echo "[wrkr] UAT summary json: ${summary_json}"

if [[ "$status" != "pass" ]]; then
  echo "[wrkr] local UAT failed"
  exit 1
fi

echo "[wrkr] local UAT passed"
