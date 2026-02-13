#!/usr/bin/env python3
"""Validate scale-profile performance for larger structured jobs."""

from __future__ import annotations

import argparse
import json
import os
import pathlib
import subprocess
import tempfile
import time
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate Wrkr scale profile budgets.")
    parser.add_argument("--budgets", default="perf/scale_profile_budgets.json")
    parser.add_argument("--report", default="wrkr-out/reports/scale_profile_report.json")
    return parser.parse_args()


def run_checked(cmd: list[str], env: dict[str, str]) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(cmd, env=env, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        raise RuntimeError(f"command failed ({proc.returncode}): {' '.join(cmd)}\nstdout={proc.stdout}\nstderr={proc.stderr}")
    return proc


def run_timed(cmd: list[str], env: dict[str, str]) -> tuple[subprocess.CompletedProcess[str], int]:
    started = time.perf_counter()
    proc = run_checked(cmd, env)
    elapsed_ms = int((time.perf_counter() - started) * 1000)
    return proc, elapsed_ms


def ensure_parent(path: pathlib.Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def build_jobspec(step_count: int) -> str:
    lines = [
        "schema_id: wrkr.jobspec",
        "schema_version: v1",
        'created_at: "2026-02-14T06:00:00Z"',
        "producer_version: perf",
        "name: scale-profile",
        "objective: scale profile fixture",
        "inputs:",
        "  steps:",
    ]
    for i in range(step_count):
        lines.extend(
            [
                f"    - id: step_{i:04d}",
                f"      summary: scale step {i}",
                '      command: "true"',
                f"      artifacts: [reports/scale_{i:04d}.md]",
                "      executed: true",
            ]
        )
    lines.extend(
        [
            "expected_artifacts:",
            "  - reports/scale_0000.md",
            "adapter:",
            "  name: reference",
            "budgets:",
            "  max_wall_time_seconds: 600",
            "  max_retries: 2",
            f"  max_step_count: {step_count + 10}",
            f"  max_tool_calls: {step_count + 10}",
            "checkpoint_policy:",
            "  min_interval_seconds: 1",
            "  required_types: [plan, progress, completed]",
            "environment_fingerprint:",
            "  rules: [go_version]",
        ]
    )
    return "\n".join(lines) + "\n"


def main() -> int:
    args = parse_args()
    repo_root = pathlib.Path(__file__).resolve().parent.parent
    budgets_path = pathlib.Path(args.budgets)
    config = json.loads(budgets_path.read_text(encoding="utf-8"))

    profile = config.get("profile", {})
    step_count = int(profile.get("step_count", 200))
    job_id = str(profile.get("job_id", "job_scale_profile"))

    budgets = config.get("budgets", {})
    max_submit_ms = int(budgets.get("submit_max_ms", 0))
    max_status_ms = int(budgets.get("status_max_ms", 0))
    max_checkpoint_list_ms = int(budgets.get("checkpoint_list_max_ms", 0))
    max_export_ms = int(budgets.get("export_max_ms", 0))
    max_verify_ms = int(budgets.get("verify_max_ms", 0))
    max_jobpack_bytes = int(budgets.get("jobpack_max_bytes", 0))

    report_path = pathlib.Path(args.report)
    if not report_path.is_absolute():
        report_path = repo_root / report_path

    with tempfile.TemporaryDirectory(prefix="wrkr-scale-") as tmp:
        tmp_path = pathlib.Path(tmp)
        runtime_home = tmp_path / "home"
        runtime_home.mkdir(parents=True, exist_ok=True)
        out_dir = tmp_path / "wrkr-out"
        out_dir.mkdir(parents=True, exist_ok=True)
        bin_path = tmp_path / "wrkr"

        run_checked(["go", "build", "-o", str(bin_path), "./cmd/wrkr"], env=os.environ.copy())
        env = os.environ.copy()
        env["HOME"] = str(runtime_home)

        jobspec_path = tmp_path / "jobspec_scale.yaml"
        jobspec_path.write_text(build_jobspec(step_count), encoding="utf-8")

        _, submit_ms = run_timed(
            [str(bin_path), "--json", "submit", str(jobspec_path), "--job-id", job_id],
            env,
        )
        _, status_ms = run_timed([str(bin_path), "--json", "status", job_id], env)
        checkpoints_proc, checkpoint_list_ms = run_timed(
            [str(bin_path), "--json", "checkpoint", "list", job_id],
            env,
        )
        checkpoint_count = len(json.loads(checkpoints_proc.stdout))

        _, export_ms = run_timed(
            [str(bin_path), "--json", "export", job_id, "--out-dir", str(out_dir)],
            env,
        )
        _, verify_ms = run_timed(
            [str(bin_path), "--json", "verify", job_id, "--out-dir", str(out_dir)],
            env,
        )
        jobpack_path = out_dir / "jobpacks" / f"jobpack_{job_id}.zip"
        jobpack_bytes = jobpack_path.stat().st_size if jobpack_path.exists() else 0

    checks = [
        {"metric": "submit_ms", "value": submit_ms, "max": max_submit_ms, "ok": max_submit_ms <= 0 or submit_ms <= max_submit_ms},
        {"metric": "status_ms", "value": status_ms, "max": max_status_ms, "ok": max_status_ms <= 0 or status_ms <= max_status_ms},
        {
            "metric": "checkpoint_list_ms",
            "value": checkpoint_list_ms,
            "max": max_checkpoint_list_ms,
            "ok": max_checkpoint_list_ms <= 0 or checkpoint_list_ms <= max_checkpoint_list_ms,
        },
        {"metric": "export_ms", "value": export_ms, "max": max_export_ms, "ok": max_export_ms <= 0 or export_ms <= max_export_ms},
        {"metric": "verify_ms", "value": verify_ms, "max": max_verify_ms, "ok": max_verify_ms <= 0 or verify_ms <= max_verify_ms},
        {
            "metric": "jobpack_bytes",
            "value": jobpack_bytes,
            "max": max_jobpack_bytes,
            "ok": max_jobpack_bytes <= 0 or jobpack_bytes <= max_jobpack_bytes,
        },
    ]

    failed = any(not item["ok"] for item in checks)
    report: dict[str, Any] = {
        "schema_version": "v1",
        "checked_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "budget_file": str(budgets_path),
        "profile": {"job_id": job_id, "step_count": step_count, "checkpoint_count": checkpoint_count},
        "checks": checks,
        "ok": not failed,
    }

    ensure_parent(report_path)
    report_path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"[wrkr] scale profile report: {report_path}")
    if failed:
        print("[wrkr] scale profile check failed")
        return 1
    print("[wrkr] scale profile check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
