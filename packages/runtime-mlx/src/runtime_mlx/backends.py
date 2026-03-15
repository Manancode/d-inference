from __future__ import annotations

import os
import sys
import threading
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

from .errors import RuntimeErrorCode, RuntimeServiceError
from .models import GenerateRequest, GenerateResult, JobState, LoadModelRequest, WorkerHealth


def _ensure_upstream_paths() -> None:
    candidates = [
        Path(os.environ.get("DGINF_MLXLM_UPSTREAM", "")).expanduser(),
        Path(os.environ.get("DGINF_MLX_UPSTREAM", "")).expanduser(),
    ]

    root = Path(__file__).resolve().parents[4]
    defaults = [
        root / ".local" / "upstream" / "mlx-lm",
        root / ".local" / "upstream" / "mlx",
    ]

    for candidate in [*candidates, *defaults]:
        if candidate and str(candidate) not in sys.path and candidate.exists():
            sys.path.insert(0, str(candidate))


class EchoBackend:
    name = "echo-backend"

    def __init__(self) -> None:
        self._loaded: str | None = None
        self._cancelled: set[str] = set()

    def load_model(self, request: LoadModelRequest) -> None:
        self._loaded = request.model_id

    def cancel_job(self, job_id: str) -> bool:
        self._cancelled.add(job_id)
        return True

    def health(self) -> tuple[WorkerHealth, list[str]]:
        note = f"loaded={self._loaded}" if self._loaded else "no-model"
        return WorkerHealth.HEALTHY, [note]

    def generate(self, request: GenerateRequest, model_id: str) -> GenerateResult:
        output_words = [f"{model_id}:{idx}" for idx in range(min(request.max_output_tokens, 8))]
        output_text = " ".join(output_words)
        return GenerateResult(
            job_id=request.job_id,
            model_id=model_id,
            output_text=output_text,
            prompt_tokens=len([part for part in request.prompt.split() if part]),
            completion_tokens=len(output_words),
            state=JobState.COMPLETED,
            finished_at=datetime.now(tz=UTC),
        )


@dataclass
class _LoadedModel:
    model_id: str
    model_path: str
    model: object
    tokenizer: object


class MlxLmBackend:
    name = "mlx-lm"

    def __init__(self) -> None:
        self._loaded: _LoadedModel | None = None
        self._cancelled: set[str] = set()
        self._lock = threading.RLock()

    def load_model(self, request: LoadModelRequest) -> None:
        _ensure_upstream_paths()
        from mlx_lm import load

        model_path = request.model_path or request.model_id
        model, tokenizer = load(model_path)
        with self._lock:
            self._loaded = _LoadedModel(
                model_id=request.model_id,
                model_path=model_path,
                model=model,
                tokenizer=tokenizer,
            )
            self._cancelled.clear()

    def cancel_job(self, job_id: str) -> bool:
        with self._lock:
            self._cancelled.add(job_id)
        return True

    def health(self) -> tuple[WorkerHealth, list[str]]:
        with self._lock:
            loaded = self._loaded
        if loaded is None:
            return WorkerHealth.HEALTHY, ["no-model"]
        return WorkerHealth.HEALTHY, [f"loaded={loaded.model_id}", f"path={loaded.model_path}"]

    def generate(self, request: GenerateRequest, model_id: str) -> GenerateResult:
        _ensure_upstream_paths()
        from mlx_lm.generate import stream_generate

        with self._lock:
            loaded = self._loaded
        if loaded is None:
            raise RuntimeServiceError(
                code=RuntimeErrorCode.MODEL_NOT_LOADED,
                message="no model loaded",
            )

        segments: list[str] = []
        prompt_tokens = 0
        completion_tokens = 0
        for response in stream_generate(
            loaded.model,
            loaded.tokenizer,
            request.prompt,
            max_tokens=request.max_output_tokens,
        ):
            with self._lock:
                if request.job_id in self._cancelled:
                    self._cancelled.discard(request.job_id)
                    raise RuntimeServiceError(
                        code=RuntimeErrorCode.JOB_NOT_CANCELLABLE,
                        message=f"job {request.job_id} cancelled",
                    )
            segments.append(response.text)
            prompt_tokens = int(response.prompt_tokens)
            completion_tokens = int(response.generation_tokens)

        return GenerateResult(
            job_id=request.job_id,
            model_id=model_id,
            output_text="".join(segments),
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
            state=JobState.COMPLETED,
            finished_at=datetime.now(tz=UTC),
        )


def build_backend(kind: str):
    normalized = kind.lower()
    if normalized == "mlx-lm":
        return MlxLmBackend()
    return EchoBackend()
