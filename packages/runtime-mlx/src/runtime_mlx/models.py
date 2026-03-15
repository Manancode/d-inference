from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC, datetime
from enum import StrEnum


def utcnow() -> datetime:
    return datetime.now(tz=UTC)


class WorkerHealth(StrEnum):
    HEALTHY = "healthy"
    DEGRADED = "degraded"
    UNHEALTHY = "unhealthy"


class JobState(StrEnum):
    RUNNING = "running"
    COMPLETED = "completed"
    CANCELLED = "cancelled"
    FAILED = "failed"


@dataclass(slots=True)
class LoadModelRequest:
    model_id: str
    model_path: str
    revision: str | None = None


@dataclass(slots=True)
class LoadModelResult:
    model_id: str
    revision: str | None
    loaded_at: datetime
    backend_name: str


@dataclass(slots=True)
class GenerateRequest:
    job_id: str
    prompt: str
    max_output_tokens: int


@dataclass(slots=True)
class GenerateResult:
    job_id: str
    model_id: str
    output_text: str
    prompt_tokens: int
    completion_tokens: int
    state: JobState
    finished_at: datetime


@dataclass(slots=True)
class ActiveJob:
    job_id: str
    model_id: str
    state: JobState = JobState.RUNNING
    prompt_tokens: int = 0
    completion_tokens: int = 0
    started_at: datetime = field(default_factory=utcnow)
    updated_at: datetime = field(default_factory=utcnow)
    error_message: str | None = None


@dataclass(slots=True)
class CancelJobResult:
    job_id: str
    cancelled: bool
    state: JobState


@dataclass(slots=True)
class UsageReport:
    job_id: str
    model_id: str
    state: JobState
    prompt_tokens: int
    completion_tokens: int
    updated_at: datetime
    error_message: str | None = None


@dataclass(slots=True)
class HealthCheckResult:
    status: WorkerHealth
    backend_name: str
    loaded_model_id: str | None
    active_job_count: int
    checked_at: datetime
    notes: list[str] = field(default_factory=list)


@dataclass(slots=True)
class ErrorResponse:
    code: str
    message: str
