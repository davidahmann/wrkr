#!/usr/bin/env python3
"""Validate Go + Python coverage against configured thresholds."""

from __future__ import annotations

import argparse
import json
import pathlib
import re
import subprocess
import tempfile
import time
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check coverage thresholds for wrkr")
    parser.add_argument("--config", default="perf/coverage_thresholds.json")
    parser.add_argument("--report", default="wrkr-out/reports/coverage_report.json")
    return parser.parse_args()


def run_checked(cmd: list[str], cwd: pathlib.Path) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        joined = " ".join(cmd)
        raise RuntimeError(f"command failed ({proc.returncode}): {joined}\nstdout={proc.stdout}\nstderr={proc.stderr}")
    return proc


def parse_go_total(go_cover_path: pathlib.Path, repo_root: pathlib.Path) -> float:
    proc = run_checked(["go", "tool", "cover", "-func", str(go_cover_path)], cwd=repo_root)
    total_line = proc.stdout.strip().splitlines()[-1]
    match = re.search(r"([0-9]+(?:\.[0-9]+)?)%$", total_line)
    if not match:
        raise RuntimeError(f"unable to parse go coverage total from line: {total_line}")
    return float(match.group(1))


def parse_python_total(py_cov_json: pathlib.Path) -> float:
    payload = json.loads(py_cov_json.read_text(encoding="utf-8"))
    totals = payload.get("totals", {})
    for key in ("percent_covered", "percent_covered_display"):
        value = totals.get(key)
        if value is None:
            continue
        return float(value)
    raise RuntimeError("unable to parse python coverage percent from json report")


def ensure_parent(path: pathlib.Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def main() -> int:
    args = parse_args()
    repo_root = pathlib.Path(__file__).resolve().parent.parent
    config_path = pathlib.Path(args.config)
    if not config_path.is_absolute():
        config_path = repo_root / config_path

    report_path = pathlib.Path(args.report)
    if not report_path.is_absolute():
        report_path = repo_root / report_path

    config = json.loads(config_path.read_text(encoding="utf-8"))
    go_min = float(config["go_min_percent"])
    py_min = float(config["python_min_percent"])

    with tempfile.TemporaryDirectory(prefix="wrkr-coverage-") as tmp:
        tmp_path = pathlib.Path(tmp)
        go_cover = tmp_path / "go.cover"
        py_cover = tmp_path / "python_coverage.json"

        run_checked(
            ["go", "test", "./...", "-covermode=atomic", "-coverprofile", str(go_cover)],
            cwd=repo_root,
        )
        go_percent = parse_go_total(go_cover, repo_root)

        run_checked(
            [
                "uv",
                "run",
                "--python",
                "3.13",
                "--extra",
                "dev",
                "pytest",
                "-q",
                "--cov=wrkr",
                "--cov-report",
                f"json:{py_cover}",
            ],
            cwd=repo_root / "sdk/python",
        )
        py_percent = parse_python_total(py_cover)

    results: dict[str, Any] = {
        "schema_version": "v1",
        "checked_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "config_path": str(config_path),
        "go": {
            "measured_percent": round(go_percent, 2),
            "min_percent": go_min,
            "ok": go_percent >= go_min,
        },
        "python": {
            "measured_percent": round(py_percent, 2),
            "min_percent": py_min,
            "ok": py_percent >= py_min,
        },
    }
    results["ok"] = bool(results["go"]["ok"] and results["python"]["ok"])

    ensure_parent(report_path)
    report_path.write_text(json.dumps(results, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    print(f"[wrkr] coverage report: {report_path}")
    if not results["ok"]:
        print(
            f"[wrkr] coverage check failed: go={results['go']['measured_percent']}% (min {go_min}%), "
            f"python={results['python']['measured_percent']}% (min {py_min}%)"
        )
        return 1

    print(
        f"[wrkr] coverage check passed: go={results['go']['measured_percent']}% "
        f"python={results['python']['measured_percent']}%"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
