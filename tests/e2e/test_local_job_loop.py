from __future__ import annotations

import os
import socket
import subprocess
import sys
import time
from contextlib import closing
from pathlib import Path

import httpx


ROOT = Path(__file__).resolve().parents[2]


def get_free_port() -> int:
    with closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def wait_for(url: str, timeout: float = 15.0) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            response = httpx.get(url, timeout=1.0)
            if response.status_code < 500:
                return
        except httpx.HTTPError:
            pass
        time.sleep(0.2)
    raise AssertionError(f"service did not become ready: {url}")


def start_process(command: list[str], *, cwd: Path, env: dict[str, str]) -> subprocess.Popen[str]:
    return subprocess.Popen(
        command,
        cwd=cwd,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )


def build_go_binary(*, cwd: Path, output_path: Path, env: dict[str, str]) -> None:
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


def stop_process(process: subprocess.Popen[str]) -> None:
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)


def test_local_job_loop() -> None:
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
        "PYTHONPATH": str(ROOT / "packages" / "runtime-mlx" / "src"),
        "GOCACHE": str(go_cache_dir),
        "GOMODCACHE": str(go_mod_cache_dir),
        "GOTMPDIR": str(go_tmp_dir),
    }
    bin_dir = ROOT / ".local" / "bin"
    coordinator_bin = bin_dir / "coordinator"
    providerd_bin = bin_dir / "providerd"
    build_go_binary(cwd=ROOT / "services" / "coordinator", output_path=coordinator_bin, env=base_env)
    build_go_binary(cwd=ROOT / "services" / "providerd", output_path=providerd_bin, env=base_env)

    runtime_proc = start_process(
        [sys.executable, "-m", "runtime_mlx.server", "--host", "127.0.0.1", "--port", str(runtime_port)],
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
            "DGINF_PROVIDERD_NODE_ID": "node-1",
            "DGINF_PROVIDERD_MODEL": "qwen3.5-35b-a3b",
        },
    )
    coordinator_proc = start_process(
        [str(coordinator_bin)],
        cwd=ROOT / "services" / "coordinator",
        env=base_env
        | {
            "DGINF_COORDINATOR_ADDR": f"127.0.0.1:{coordinator_port}",
        },
    )

    try:
        wait_for(f"{runtime_url}/healthz")
        wait_for(f"{provider_url}/v1/status")
        wait_for(f"{coordinator_url}/v1/models", timeout=30.0)

        with httpx.Client(timeout=5.0) as client:
            seed = client.post(
                f"{coordinator_url}/v1/dev/seed-balance",
                json={"wallet": "0xconsumer", "availableUsdc": 20_000, "withdrawableUsdc": 0},
            )
            assert seed.status_code == 200, seed.text

            register = client.post(
                f"{coordinator_url}/v1/providers/register",
                json={
                    "providerWallet": "0xprovider",
                    "nodeId": "node-1",
                    "secureEnclaveSigningPubkey": "pk",
                    "providerSessionPubkey": "sessionpk",
                    "providerSessionSignature": "sig",
                    "hardwareProfile": "M3 Max 64GB",
                    "memoryGb": 64,
                    "selectedModelId": "qwen3.5-35b-a3b",
                    "rateCard": {
                        "minJobUsdc": 100,
                        "input1mUsdc": 20_000,
                        "output1mUsdc": 40_000,
                    },
                },
            )
            assert register.status_code == 201, register.text

            load = client.post(f"{provider_url}/v1/runtime/load-selected-model")
            assert load.status_code == 202, load.text

            quote = client.post(
                f"{coordinator_url}/v1/jobs/quote",
                json={
                    "consumerWallet": "0xconsumer",
                    "modelId": "qwen3.5-35b-a3b",
                    "estimatedInputTokens": 2,
                    "maxOutputTokens": 8,
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
                json={"jobId": job_payload["jobId"], "prompt": "hello world", "maxOutputTokens": 8},
            )
            assert execute.status_code == 200, execute.text
            execute_payload = execute.json()

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
            assert complete_payload["billedUsdc"] == 100

            consumer_balance = client.get(f"{coordinator_url}/v1/balances", params={"wallet": "0xconsumer"})
            provider_balance = client.get(f"{coordinator_url}/v1/balances", params={"wallet": "0xprovider"})
            assert consumer_balance.status_code == 200, consumer_balance.text
            assert provider_balance.status_code == 200, provider_balance.text
            assert consumer_balance.json()["availableUsdc"] == 19_900
            assert consumer_balance.json()["reservedUsdc"] == 0
            assert provider_balance.json()["withdrawableUsdc"] == 100
    finally:
        stop_process(runtime_proc)
        stop_process(provider_proc)
        stop_process(coordinator_proc)
