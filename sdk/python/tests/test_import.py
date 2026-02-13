from wrkr import run_wrkr


def test_import() -> None:
    assert callable(run_wrkr)
