from __future__ import annotations

from dataclasses import replace
from datetime import UTC, datetime
from typing import Callable, Protocol

from .errors import RuntimeErrorCode, RuntimeServiceError
from .models import (
    ActiveJob,
    CancelJobResult,
    GenerateRequest,
    GenerateResult,
    HealthCheckResult,
    JobState,
    LoadModelRequest,
    LoadModelResult,
    UsageReport,
    WorkerHealth,
)


class RuntimeBackend(Protocol):
    name: str

    def load_model(self, request: LoadModelRequest) -> None: ...

    def cancel_job(self, job_id: str) -> bool: ...

    def health(self) -> tuple[WorkerHealth, list[str]]: ...

    def generate(self, request: GenerateRequest, model_id: str) -> GenerateResult: ...


class RuntimeService:
    def __init__(self, backend: RuntimeBackend, now: Callable[[], datetime] | None = None) -> None:
        self._backend = backend
        self._now = now or (lambda: datetime.now(tz=UTC))
        self._loaded_model: LoadModelResult | None = None
        self._jobs: dict[str, ActiveJob] = {}

    def load_model(self, request: LoadModelRequest) -> LoadModelResult:
        if self._loaded_model and self._loaded_model.model_id == request.model_id:
            raise RuntimeServiceError(
                code=RuntimeErrorCode.MODEL_ALREADY_LOADED,
                message=f"model {request.model_id} already loaded",
            )

        self._backend.load_model(request)
        self._loaded_model = LoadModelResult(
            model_id=request.model_id,
            revision=request.revision,
            loaded_at=self._now(),
            backend_name=self._backend.name,
        )
        self._jobs.clear()
        return self._loaded_model

    def generate(self, request: GenerateRequest) -> GenerateResult:
        if self._loaded_model is None:
            raise RuntimeServiceError(
                code=RuntimeErrorCode.MODEL_NOT_LOADED,
                message="cannot generate before a model is loaded",
            )

        self.register_job(request.job_id, self._loaded_model.model_id)
        result = self._backend.generate(request, self._loaded_model.model_id)
        state = JobState.COMPLETED if result.state is JobState.COMPLETED else result.state
        self.update_usage(
            request.job_id,
            prompt_tokens=result.prompt_tokens,
            completion_tokens=result.completion_tokens,
            state=state,
        )
        return result

    def register_job(self, job_id: str, model_id: str | None = None) -> ActiveJob:
        active_model = model_id or (self._loaded_model.model_id if self._loaded_model else None)
        if active_model is None:
            raise RuntimeServiceError(
                code=RuntimeErrorCode.MODEL_NOT_LOADED,
                message="cannot register a job before a model is loaded",
            )

        job = ActiveJob(job_id=job_id, model_id=active_model, updated_at=self._now())
        self._jobs[job_id] = job
        return job

    def update_usage(
        self,
        job_id: str,
        *,
        prompt_tokens: int = 0,
        completion_tokens: int = 0,
        state: JobState | None = None,
        error_message: str | None = None,
    ) -> ActiveJob:
        job = self._get_job(job_id)
        job.prompt_tokens += prompt_tokens
        job.completion_tokens += completion_tokens
        job.updated_at = self._now()
        if state is not None:
            job.state = state
        if error_message is not None:
            job.error_message = error_message
        return job

    def cancel_job(self, job_id: str) -> CancelJobResult:
        job = self._get_job(job_id)
        if job.state is not JobState.RUNNING:
            raise RuntimeServiceError(
                code=RuntimeErrorCode.JOB_NOT_CANCELLABLE,
                message=f"job {job_id} is not running",
            )

        cancelled = self._backend.cancel_job(job_id)
        if not cancelled:
            raise RuntimeServiceError(
                code=RuntimeErrorCode.INVALID_STATE,
                message=f"backend refused to cancel job {job_id}",
                retryable=True,
            )

        job.state = JobState.CANCELLED
        job.updated_at = self._now()
        return CancelJobResult(job_id=job_id, cancelled=True, state=job.state)

    def usage_report(self, job_id: str) -> UsageReport:
        job = self._get_job(job_id)
        return UsageReport(
            job_id=job.job_id,
            model_id=job.model_id,
            state=job.state,
            prompt_tokens=job.prompt_tokens,
            completion_tokens=job.completion_tokens,
            updated_at=job.updated_at,
            error_message=job.error_message,
        )

    def health_check(self) -> HealthCheckResult:
        backend_health, notes = self._backend.health()
        active_jobs = sum(1 for job in self._jobs.values() if job.state is JobState.RUNNING)
        return HealthCheckResult(
            status=backend_health,
            backend_name=self._backend.name,
            loaded_model_id=self._loaded_model.model_id if self._loaded_model else None,
            active_job_count=active_jobs,
            checked_at=self._now(),
            notes=notes,
        )

    def list_jobs(self) -> list[ActiveJob]:
        return [replace(job) for job in self._jobs.values()]

    def _get_job(self, job_id: str) -> ActiveJob:
        try:
            return self._jobs[job_id]
        except KeyError as exc:
            raise RuntimeServiceError(
                code=RuntimeErrorCode.JOB_NOT_FOUND,
                message=f"job {job_id} is unknown",
            ) from exc
