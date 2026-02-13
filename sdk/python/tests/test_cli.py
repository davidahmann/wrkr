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


def test_wrap_without_job_id(monkeypatch) -> None:
    recorded = {}

    def fake_run(args, **_kwargs):
        recorded["args"] = args
        return subprocess.CompletedProcess(
            args=args,
            returncode=0,
            stdout='{"job_id":"job_wrap"}',
            stderr="",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    payload = cli.wrap(["echo", "ok"])
    assert payload["job_id"] == "job_wrap"
    assert recorded["args"][:2] == ["wrkr", "wrap"]
    assert "--job-id" not in recorded["args"]


def test_status_builds_expected_args(monkeypatch) -> None:
    recorded = {}

    def fake_run(args, **_kwargs):
        recorded["args"] = args
        return subprocess.CompletedProcess(
            args=args,
            returncode=0,
            stdout='{"job_id":"job_status"}',
            stderr="",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    payload = cli.status("job_status")
    assert payload["job_id"] == "job_status"
    assert recorded["args"] == ["wrkr", "status", "job_status", "--json"]


def test_accept_run_builds_expected_args(monkeypatch) -> None:
    recorded = {}

    def fake_run(args, **_kwargs):
        recorded["args"] = args
        return subprocess.CompletedProcess(
            args=args,
            returncode=0,
            stdout='{"job_id":"job_accept"}',
            stderr="",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    payload = cli.accept_run("job_accept", config="accept.yaml", ci=True)
    assert payload["job_id"] == "job_accept"
    assert recorded["args"] == ["wrkr", "accept", "run", "job_accept", "--config", "accept.yaml", "--ci", "--json"]


def test_run_wrkr_json_raises_on_non_zero_with_stderr(monkeypatch) -> None:
    def fake_run(*_args, **_kwargs):
        return subprocess.CompletedProcess(
            args=["wrkr", "status", "job_1", "--json"],
            returncode=5,
            stdout="",
            stderr="acceptance failed",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    try:
        cli.run_wrkr_json(["status", "job_1"])
    except RuntimeError as exc:
        assert "acceptance failed" in str(exc)
    else:
        raise AssertionError("expected RuntimeError")


def test_run_wrkr_json_raises_on_non_zero_without_stderr(monkeypatch) -> None:
    def fake_run(*_args, **_kwargs):
        return subprocess.CompletedProcess(
            args=["wrkr", "status", "job_1", "--json"],
            returncode=9,
            stdout="",
            stderr="",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    try:
        cli.run_wrkr_json(["status", "job_1"])
    except RuntimeError as exc:
        assert "exit code 9" in str(exc)
    else:
        raise AssertionError("expected RuntimeError")


def test_run_wrkr_json_raises_on_invalid_json(monkeypatch) -> None:
    def fake_run(*_args, **_kwargs):
        return subprocess.CompletedProcess(
            args=["wrkr", "status", "job_1", "--json"],
            returncode=0,
            stdout="{invalid-json",
            stderr="",
        )

    monkeypatch.setattr(cli.subprocess, "run", fake_run)
    try:
        cli.run_wrkr_json(["status", "job_1"])
    except RuntimeError as exc:
        assert "invalid JSON" in str(exc)
    else:
        raise AssertionError("expected RuntimeError")
