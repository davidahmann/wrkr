#!/usr/bin/env python3
"""Validate serve-mode latency and error-rate budgets under bounded load."""

from __future__ import annotations

import argparse
import concurrent.futures
import json
import os
import pathlib
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate Wrkr serve performance budgets.")
    parser.add_argument("--budgets", default="perf/serve_slo_budgets.json")
    parser.add_argument("--report", default="wrkr-out/reports/serve_slo_report.json")
    return parser.parse_args()


def run_checked(cmd: list[str], env: dict[str, str]) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(cmd, env=env, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        raise RuntimeError(f"command failed ({proc.returncode}): {' '.join(cmd)}\nstdout={proc.stdout}\nstderr={proc.stderr}")
    return proc


def ensure_parent(path: pathlib.Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def percentile_ms(samples: list[int], p: float) -> int:
    if not samples:
        return 0
    ordered = sorted(samples)
    rank = max(0, int((p / 100.0) * len(ordered) + 0.9999) - 1)
    return ordered[min(rank, len(ordered) - 1)]


def pick_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def wait_for_ready(url: str, timeout_seconds: float) -> None:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=1.0) as resp:
                if resp.status == 200:
                    return
        except Exception:
            time.sleep(0.1)
    raise RuntimeError(f"serve endpoint did not become ready: {url}")


def request_once(url: str, timeout_seconds: float) -> tuple[int, bool]:
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(url, timeout=timeout_seconds) as resp:
            ok = resp.status == 200
            return int((time.perf_counter() - started) * 1000), ok
    except (urllib.error.URLError, TimeoutError):
        return int((time.perf_counter() - started) * 1000), False


def run_load(url: str, concurrency: int, requests: int, timeout_seconds: float) -> dict[str, Any]:
    latencies: list[int] = []
    success = 0

    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
        futures = [pool.submit(request_once, url, timeout_seconds) for _ in range(requests)]
        for fut in concurrent.futures.as_completed(futures):
            latency_ms, ok = fut.result()
            latencies.append(latency_ms)
            if ok:
                success += 1

    failures = requests - success
    error_rate = failures / requests if requests > 0 else 1.0
    return {
        "requests": requests,
        "success": success,
        "failures": failures,
        "error_rate": error_rate,
        "latency_p50_ms": percentile_ms(latencies, 50),
        "latency_p95_ms": percentile_ms(latencies, 95),
        "latencies_ms": latencies,
    }


def main() -> int:
    args = parse_args()
    repo_root = pathlib.Path(__file__).resolve().parent.parent
    budgets_path = pathlib.Path(args.budgets)
    config = json.loads(budgets_path.read_text(encoding="utf-8"))

    load_cfg = config.get("load", {})
    concurrency = int(load_cfg.get("concurrency", 8))
    requests = int(load_cfg.get("requests", 120))
    timeout_seconds = float(load_cfg.get("timeout_seconds", 2.0))

    budgets = config.get("budgets", {})
    max_status_p95_ms = int(budgets.get("status_p95_ms", 0))
    max_checkpoint_p95_ms = int(budgets.get("checkpoint_list_p95_ms", 0))
    max_error_rate = float(budgets.get("error_rate_max", 1.0))

    report_path = pathlib.Path(args.report)
    if not report_path.is_absolute():
        report_path = repo_root / report_path

    with tempfile.TemporaryDirectory(prefix="wrkr-serve-slo-") as tmp:
        tmp_path = pathlib.Path(tmp)
        runtime_home = tmp_path / "home"
        runtime_home.mkdir(parents=True, exist_ok=True)
        out_dir = tmp_path / "wrkr-out"
        out_dir.mkdir(parents=True, exist_ok=True)
        bin_path = tmp_path / "wrkr"
        env = os.environ.copy()
        env["HOME"] = str(runtime_home)

        run_checked(["go", "build", "-o", str(bin_path), "./cmd/wrkr"], env=os.environ.copy())
        demo = run_checked([str(bin_path), "--json", "demo", "--out-dir", str(out_dir)], env=env)
        job_id = str(json.loads(demo.stdout)["job_id"])

        port = pick_free_port()
        listen = f"127.0.0.1:{port}"
        server = subprocess.Popen(
            [str(bin_path), "serve", "--listen", listen],
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

        try:
            status_url = f"http://{listen}/v1/jobs/{job_id}:status"
            checkpoints_url = f"http://{listen}/v1/jobs/{job_id}/checkpoints"
            wait_for_ready(status_url, timeout_seconds=8.0)

            status_result = run_load(status_url, concurrency, requests, timeout_seconds)
            checkpoint_result = run_load(checkpoints_url, concurrency, requests, timeout_seconds)
        finally:
            server.terminate()
            try:
                server.wait(timeout=3.0)
            except subprocess.TimeoutExpired:
                server.kill()
                server.wait(timeout=3.0)

    checks = [
        {
            "metric": "status_latency_p95_ms",
            "value": status_result["latency_p95_ms"],
            "max": max_status_p95_ms,
            "ok": max_status_p95_ms <= 0 or status_result["latency_p95_ms"] <= max_status_p95_ms,
        },
        {
            "metric": "checkpoint_list_latency_p95_ms",
            "value": checkpoint_result["latency_p95_ms"],
            "max": max_checkpoint_p95_ms,
            "ok": max_checkpoint_p95_ms <= 0 or checkpoint_result["latency_p95_ms"] <= max_checkpoint_p95_ms,
        },
        {
            "metric": "status_error_rate",
            "value": status_result["error_rate"],
            "max": max_error_rate,
            "ok": status_result["error_rate"] <= max_error_rate,
        },
        {
            "metric": "checkpoint_list_error_rate",
            "value": checkpoint_result["error_rate"],
            "max": max_error_rate,
            "ok": checkpoint_result["error_rate"] <= max_error_rate,
        },
    ]
    failed = any(not item["ok"] for item in checks)

    report = {
        "schema_version": "v1",
        "checked_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "budget_file": str(budgets_path),
        "load": {
            "concurrency": concurrency,
            "requests": requests,
            "timeout_seconds": timeout_seconds,
        },
        "status": status_result,
        "checkpoint_list": checkpoint_result,
        "checks": checks,
        "ok": not failed,
    }

    ensure_parent(report_path)
    report_path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"[wrkr] serve perf report: {report_path}")
    if failed:
        print("[wrkr] serve perf check failed")
        return 1
    print("[wrkr] serve perf check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
