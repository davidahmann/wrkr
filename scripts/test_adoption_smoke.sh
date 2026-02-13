#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] adoption smoke"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
preserve_tmp="${WRKR_KEEP_TMP:-0}"
if [[ "${preserve_tmp}" != "1" ]]; then
  trap 'rm -rf "$tmp_root"' EXIT
fi

runtime_home="$tmp_root/home"
out_dir="$tmp_root/wrkr-out"
mkdir -p "$runtime_home" "$out_dir"
bin_path="$tmp_root/wrkr"

CGO_ENABLED=0 go build -o "$bin_path" ./cmd/wrkr

fail_stage() {
  local stage="$1"
  local message="$2"
  echo "[wrkr][adoption][FAIL][$stage] $message" >&2
  if [[ -f "$tmp_root/${stage}.json" ]]; then
    echo "[wrkr][adoption][${stage}] payload:" >&2
    cat "$tmp_root/${stage}.json" >&2 || true
  fi
  if [[ -f "$tmp_root/${stage}.err" ]]; then
    echo "[wrkr][adoption][${stage}] stderr:" >&2
    cat "$tmp_root/${stage}.err" >&2 || true
  fi
  exit 1
}

json_get() {
  local file="$1"
  local expr="$2"
  python3 -c "import json,sys; data=json.load(open(sys.argv[1])); print(${expr})" "$file"
}

echo "[wrkr][adoption] stage=demo"
HOME="$runtime_home" "$bin_path" --json demo --out-dir "$out_dir" > "$tmp_root/demo.json" 2> "$tmp_root/demo.err" || fail_stage demo "command failed"
job_id="$(json_get "$tmp_root/demo.json" "data['job_id']")"
jobpack_path="$(json_get "$tmp_root/demo.json" "data['jobpack']")"
[[ -f "$jobpack_path" ]] || fail_stage demo "missing jobpack at $jobpack_path"
[[ -f "$out_dir/jobpacks/jobpack_${job_id}.zip" ]] || fail_stage demo "missing deterministic jobpack path"

echo "[wrkr][adoption] stage=verify"
HOME="$runtime_home" "$bin_path" --json verify "$job_id" --out-dir "$out_dir" > "$tmp_root/verify.json" 2> "$tmp_root/verify.err" || fail_stage verify "verify failed"
files_verified="$(json_get "$tmp_root/verify.json" "data['files_verified']")"
if [[ "$files_verified" -lt 4 ]]; then
  fail_stage verify "expected at least 4 verified files, got $files_verified"
fi

echo "[wrkr][adoption] stage=accept-report"
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

export GITHUB_STEP_SUMMARY="$tmp_root/step-summary.md"
HOME="$runtime_home" "$bin_path" --json accept run "$job_id" --config "$accept_cfg" --ci --out-dir "$out_dir" > "$tmp_root/accept.json" 2> "$tmp_root/accept.err" || fail_stage accept "accept run failed"
checks_failed="$(json_get "$tmp_root/accept.json" "len(data['accept_result'].get('failures', []))")"
[[ "$checks_failed" -eq 0 ]] || fail_stage accept "expected 0 acceptance failures, got $checks_failed"

HOME="$runtime_home" "$bin_path" --json report github "$job_id" --out-dir "$out_dir" > "$tmp_root/report.json" 2> "$tmp_root/report.err" || fail_stage report "report generation failed"
[[ -f "$out_dir/reports/github_summary_${job_id}.json" ]] || fail_stage report "missing github summary json"
[[ -f "$out_dir/reports/github_summary_${job_id}.md" ]] || fail_stage report "missing github summary markdown"
[[ -f "$GITHUB_STEP_SUMMARY" ]] || fail_stage report "missing GITHUB_STEP_SUMMARY output"

echo "[wrkr][adoption] stage=submit-status-checkpoint"
jobspec_path="$tmp_root/jobspec.yaml"
"$bin_path" init "$jobspec_path" > /dev/null 2> "$tmp_root/submit.err" || fail_stage submit "jobspec init failed"
submit_job_id="job_adoption_submit"
HOME="$runtime_home" "$bin_path" --json submit "$jobspec_path" --job-id "$submit_job_id" > "$tmp_root/submit.json" 2>> "$tmp_root/submit.err" || fail_stage submit "submit failed"
HOME="$runtime_home" "$bin_path" --json status "$submit_job_id" > "$tmp_root/status.json" 2> "$tmp_root/status.err" || fail_stage status "status failed"
status_value="$(json_get "$tmp_root/status.json" "data['status']")"
[[ "$status_value" == "blocked_decision" ]] || fail_stage status "expected blocked_decision, got $status_value"

HOME="$runtime_home" "$bin_path" --json checkpoint list "$submit_job_id" > "$tmp_root/checkpoints.json" 2> "$tmp_root/checkpoints.err" || fail_stage checkpoints "checkpoint list failed"
decision_checkpoint="$(python3 -c 'import json,sys; data=json.load(open(sys.argv[1])); values=[c["checkpoint_id"] for c in data if c.get("type")=="decision-needed"]; print(values[0] if values else "")' "$tmp_root/checkpoints.json")"
[[ -n "$decision_checkpoint" ]] || fail_stage checkpoints "missing decision-needed checkpoint"

echo "[wrkr][adoption] stage=bridge"
HOME="$runtime_home" "$bin_path" --json bridge work-item "$submit_job_id" --checkpoint "$decision_checkpoint" --template github --out-dir "$out_dir" > "$tmp_root/bridge.json" 2> "$tmp_root/bridge.err" || fail_stage bridge "bridge work-item failed"
bridge_path="$(json_get "$tmp_root/bridge.json" "data['json_path']")"
[[ -f "$bridge_path" ]] || fail_stage bridge "missing bridge payload file"

echo "[wrkr][adoption] stage=approve-resume"
HOME="$runtime_home" "$bin_path" --json approve "$submit_job_id" --checkpoint "$decision_checkpoint" --reason "adoption smoke approval" > "$tmp_root/approve.json" 2> "$tmp_root/approve.err" || fail_stage approve "approve failed"
HOME="$runtime_home" "$bin_path" --json resume "$submit_job_id" > "$tmp_root/resume.json" 2> "$tmp_root/resume.err" || fail_stage resume "resume failed"

HOME="$runtime_home" "$bin_path" --json status "$submit_job_id" > "$tmp_root/status_after_resume.json" 2> "$tmp_root/status_after_resume.err" || fail_stage status_after_resume "status after resume failed"
status_after_resume="$(json_get "$tmp_root/status_after_resume.json" "data['status']")"
[[ "$status_after_resume" == "running" ]] || fail_stage status_after_resume "expected running after resume, got $status_after_resume"

echo "[wrkr][adoption] stage=serve-hardening"
./scripts/test_serve_hardening.sh > "$tmp_root/serve-hardening.log" 2>&1 || fail_stage serve-hardening "serve hardening conformance failed"

echo "[wrkr][adoption] stage=wrap-fail-safe"
set +e
HOME="$runtime_home" "$bin_path" --json wrap --job-id job_adoption_wrap_fail --out-dir "$out_dir" -- sh -lc 'exit 3' > "$tmp_root/wrap.json" 2> "$tmp_root/wrap.err"
wrap_exit=$?
set -e
if [[ "$wrap_exit" -eq 0 ]]; then
  fail_stage wrap "expected non-zero for failing wrapped command"
fi
if ! rg -q 'E_ADAPTER_FAIL' "$tmp_root/wrap.err"; then
  fail_stage wrap "expected E_ADAPTER_FAIL in wrap stderr envelope"
fi

if [[ "${preserve_tmp}" == "1" ]]; then
  persist_dir="${repo_root}/wrkr-out/adoption/smoke"
  rm -rf "$persist_dir"
  mkdir -p "$(dirname "$persist_dir")"
  cp -R "$tmp_root" "$persist_dir"
  echo "[wrkr] preserved adoption smoke artifacts: ${persist_dir}"
fi

echo "[wrkr] adoption smoke ok: demo_job_id=$job_id submit_job_id=$submit_job_id"
