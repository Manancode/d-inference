from __future__ import annotations

import json
import os
import subprocess
import time
from pathlib import Path

import httpx
import pytest

from .test_local_job_loop import get_free_port, start_process, stop_process, wait_for


ROOT = Path(__file__).resolve().parents[2]


def test_cli_run_against_local_loop() -> None:
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
    env = os.environ | {
        "HOME": str(home_dir),
        "UV_CACHE_DIR": str(uv_cache_dir),
        "PYTHONPATH": str(ROOT / "packages" / "runtime-mlx" / "src"),
        "GOCACHE": str(go_cache_dir),
        "GOMODCACHE": str(go_mod_cache_dir),
        "GOTMPDIR": str(go_tmp_dir),
    }

    runtime_proc = start_process(
        ["/usr/local/bin/python3", "-m", "runtime_mlx.server", "--host", "127.0.0.1", "--port", str(runtime_port)],
        cwd=ROOT,
        env=env,
    )
    coordinator_proc = start_process(
        ["go", "run", "./cmd/coordinator"],
        cwd=ROOT / "services" / "coordinator",
        env=env | {"DGINF_COORDINATOR_ADDR": f"127.0.0.1:{coordinator_port}"},
    )
    provider_proc = start_process(
        ["go", "run", "./cmd/providerd"],
        cwd=ROOT / "services" / "providerd",
        env=env
        | {
            "DGINF_PROVIDERD_ADDR": f"127.0.0.1:{provider_port}",
            "DGINF_PROVIDERD_PUBLIC_URL": provider_url,
            "DGINF_RUNTIME_URL": runtime_url,
            "DGINF_PROVIDERD_NODE_ID": "node-cli-1",
            "DGINF_PROVIDERD_MODEL": "qwen3.5-35b-a3b",
            "DGINF_PROVIDERD_WALLET": "0xprovider-cli",
            "DGINF_PROVIDERD_MIN_JOB_USDC": "100",
            "DGINF_PROVIDERD_INPUT_1M_USDC": "10000",
            "DGINF_PROVIDERD_OUTPUT_1M_USDC": "20000",
            "DGINF_COORDINATOR_URL": coordinator_url,
        },
    )

    try:
        wait_for(f"{runtime_url}/healthz", timeout=30.0)
        wait_for(f"{coordinator_url}/v1/models", timeout=30.0)
        wait_for(f"{provider_url}/v1/status", timeout=30.0)
        wait_for_provider(coordinator_url, "node-cli-1")

        subprocess.run(
            [
                "uv",
                "run",
                "--package",
                "dginf-sdk",
                "dginf",
                "--base-url",
                coordinator_url,
                "seed-balance",
                "0xconsumer-cli",
                "20000",
            ],
            cwd=ROOT,
            env=env,
            check=True,
            capture_output=True,
            text=True,
        )
        completed = subprocess.run(
            [
                "uv",
                "run",
                "--package",
                "dginf-sdk",
                "dginf",
                "--base-url",
                coordinator_url,
                "run",
                "0xconsumer-cli",
                "qwen3.5-35b-a3b",
                "hello world",
                "--max-output-tokens",
                "8",
            ],
            cwd=ROOT,
            env=env,
            check=True,
            capture_output=True,
            text=True,
        )
        payload = json.loads(completed.stdout.strip())
        assert payload["status"] == "completed"
        assert payload["output_text"]
        assert payload["billed_usdc"] == 100
    finally:
        stop_process(runtime_proc)
        stop_process(provider_proc)
        stop_process(coordinator_proc)


def wait_for_provider(coordinator_url: str, node_id: str, timeout: float = 30.0) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            response = httpx.get(f"{coordinator_url}/v1/providers", timeout=1.0)
            if response.status_code == 200:
                providers = response.json().get("providers", [])
                if any(provider.get("nodeId") == node_id for provider in providers):
                    return
        except httpx.HTTPError:
            pass
        time.sleep(0.2)
    raise AssertionError(f"provider {node_id} did not register with coordinator")
