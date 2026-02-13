#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] local UAT"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
summary_dir="$repo_root/wrkr-out/reports"
mkdir -p "$summary_dir"
summary_json="$summary_dir/uat_summary.json"
summary_md="$summary_dir/uat_summary.md"

steps=(
  "test_adoption_smoke:./scripts/test_adoption_smoke.sh"
  "test_adapter_parity:./scripts/test_adapter_parity.sh"
)

status="pass"
results_json="[]"
{
  echo "# Wrkr Local UAT Summary"
  echo
  echo "| Step | Status |"
  echo "| --- | --- |"
} > "$summary_md"

for item in "${steps[@]}"; do
  name="${item%%:*}"
  cmd="${item#*:}"
  log_path="$summary_dir/${name}.log"
  if eval "$cmd" >"$log_path" 2>&1; then
    step_status="pass"
  else
    step_status="fail"
    status="fail"
  fi

  results_json="$(python3 -c 'import json,sys; data=json.loads(sys.argv[1]); data.append({"step": sys.argv[2], "status": sys.argv[3], "log_path": sys.argv[4]}); print(json.dumps(data, separators=(",",":"), sort_keys=True))' "$results_json" "$name" "$step_status" "$log_path")"
  echo "| ${name} | ${step_status} |" >> "$summary_md"
done

python3 - <<PY
import json
from datetime import datetime, timezone
payload = {
  "schema_id": "wrkr.uat_summary",
  "schema_version": "v1",
  "created_at": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
  "status": "${status}",
  "results": json.loads('''${results_json}'''),
}
with open("${summary_json}", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, indent=2, sort_keys=True)
    fh.write("\n")
PY

echo "[wrkr] UAT summary json: $summary_json"
echo "[wrkr] UAT summary md: $summary_md"

if [[ "$status" != "pass" ]]; then
  echo "[wrkr] local UAT failed"
  exit 1
fi

echo "[wrkr] local UAT passed"
