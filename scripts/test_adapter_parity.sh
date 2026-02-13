#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] adapter parity"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

runtime_home="$tmp_root/home"
out_dir="$tmp_root/wrkr-out"
mkdir -p "$runtime_home" "$out_dir"
bin_path="$tmp_root/wrkr"

CGO_ENABLED=0 go build -o "$bin_path" ./cmd/wrkr

go test ./core/adapters/reference ./core/adapters/wrap ./core/bridge -count=1

HOME="$runtime_home" "$bin_path" --json demo --out-dir "$out_dir" > "$tmp_root/demo.json"
demo_job_id="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["job_id"])' "$tmp_root/demo.json")"
if [[ ! -f "$out_dir/jobpacks/jobpack_${demo_job_id}.zip" ]]; then
  echo "[wrkr][adapter parity] missing demo jobpack"
  exit 1
fi

HOME="$runtime_home" "$bin_path" --json wrap --job-id job_adapter_parity_wrap --artifact reports/wrap.txt --out-dir "$out_dir" -- sh -lc 'printf wrapped > /dev/null' > "$tmp_root/wrap.json"
if ! rg -q '"job_id": "job_adapter_parity_wrap"' "$tmp_root/wrap.json"; then
  echo "[wrkr][adapter parity] missing wrap job id payload"
  cat "$tmp_root/wrap.json"
  exit 1
fi

work_dir="$tmp_root/sidecar-run"
mkdir -p "$work_dir"
cp "$repo_root/examples/sidecar/request_fixture.json" "$work_dir/request.json"
(cd "$work_dir" && python3 "$repo_root/examples/sidecar/sidecar.py" --request "$work_dir/request.json" --dry-run >/dev/null)
first_result="$work_dir/wrkr-out/integrations/fixture/result.json"
cp "$first_result" "$work_dir/result.first.json"
(cd "$work_dir" && python3 "$repo_root/examples/sidecar/sidecar.py" --request "$work_dir/request.json" --dry-run >/dev/null)
second_result="$work_dir/wrkr-out/integrations/fixture/result.json"

cmp "$work_dir/result.first.json" "$second_result" >/dev/null || {
  echo "[wrkr][adapter parity] sidecar dry-run output is not deterministic"
  diff -u "$work_dir/result.first.json" "$second_result" || true
  exit 1
}

for required in request.json result.json sidecar.log; do
  if [[ ! -f "$work_dir/wrkr-out/integrations/fixture/$required" ]]; then
    echo "[wrkr][adapter parity] missing sidecar output: $required"
    exit 1
  fi
done

echo "[wrkr] adapter parity ok"
