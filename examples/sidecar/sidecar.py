#!/usr/bin/env python3
"""Deterministic sidecar transport example for Wrkr lanes."""

from __future__ import annotations

import argparse
import json
import pathlib
import subprocess
from datetime import datetime, timezone


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Wrkr sidecar example")
    parser.add_argument("--request", required=True, help="Path to sidecar request JSON")
    parser.add_argument("--dry-run", action="store_true", help="Do not execute command, only emit deterministic output")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    request_path = pathlib.Path(args.request).resolve()
    request = json.loads(request_path.read_text(encoding="utf-8"))

    lane = str(request.get("lane", "default")).strip() or "default"
    out_root = pathlib.Path("wrkr-out") / "integrations" / lane
    out_root.mkdir(parents=True, exist_ok=True)

    normalized_request_path = out_root / "request.json"
    normalized_request_path.write_text(
        json.dumps(request, sort_keys=True, separators=(",", ":")) + "\n",
        encoding="utf-8",
    )

    command = list(request.get("command", []))
    dry_run = bool(request.get("dry_run", False)) or args.dry_run

    timestamp = str(request.get("timestamp_utc") or "").strip()
    if not timestamp:
        timestamp = datetime.now(timezone.utc).replace(microsecond=0).isoformat()

    result = {
        "timestamp_utc": timestamp,
        "lane": lane,
        "job_id": request.get("job_id"),
        "dry_run": dry_run,
        "command": command,
        "returncode": 0,
        "stdout": "",
        "stderr": "",
    }
    if not dry_run and command:
        proc = subprocess.run(command, check=False, text=True, capture_output=True)
        result["returncode"] = proc.returncode
        result["stdout"] = proc.stdout
        result["stderr"] = proc.stderr

    result_path = out_root / "result.json"
    result_path.write_text(
        json.dumps(result, sort_keys=True, separators=(",", ":")) + "\n",
        encoding="utf-8",
    )

    log_path = out_root / "sidecar.log"
    log_path.write_text(
        f"lane={lane} job_id={request.get('job_id')} dry_run={dry_run} result={result_path}\n",
        encoding="utf-8",
    )
    print(f"result={result_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
