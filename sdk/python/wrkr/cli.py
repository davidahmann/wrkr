"""Thin wrapper around the wrkr CLI."""

from __future__ import annotations

import json
import subprocess
from typing import Any, Sequence


def run_wrkr(args: Sequence[str]) -> subprocess.CompletedProcess[str]:
    """Execute wrkr and return the completed process."""

    return subprocess.run(
        ["wrkr", *args],
        check=False,
        text=True,
        capture_output=True,
    )


def run_wrkr_json(args: Sequence[str]) -> dict[str, Any]:
    """Execute wrkr with --json and parse stdout."""

    proc = run_wrkr([*args, "--json"])
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or f"wrkr failed with exit code {proc.returncode}")
    try:
        return json.loads(proc.stdout or "{}")
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"wrkr returned invalid JSON: {exc}") from exc


def status(job_id: str) -> dict[str, Any]:
    """Get `wrkr status` JSON."""

    return run_wrkr_json(["status", job_id])


def wrap(command: Sequence[str], *, job_id: str | None = None) -> dict[str, Any]:
    """Run `wrkr wrap -- <command...>` and parse JSON output."""

    args: list[str] = ["wrap"]
    if job_id:
        args.extend(["--job-id", job_id])
    args.append("--")
    args.extend(command)
    return run_wrkr_json(args)


def accept_run(job_id: str, *, config: str | None = None, ci: bool = False) -> dict[str, Any]:
    """Run `wrkr accept run` and parse JSON output."""

    args: list[str] = ["accept", "run", job_id]
    if config:
        args.extend(["--config", config])
    if ci:
        args.append("--ci")
    return run_wrkr_json(args)
