#!/usr/bin/env python3
"""Check static and runtime resource budgets for wrkr artifacts."""

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
    parser = argparse.ArgumentParser(description="Validate resource budgets.")
    parser.add_argument("--budgets", default="perf/resource_budgets.json")
    parser.add_argument("--report", default="wrkr-out/reports/resource_budget_report.json")
    return parser.parse_args()


def run_checked(cmd: list[str], env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(cmd, env=env, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        raise RuntimeError(f"command failed ({proc.returncode}): {' '.join(cmd)}\nstdout={proc.stdout}\nstderr={proc.stderr}")
    return proc


def dir_size(path: pathlib.Path) -> int:
    if not path.exists():
        return 0
    total = 0
    for item in path.rglob("*"):
        if item.is_file():
            total += item.stat().st_size
    return total


def ensure_parent(path: pathlib.Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def main() -> int:
    args = parse_args()
    repo_root = pathlib.Path(__file__).resolve().parent.parent

    budgets_path = pathlib.Path(args.budgets)
    config = json.loads(budgets_path.read_text(encoding="utf-8"))
    budgets: dict[str, int] = {k: int(v) for k, v in config.get("budgets", {}).items()}
    if not budgets:
        raise RuntimeError(f"no budgets defined in {budgets_path}")

    report_path = pathlib.Path(args.report)
    if not report_path.is_absolute():
        report_path = repo_root / report_path

    with tempfile.TemporaryDirectory(prefix="wrkr-resource-") as tmp:
        tmp_path = pathlib.Path(tmp)
        runtime_home = tmp_path / "home"
        runtime_home.mkdir(parents=True, exist_ok=True)
        out_dir = tmp_path / "wrkr-out"
        out_dir.mkdir(parents=True, exist_ok=True)
        bin_path = tmp_path / "wrkr"

        run_checked(["go", "build", "-o", str(bin_path), "./cmd/wrkr"], env=os.environ.copy())
        env = os.environ.copy()
        env["HOME"] = str(runtime_home)

        demo = run_checked([str(bin_path), "--json", "demo", "--out-dir", str(out_dir)], env=env)
        demo_payload: dict[str, Any] = json.loads(demo.stdout)
        job_id = str(demo_payload["job_id"])
        jobpack_path = pathlib.Path(str(demo_payload["jobpack"]))

        metrics = {
            "wrkr_binary_bytes": bin_path.stat().st_size,
            "demo_jobpack_bytes": jobpack_path.stat().st_size if jobpack_path.exists() else 0,
            "demo_store_job_bytes": dir_size(runtime_home / ".wrkr" / "jobs" / job_id),
            "demo_reports_dir_bytes": dir_size(out_dir / "reports"),
        }

    checks = []
    failed = False
    for metric_name, metric_value in sorted(metrics.items()):
        budget_key = metric_name.replace("_bytes", "_max_bytes")
        limit = budgets.get(budget_key)
        if limit is None:
            continue
        ok = metric_value <= limit
        if not ok:
            failed = True
        checks.append(
            {
                "metric": metric_name,
                "value": metric_value,
                "budget_key": budget_key,
                "max": limit,
                "ok": ok,
            }
        )

    report = {
        "schema_version": "v1",
        "checked_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "budget_file": str(budgets_path),
        "checks": checks,
        "ok": not failed,
    }

    ensure_parent(report_path)
    report_path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"[wrkr] resource budget report: {report_path}")

    if failed:
        print("[wrkr] resource budget check failed")
        return 1
    print("[wrkr] resource budget check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
