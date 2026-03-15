from __future__ import annotations

from datetime import UTC, datetime

import pytest

from runtime_mlx.errors import RuntimeErrorCode, RuntimeServiceError
from runtime_mlx.models import JobState, LoadModelRequest, WorkerHealth
from runtime_mlx.service import RuntimeService


class FakeBackend:
    name = "fake-mlx"

    def __init__(self) -> None:
        self.loaded: list[str] = []
        self.cancelled: list[str] = []
        self._health = WorkerHealth.HEALTHY
        self._notes: list[str] = ["ready"]

    def load_model(self, request: LoadModelRequest) -> None:
        self.loaded.append(request.model_id)

    def cancel_job(self, job_id: str) -> bool:
        self.cancelled.append(job_id)
        return True

    def health(self) -> tuple[WorkerHealth, list[str]]:
        return self._health, list(self._notes)


def fixed_now() -> datetime:
    return datetime(2026, 3, 14, 12, 0, tzinfo=UTC)


def test_load_model_clears_previous_jobs() -> None:
    backend = FakeBackend()
    service = RuntimeService(backend, now=fixed_now)

    result = service.load_model(LoadModelRequest(model_id="qwen35", model_path="/tmp/model"))
    service.register_job("job-1")

    new_result = service.load_model(LoadModelRequest(model_id="qwen122", model_path="/tmp/model2"))

    assert result.model_id == "qwen35"
    assert new_result.model_id == "qwen122"
    assert backend.loaded == ["qwen35", "qwen122"]
    assert service.list_jobs() == []


def test_register_job_requires_loaded_model() -> None:
    service = RuntimeService(FakeBackend(), now=fixed_now)

    with pytest.raises(RuntimeServiceError) as exc:
        service.register_job("job-1")

    assert exc.value.code is RuntimeErrorCode.MODEL_NOT_LOADED


def test_usage_report_tracks_runtime_updates() -> None:
    service = RuntimeService(FakeBackend(), now=fixed_now)
    service.load_model(LoadModelRequest(model_id="qwen35", model_path="/tmp/model"))
    service.register_job("job-1")

    service.update_usage("job-1", prompt_tokens=12, completion_tokens=21, state=JobState.COMPLETED)
    report = service.usage_report("job-1")

    assert report.prompt_tokens == 12
    assert report.completion_tokens == 21
    assert report.state is JobState.COMPLETED


def test_cancel_job_marks_state_and_delegates_to_backend() -> None:
    backend = FakeBackend()
    service = RuntimeService(backend, now=fixed_now)
    service.load_model(LoadModelRequest(model_id="qwen35", model_path="/tmp/model"))
    service.register_job("job-1")

    result = service.cancel_job("job-1")

    assert result.cancelled is True
    assert result.state is JobState.CANCELLED
    assert backend.cancelled == ["job-1"]


def test_cancel_job_rejects_completed_jobs() -> None:
    service = RuntimeService(FakeBackend(), now=fixed_now)
    service.load_model(LoadModelRequest(model_id="qwen35", model_path="/tmp/model"))
    service.register_job("job-1")
    service.update_usage("job-1", state=JobState.COMPLETED)

    with pytest.raises(RuntimeServiceError) as exc:
        service.cancel_job("job-1")

    assert exc.value.code is RuntimeErrorCode.JOB_NOT_CANCELLABLE


def test_health_check_surfaces_backend_and_loaded_model() -> None:
    service = RuntimeService(FakeBackend(), now=fixed_now)
    service.load_model(LoadModelRequest(model_id="qwen35", model_path="/tmp/model"))
    service.register_job("job-1")

    health = service.health_check()

    assert health.status is WorkerHealth.HEALTHY
    assert health.loaded_model_id == "qwen35"
    assert health.active_job_count == 1
    assert health.notes == ["ready"]

