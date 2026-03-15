from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

import pytest


ROOT = Path(__file__).resolve().parents[2]
MODEL_PATH = Path(
    "/Users/gaj/.cache/huggingface/hub/models--mlx-community--Qwen3.5-4B-MLX-4bit/snapshots/32f3e8ecf65426fc3306969496342d504bfa13f3"
)


@pytest.mark.skipif(not MODEL_PATH.exists(), reason="local MLX Qwen model is not available")
def test_real_mlx_backend_load_and_generate() -> None:
    runtime_python = _system_python()
    env = os.environ | {
        "PYTHONPATH": os.pathsep.join(
            [
                str(ROOT / "packages" / "runtime-mlx" / "src"),
                str(ROOT / ".local" / "upstream" / "mlx-lm"),
                str(ROOT / ".local" / "upstream" / "mlx"),
            ]
        ),
        "DGINF_MLXLM_UPSTREAM": str(ROOT / ".local" / "upstream" / "mlx-lm"),
        "DGINF_MLX_UPSTREAM": str(ROOT / ".local" / "upstream" / "mlx"),
    }
    probe = """
from runtime_mlx.backends import MlxLmBackend
from runtime_mlx.models import GenerateRequest, LoadModelRequest
import json

backend = MlxLmBackend()
backend.load_model(LoadModelRequest(
    model_id="qwen3.5-4b-mlx-4bit",
    model_path=r"%s",
))
result = backend.generate(
    GenerateRequest(
        job_id="job-1",
        prompt="Say hello in one short sentence.",
        max_output_tokens=12,
    ),
    "qwen3.5-4b-mlx-4bit",
)
print(json.dumps({
    "prompt_tokens": result.prompt_tokens,
    "completion_tokens": result.completion_tokens,
    "output_text": result.output_text,
}))
""" % str(MODEL_PATH)
    completed = subprocess.run(
        [runtime_python, "-c", probe],
        cwd=ROOT,
        env=env,
        check=True,
        capture_output=True,
        text=True,
        timeout=180,
    )

    payload = json.loads(completed.stdout.strip().splitlines()[-1])
    assert payload["prompt_tokens"] > 0
    assert payload["completion_tokens"] > 0
    assert payload["output_text"].strip()


def _system_python() -> str:
    major = sys.version_info.major
    minor = sys.version_info.minor
    candidates = [
        Path("/usr/local/bin/python3"),
        Path(sys.base_prefix) / "bin" / f"python{major}.{minor}",
        Path(sys.base_prefix) / "bin" / "python3",
        Path("/opt/homebrew/opt/python@3.12/bin/python3.12"),
    ]
    for candidate in candidates:
        if candidate.exists():
            return str(candidate)
    return shutil.which("python3") or sys.executable
