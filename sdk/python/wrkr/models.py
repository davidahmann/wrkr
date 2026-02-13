"""Lightweight response models for wrkr CLI wrappers."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass(frozen=True)
class CommandResult:
    """Generic parsed payload wrapper."""

    raw: dict[str, Any]

    @property
    def job_id(self) -> str | None:
        value = self.raw.get("job_id")
        if isinstance(value, str):
            return value
        return None
