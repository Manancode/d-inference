from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass(slots=True)
class CoordinatorClientError(Exception):
    message: str
    status_code: int | None = None
    response_body: Any | None = None
    request_path: str | None = None
    details: dict[str, Any] = field(default_factory=dict)

    def __str__(self) -> str:
        status = f" status={self.status_code}" if self.status_code is not None else ""
        path = f" path={self.request_path}" if self.request_path else ""
        return f"{self.message}{status}{path}"

