#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] wrkr-compatible conformance"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
preserve_tmp="${WRKR_KEEP_TMP:-0}"
if [[ "${preserve_tmp}" != "1" ]]; then
  trap 'rm -rf "$tmp_root"' EXIT
fi

runtime_home="$tmp_root/home"
mkdir -p "$runtime_home"
out_dir="$tmp_root/wrkr-out"
mkdir -p "$out_dir"
bin_path="$tmp_root/wrkr"

CGO_ENABLED=0 go build -o "$bin_path" ./cmd/wrkr

accept_cfg="$tmp_root/accept.yaml"
cat > "$accept_cfg" <<'YAML'
schema_id: wrkr.accept_config
schema_version: v1
required_artifacts:
  - reports/demo.md
test_command: "true"
lint_command: "true"
path_rules:
  max_artifact_paths: 0
  forbidden_prefixes: []
  allowed_prefixes: []
YAML

# demo -> export -> verify -> accept -> report
HOME="$runtime_home" "$bin_path" --json demo --out-dir "$out_dir" > "$tmp_root/demo.json"
job_id="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["job_id"])' "$tmp_root/demo.json")"

HOME="$runtime_home" "$bin_path" --json export "$job_id" --out-dir "$out_dir" > "$tmp_root/export.json"
HOME="$runtime_home" "$bin_path" --json verify "$job_id" --out-dir "$out_dir" > "$tmp_root/verify.json"

export GITHUB_STEP_SUMMARY="$tmp_root/step-summary.md"
HOME="$runtime_home" "$bin_path" --json accept run "$job_id" --config "$accept_cfg" --ci --out-dir "$out_dir" > "$tmp_root/accept.json"
HOME="$runtime_home" "$bin_path" --json report github "$job_id" --out-dir "$out_dir" > "$tmp_root/report.json"

# deterministic output layout assertions
[[ -f "$out_dir/jobpacks/jobpack_${job_id}.zip" ]]
[[ -f "$out_dir/reports/github_summary_${job_id}.json" ]]
[[ -f "$out_dir/reports/github_summary_${job_id}.md" ]]
[[ -f "$out_dir/reports/accept_${job_id}.junit.xml" ]]
[[ -f "$GITHUB_STEP_SUMMARY" ]]

# stable acceptance failure exit code + reason codes
fail_cfg="$tmp_root/accept-fail.yaml"
cat > "$fail_cfg" <<'YAML'
schema_id: wrkr.accept_config
schema_version: v1
required_artifacts:
  - missing/never-produced.file
test_command: "true"
lint_command: "true"
path_rules:
  max_artifact_paths: 0
  forbidden_prefixes: []
  allowed_prefixes: []
YAML

set +e
HOME="$runtime_home" "$bin_path" --json accept run "$job_id" --config "$fail_cfg" --out-dir "$out_dir" > "$tmp_root/accept-fail.json"
exit_code=$?
set -e

if [[ "$exit_code" -ne 5 ]]; then
  echo "[wrkr] expected exit code 5 for acceptance failure, got $exit_code"
  exit 1
fi
if ! grep -q 'E_ACCEPT_MISSING_ARTIFACT' "$tmp_root/accept-fail.json"; then
  echo "[wrkr] expected E_ACCEPT_MISSING_ARTIFACT in acceptance failure output"
  cat "$tmp_root/accept-fail.json"
  exit 1
fi

# adapter parity (available v1 adapters)
go test ./core/adapters/wrap ./core/adapters/reference ./core/bridge -count=1

if [[ "${preserve_tmp}" == "1" ]]; then
  persist_dir="${repo_root}/wrkr-out/conformance/wrkr-compatible"
  rm -rf "${persist_dir}"
  mkdir -p "$(dirname "${persist_dir}")"
  cp -R "${tmp_root}" "${persist_dir}"
  echo "[wrkr] preserved conformance artifacts: ${persist_dir}"
fi

echo "[wrkr] conformance ok: job_id=$job_id"
