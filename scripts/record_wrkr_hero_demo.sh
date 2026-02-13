#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WRKR_BIN="${WRKR_BIN:-}"
OUTPUT_DIR="${OUTPUT_DIR:-${REPO_ROOT}/docs/assets}"
CAST_PATH="${CAST_PATH:-${OUTPUT_DIR}/wrkr_demo_25s.cast}"
GIF_PATH="${GIF_PATH:-${OUTPUT_DIR}/wrkr_demo_25s.gif}"
MP4_PATH="${MP4_PATH:-${OUTPUT_DIR}/wrkr_demo_25s.mp4}"
WORKSPACE="${WORKSPACE:-${REPO_ROOT}/wrkr-out/hero_demo/workspace}"

if [[ -z "${WRKR_BIN}" ]]; then
  if command -v wrkr >/dev/null 2>&1; then
    WRKR_BIN="$(command -v wrkr)"
  elif [[ -x "${REPO_ROOT}/wrkr" ]]; then
    WRKR_BIN="${REPO_ROOT}/wrkr"
  else
    (cd "${REPO_ROOT}" && go build -o ./wrkr ./cmd/wrkr)
    WRKR_BIN="${REPO_ROOT}/wrkr"
  fi
fi

for required in asciinema agg python3; do
  if ! command -v "${required}" >/dev/null 2>&1; then
    echo "missing required dependency: ${required}" >&2
    exit 2
  fi
done

mkdir -p "${OUTPUT_DIR}" "${WORKSPACE}" "${REPO_ROOT}/docs-site/public/assets"

DRIVER_SCRIPT="$(mktemp)"
cat > "${DRIVER_SCRIPT}" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

WRKR_BIN="$1"
REPO_ROOT="$2"
WORKSPACE="$3"

mkdir -p "${WORKSPACE}" "${WORKSPACE}/home"
export HOME="${WORKSPACE}/home"
cd "${WORKSPACE}"

echo '$ wrkr demo --json'
"${WRKR_BIN}" demo --json > demo.json
demo_job_id="$(python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path("demo.json").read_text(encoding="utf-8"))
print(payload.get("job_id", ""))
PY
)"
python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path("demo.json").read_text(encoding="utf-8"))
print(f"job_id={payload.get('job_id')}")
print(f"jobpack={payload.get('jobpack')}")
PY
sleep 3

echo
echo '$ wrkr verify ${demo_job_id} --json'
"${WRKR_BIN}" verify "${demo_job_id}" --json > verify.json
python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path("verify.json").read_text(encoding="utf-8"))
print(f"files_verified={payload.get('files_verified')}")
print(f"manifest_sha256={payload.get('manifest_sha256')}")
PY
sleep 3

job_id="job_hero_$(date +%s)"
echo
echo '$ wrkr submit examples/integrations/blessed_jobspec.yaml --job-id ${job_id} --json'
"${WRKR_BIN}" submit "${REPO_ROOT}/examples/integrations/blessed_jobspec.yaml" --job-id "${job_id}" --json > submit.json
python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path("submit.json").read_text(encoding="utf-8"))
print(f"submitted_job={payload.get('job_id')}")
print(f"status={payload.get('status')}")
PY
sleep 3

echo
echo '$ wrkr checkpoint list ${job_id} --json'
"${WRKR_BIN}" checkpoint list "${job_id}" --json > checkpoints.json
decision_checkpoint="$(python3 - <<'PY'
import json
from pathlib import Path
items = json.loads(Path("checkpoints.json").read_text(encoding="utf-8"))
decision = ""
for item in items:
    if item.get("type") == "decision-needed":
        decision = item.get("checkpoint_id", "")
        break
print(decision)
PY
)"
printf 'decision_checkpoint=%s\n' "${decision_checkpoint}"
sleep 3

echo
echo '$ wrkr approve ${job_id} --checkpoint ${decision_checkpoint} --reason "hero demo approval" --json'
"${WRKR_BIN}" approve "${job_id}" --checkpoint "${decision_checkpoint}" --reason "hero demo approval" --json > approve.json
python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path("approve.json").read_text(encoding="utf-8"))
print(f"approved_checkpoint={payload.get('checkpoint_id')}")
print(f"approved_by={payload.get('approved_by')}")
PY
sleep 3

echo
echo '$ wrkr resume ${job_id} --json'
"${WRKR_BIN}" resume "${job_id}" --json > resume.json
python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path("resume.json").read_text(encoding="utf-8"))
print(f"resume_status={payload.get('status')}")
print(f"next_step_index={payload.get('next_step_index')}")
PY
sleep 3
SH
chmod +x "${DRIVER_SCRIPT}"

asciinema rec \
  --overwrite \
  --idle-time-limit 5 \
  --quiet \
  --command "bash ${DRIVER_SCRIPT} $(printf '%q' "${WRKR_BIN}") $(printf '%q' "${REPO_ROOT}") $(printf '%q' "${WORKSPACE}")" \
  "${CAST_PATH}"

agg \
  --theme github-dark \
  --speed 1.0 \
  --idle-time-limit 5 \
  --font-size 16 \
  "${CAST_PATH}" \
  "${GIF_PATH}"

if command -v ffmpeg >/dev/null 2>&1; then
  ffmpeg -y -loglevel error -i "${GIF_PATH}" -movflags faststart "${MP4_PATH}"
fi

cp "${GIF_PATH}" "${REPO_ROOT}/docs-site/public/assets/$(basename "${GIF_PATH}")"
if [[ -f "${MP4_PATH}" ]]; then
  cp "${MP4_PATH}" "${REPO_ROOT}/docs-site/public/assets/$(basename "${MP4_PATH}")"
fi

rm -f "${DRIVER_SCRIPT}"

echo "wrote cast: ${CAST_PATH}"
echo "wrote gif: ${GIF_PATH}"
if [[ -f "${MP4_PATH}" ]]; then
  echo "wrote mp4: ${MP4_PATH}"
fi
