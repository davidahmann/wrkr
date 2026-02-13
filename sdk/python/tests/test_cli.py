from __future__ import annotations

import subprocess

import wrkr.cli as cli


def test_run_wrkr_json_parses(monkeypatch) -> None:
    def fake_run(*_args, **_kwargs):
        return subprocess.CompletedProcess(
            args=["wrkr", "status", "job_1", "--json"],
            returncode=0,
            stdout='{"job_id":"job_1","status":"running"}',
            stderr="",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    payload = cli.run_wrkr_json(["status", "job_1"])
    assert payload["job_id"] == "job_1"


def test_wrap_builds_command(monkeypatch) -> None:
    recorded = {}

    def fake_run(args, **_kwargs):
        recorded["args"] = args
        return subprocess.CompletedProcess(
            args=args,
            returncode=0,
            stdout='{"job_id":"job_wrap","jobpack_path":"x"}',
            stderr="",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    payload = cli.wrap(["sh", "-lc", "echo ok"], job_id="job_wrap")
    assert payload["job_id"] == "job_wrap"
    assert recorded["args"][:4] == ["wrkr", "wrap", "--job-id", "job_wrap"]
