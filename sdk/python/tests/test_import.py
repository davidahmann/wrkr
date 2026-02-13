from wrkr import CommandResult, accept_run, run_wrkr, run_wrkr_json, status, wrap


def test_import() -> None:
    assert callable(run_wrkr)
    assert callable(run_wrkr_json)
    assert callable(status)
    assert callable(wrap)
    assert callable(accept_run)
    assert CommandResult({"job_id": "job_1"}).job_id == "job_1"
