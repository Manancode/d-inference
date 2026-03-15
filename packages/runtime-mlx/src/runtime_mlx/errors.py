from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum
from typing import Any


class RuntimeErrorCode(StrEnum):
    MODEL_NOT_LOADED = "model_not_loaded"
    MODEL_ALREADY_LOADED = "model_already_loaded"
    JOB_NOT_FOUND = "job_not_found"
    JOB_NOT_CANCELLABLE = "job_not_cancellable"
    BACKEND_UNHEALTHY = "backend_unhealthy"
    INVALID_STATE = "invalid_state"
    INVALID_REQUEST = "invalid_request"


@dataclass(slots=True)
class RuntimeServiceError(Exception):
    code: RuntimeErrorCode
    message: str
    retryable: bool = False
    details: dict[str, Any] = field(default_factory=dict)

    def __str__(self) -> str:
        return f"{self.code}: {self.message}"
