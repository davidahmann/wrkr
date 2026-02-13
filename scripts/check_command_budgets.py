#!/usr/bin/env python3
"""Check command runtime budgets for key wrkr CLI operations."""

from __future__ import annotations

import argparse
import json
import math
import os
import pathlib
import subprocess
import tempfile
import time
from typing import Any


def run_checked(cmd: list[str], env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(cmd, env=env, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        raise RuntimeError(f"command failed ({proc.returncode}): {' '.join(cmd)}\nstdout={proc.stdout}\nstderr={proc.stderr}")
    return proc


def ensure_parent(path: pathlib.Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate runtime SLO command budgets.")
    parser.add_argument("--budgets", default="perf/runtime_slo_budgets.json")
    parser.add_argument("--report", default="wrkr-out/reports/runtime_slo_report.json")
    return parser.parse_args()


def substitute_args(args: list[str], values: dict[str, str]) -> list[str]:
    out: list[str] = []
    for item in args:
        replaced = item
        for key, value in values.items():
            replaced = replaced.replace("{" + key + "}", value)
        out.append(replaced)
    return out


def percentile_ms(samples: list[int], p: float) -> int:
    if not samples:
        return 0
    ordered = sorted(samples)
    rank = max(0, math.ceil((p / 100.0) * len(ordered)) - 1)
    return ordered[min(rank, len(ordered) - 1)]


def write_file(path: pathlib.Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def main() -> int:
    args = parse_args()
    budgets_path = pathlib.Path(args.budgets)
    repo_root = pathlib.Path(__file__).resolve().parent.parent
    report_path = pathlib.Path(args.report)
    if not report_path.is_absolute():
        report_path = repo_root / report_path
    budgets = json.loads(budgets_path.read_text(encoding="utf-8"))
    commands: list[dict[str, Any]] = budgets.get("commands", [])
    if not commands:
        raise RuntimeError(f"no commands defined in {budgets_path}")

    with tempfile.TemporaryDirectory(prefix="wrkr-slo-") as tmp:
        tmp_path = pathlib.Path(tmp)
        runtime_home = tmp_path / "home"
        runtime_home.mkdir(parents=True, exist_ok=True)
        out_dir = tmp_path / "wrkr-out"
        out_dir.mkdir(parents=True, exist_ok=True)
        bin_path = tmp_path / "wrkr"

        run_checked(["go", "build", "-o", str(bin_path), "./cmd/wrkr"], env=os.environ.copy())
        env = os.environ.copy()
        env["HOME"] = str(runtime_home)

        # Bootstrap deterministic fixtures used by lifecycle command budgets.
        boot = run_checked([str(bin_path), "--json", "demo", "--out-dir", str(out_dir)], env=env)
        demo_job_id = json.loads(boot.stdout)["job_id"]

        jobspec_path = tmp_path / "jobspec_perf.yaml"
        write_file(
            jobspec_path,
            """schema_id: wrkr.jobspec
schema_version: v1
created_at: "2026-02-14T05:00:00Z"
producer_version: perf
name: perf-submit
objective: lifecycle budget fixture
inputs:
  steps:
    - id: build
      summary: build
      command: "true"
      artifacts: [reports/perf_build.md]
      executed: true
    - id: review
      summary: approval
      decision_needed: true
      required_action: approval
      executed: false
    - id: finalize
      summary: finalize
      command: "true"
      artifacts: [reports/perf_final.md]
      executed: true
expected_artifacts: [reports/perf_final.md]
adapter:
  name: reference
budgets:
  max_wall_time_seconds: 120
  max_retries: 2
  max_step_count: 20
  max_tool_calls: 20
checkpoint_policy:
  min_interval_seconds: 1
  required_types: [plan, progress, decision-needed, completed]
environment_fingerprint:
  rules: [go_version]
""",
        )
        submit_job_id = "job_perf_submit"
        run_checked(
            [
                str(bin_path),
                "--json",
                "submit",
                str(jobspec_path),
                "--job-id",
                submit_job_id,
            ],
            env=env,
        )
        checkpoints = run_checked(
            [str(bin_path), "--json", "checkpoint", "list", submit_job_id],
            env=env,
        )
        checkpoint_items = json.loads(checkpoints.stdout)
        decision_checkpoint = next((cp["checkpoint_id"] for cp in checkpoint_items if cp.get("type") == "decision-needed"), "")
        if not decision_checkpoint:
            raise RuntimeError("missing decision-needed checkpoint in perf bootstrap fixture")

        accept_cfg = tmp_path / "accept_perf.yaml"
        write_file(
            accept_cfg,
            """schema_id: wrkr.accept_config
schema_version: v1
required_artifacts:
  - reports/demo.md
test_command: "true"
lint_command: "true"
path_rules:
  max_artifact_paths: 10
  forbidden_prefixes: []
  allowed_prefixes:
    - reports/
""",
        )

        values = {
            "job_id": demo_job_id,
            "demo_job_id": demo_job_id,
            "submit_job_id": submit_job_id,
            "decision_checkpoint": decision_checkpoint,
            "jobspec_path": str(jobspec_path),
            "accept_config": str(accept_cfg),
            "out_dir": str(out_dir),
            "home": str(runtime_home),
        }

        results: list[dict[str, Any]] = []
        failed = False
        for item in commands:
            cmd_id = str(item["id"])
            runs = max(1, int(item.get("runs", 1)))
            expected_exit = int(item.get("expect_exit", 0))
            budget_ms = int(item.get("max_ms", 0))
            max_p50_ms = int(item.get("max_p50_ms", 0))
            max_p95_ms = int(item.get("max_p95_ms", 0))
            raw_args = [str(v) for v in item.get("args", [])]
            expanded_args = substitute_args(raw_args, values)
            elapsed_samples: list[int] = []
            exit_codes: list[int] = []
            stdouts: list[str] = []
            stderrs: list[str] = []

            for _ in range(runs):
                full_cmd = [str(bin_path), *expanded_args]
                started = time.perf_counter()
                proc = subprocess.run(full_cmd, env=env, capture_output=True, text=True, check=False)
                elapsed_ms = int((time.perf_counter() - started) * 1000)
                elapsed_samples.append(elapsed_ms)
                exit_codes.append(proc.returncode)
                stdouts.append(proc.stdout.strip())
                stderrs.append(proc.stderr.strip())

            p50 = percentile_ms(elapsed_samples, 50)
            p95 = percentile_ms(elapsed_samples, 95)
            exit_ok = all(code == expected_exit for code in exit_codes)

            budget_checks: list[dict[str, Any]] = []
            if budget_ms > 0:
                check_ok = p95 <= budget_ms
                budget_checks.append({"type": "max_ms", "limit": budget_ms, "actual": p95, "ok": check_ok})
            if max_p50_ms > 0:
                check_ok = p50 <= max_p50_ms
                budget_checks.append({"type": "max_p50_ms", "limit": max_p50_ms, "actual": p50, "ok": check_ok})
            if max_p95_ms > 0:
                check_ok = p95 <= max_p95_ms
                budget_checks.append({"type": "max_p95_ms", "limit": max_p95_ms, "actual": p95, "ok": check_ok})

            budgets_ok = all(check["ok"] for check in budget_checks)
            ok = exit_ok and budgets_ok
            if not ok:
                failed = True

            results.append(
                {
                    "id": cmd_id,
                    "args": expanded_args,
                    "runs": runs,
                    "expected_exit": expected_exit,
                    "elapsed_samples_ms": elapsed_samples,
                    "elapsed_p50_ms": p50,
                    "elapsed_p95_ms": p95,
                    "exit_codes": exit_codes,
                    "budget_checks": budget_checks,
                    "ok": ok,
                    "stdout": stdouts[-1] if stdouts else "",
                    "stderr": stderrs[-1] if stderrs else "",
                }
            )

    report = {
        "schema_version": "v1",
        "checked_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "budget_file": str(budgets_path),
        "results": results,
        "ok": not failed,
    }

    ensure_parent(report_path)
    report_path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    print(f"[wrkr] runtime budget report: {report_path}")
    if failed:
        print("[wrkr] runtime budget check failed")
        return 1
    print("[wrkr] runtime budget check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
