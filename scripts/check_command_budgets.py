#!/usr/bin/env python3
"""Check command runtime budgets for key wrkr CLI operations."""

from __future__ import annotations

import argparse
import json
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

        # Bootstrap a deterministic demo job for commands that need {job_id}.
        boot = run_checked([str(bin_path), "--json", "demo", "--out-dir", str(out_dir)], env=env)
        job_id = json.loads(boot.stdout)["job_id"]

        values = {
            "job_id": job_id,
            "out_dir": str(out_dir),
            "home": str(runtime_home),
        }

        results: list[dict[str, Any]] = []
        failed = False
        for item in commands:
            cmd_id = str(item["id"])
            budget_ms = int(item["max_ms"])
            raw_args = [str(v) for v in item.get("args", [])]
            expanded_args = substitute_args(raw_args, values)
            full_cmd = [str(bin_path), *expanded_args]

            started = time.perf_counter()
            proc = subprocess.run(full_cmd, env=env, capture_output=True, text=True, check=False)
            elapsed_ms = int((time.perf_counter() - started) * 1000)
            ok = proc.returncode == 0 and elapsed_ms <= budget_ms
            if not ok:
                failed = True

            results.append(
                {
                    "id": cmd_id,
                    "args": expanded_args,
                    "budget_ms": budget_ms,
                    "elapsed_ms": elapsed_ms,
                    "exit_code": proc.returncode,
                    "ok": ok,
                    "stdout": proc.stdout.strip(),
                    "stderr": proc.stderr.strip(),
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
