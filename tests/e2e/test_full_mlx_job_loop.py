from __future__ import annotations

import os
import shutil
import subprocess
import sys
from pathlib import Path

import httpx
import pytest

from .test_local_job_loop import get_free_port, start_process, stop_process, wait_for


ROOT = Path(__file__).resolve().parents[2]
MODEL_PATH = Path(
    "/Users/gaj/.cache/huggingface/hub/models--mlx-community--Qwen3.5-4B-MLX-4bit/snapshots/32f3e8ecf65426fc3306969496342d504bfa13f3"
)


@pytest.mark.skipif(not MODEL_PATH.exists(), reason="local MLX Qwen model is not available")
def test_full_job_loop_with_real_mlx_model() -> None:
    runtime_port = get_free_port()
    provider_port = get_free_port()
    coordinator_port = get_free_port()
    runtime_url = f"http://127.0.0.1:{runtime_port}"
    provider_url = f"http://127.0.0.1:{provider_port}"
    coordinator_url = f"http://127.0.0.1:{coordinator_port}"

    home_dir = ROOT / ".home"
    uv_cache_dir = ROOT / ".local" / "uv-cache"
    go_cache_dir = ROOT / ".local" / "go-build-cache"
    go_mod_cache_dir = ROOT / ".local" / "gomodcache"
    go_tmp_dir = ROOT / ".local" / "go-tmp"
    go_cache_dir.mkdir(parents=True, exist_ok=True)
    go_mod_cache_dir.mkdir(parents=True, exist_ok=True)
    go_tmp_dir.mkdir(parents=True, exist_ok=True)

    base_env = os.environ | {
        "HOME": str(home_dir),
        "UV_CACHE_DIR": str(uv_cache_dir),
        "PYTHONPATH": os.pathsep.join(
            [
                str(ROOT / "packages" / "runtime-mlx" / "src"),
                str(ROOT / ".local" / "upstream" / "mlx-lm"),
                str(ROOT / ".local" / "upstream" / "mlx"),
            ]
        ),
        "DGINF_RUNTIME_BACKEND": "mlx-lm",
        "DGINF_MLXLM_UPSTREAM": str(ROOT / ".local" / "upstream" / "mlx-lm"),
        "DGINF_MLX_UPSTREAM": str(ROOT / ".local" / "upstream" / "mlx"),
        "GOCACHE": str(go_cache_dir),
        "GOMODCACHE": str(go_mod_cache_dir),
        "GOTMPDIR": str(go_tmp_dir),
    }

    bin_dir = ROOT / ".local" / "bin"
    coordinator_bin = bin_dir / "coordinator"
    providerd_bin = bin_dir / "providerd"
    _build_go_binary(ROOT / "services" / "coordinator", coordinator_bin, base_env)
    _build_go_binary(ROOT / "services" / "providerd", providerd_bin, base_env)

    runtime_proc = start_process(
        [
            _system_python(),
            "-m",
            "runtime_mlx.server",
            "--host",
            "127.0.0.1",
            "--port",
            str(runtime_port),
            "--backend",
            "mlx-lm",
        ],
        cwd=ROOT,
        env=base_env,
    )
    provider_proc = start_process(
        [str(providerd_bin)],
        cwd=ROOT / "services" / "providerd",
        env=base_env
        | {
            "DGINF_PROVIDERD_ADDR": f"127.0.0.1:{provider_port}",
            "DGINF_RUNTIME_URL": runtime_url,
            "DGINF_PROVIDERD_NODE_ID": "node-real-1",
            "DGINF_PROVIDERD_MODEL": "qwen3.5-4b-mlx-4bit",
        },
    )
    coordinator_proc = start_process(
        [str(coordinator_bin)],
        cwd=ROOT / "services" / "coordinator",
        env=base_env | {"DGINF_COORDINATOR_ADDR": f"127.0.0.1:{coordinator_port}"},
    )

    try:
        wait_for(f"{runtime_url}/healthz", timeout=30.0)
        wait_for(f"{provider_url}/v1/status", timeout=30.0)
        wait_for(f"{coordinator_url}/v1/models", timeout=30.0)

        with httpx.Client(timeout=120.0) as client:
            load = client.post(
                f"{runtime_url}/v1/models/load",
                json={"model_id": "qwen3.5-4b-mlx-4bit", "model_path": str(MODEL_PATH)},
            )
            assert load.status_code == 202, load.text

            seed = client.post(
                f"{coordinator_url}/v1/dev/seed-balance",
                json={"wallet": "0xconsumer-real", "availableUsdc": 20_000, "withdrawableUsdc": 0},
            )
            assert seed.status_code == 200, seed.text

            register = client.post(
                f"{coordinator_url}/v1/providers/register",
                json={
                    "providerWallet": "0xprovider-real",
                    "nodeId": "node-real-1",
                    "secureEnclaveSigningPubkey": "pk",
                    "providerSessionPubkey": "sessionpk",
                    "providerSessionSignature": "sig",
                    "hardwareProfile": "M3 Max 64GB",
                    "memoryGb": 64,
                    "selectedModelId": "qwen3.5-4b-mlx-4bit",
                    "rateCard": {
                        "minJobUsdc": 100,
                        "input1mUsdc": 10_000,
                        "output1mUsdc": 20_000,
                    },
                },
            )
            assert register.status_code == 201, register.text

            quote = client.post(
                f"{coordinator_url}/v1/jobs/quote",
                json={
                    "consumerWallet": "0xconsumer-real",
                    "modelId": "qwen3.5-4b-mlx-4bit",
                    "estimatedInputTokens": 12,
                    "maxOutputTokens": 16,
                },
            )
            assert quote.status_code == 200, quote.text
            quote_payload = quote.json()

            job = client.post(
                f"{coordinator_url}/v1/jobs",
                json={
                    "quoteId": quote_payload["quoteId"],
                    "clientEphemeralPubkey": "client-pub",
                    "encryptedJobEnvelope": "ciphertext",
                    "maxSpendUsdc": quote_payload["reservationUsdc"],
                },
            )
            assert job.status_code == 201, job.text
            job_payload = job.json()

            execute = client.post(
                f"{provider_url}/v1/jobs/execute",
                json={
                    "jobId": job_payload["jobId"],
                    "prompt": "Say hello in one short sentence.",
                    "maxOutputTokens": 12,
                },
            )
            assert execute.status_code == 200, execute.text
            execute_payload = execute.json()
            assert execute_payload["outputText"].strip()

            complete = client.post(
                f"{coordinator_url}/v1/jobs/{job_payload['jobId']}/complete",
                json={
                    "promptTokens": execute_payload["promptTokens"],
                    "completionTokens": execute_payload["completionTokens"],
                },
            )
            assert complete.status_code == 200, complete.text
            complete_payload = complete.json()
            assert complete_payload["state"] == "completed"
            assert complete_payload["completionTokens"] > 0

            provider_balance = client.get(f"{coordinator_url}/v1/balances", params={"wallet": "0xprovider-real"})
            assert provider_balance.status_code == 200, provider_balance.text
            assert provider_balance.json()["withdrawableUsdc"] > 0
    finally:
        stop_process(runtime_proc)
        stop_process(provider_proc)
        stop_process(coordinator_proc)


def _build_go_binary(cwd: Path, output_path: Path, env: dict[str, str]) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        ["go", "build", "-o", str(output_path), "./cmd/" + cwd.name],
        cwd=cwd,
        env=env,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )


def _system_python() -> str:
    candidates = [
        Path("/usr/local/bin/python3"),
        Path(sys.base_prefix) / "bin" / f"python{sys.version_info.major}.{sys.version_info.minor}",
        Path(sys.base_prefix) / "bin" / "python3",
    ]
    for candidate in candidates:
        if candidate.exists():
            return str(candidate)
    return shutil.which("python3") or sys.executable
