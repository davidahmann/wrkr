"""Thin wrapper around the wrkr CLI."""

from __future__ import annotations

import subprocess
from typing import Sequence


def run_wrkr(args: Sequence[str]) -> subprocess.CompletedProcess[str]:
    """Execute wrkr and return the completed process."""

    return subprocess.run(
        ["wrkr", *args],
        check=False,
        text=True,
        capture_output=True,
    )
