from __future__ import annotations

import threading
from contextlib import contextmanager

import httpx

from runtime_mlx.backends import EchoBackend
from runtime_mlx.server import serve


@contextmanager
def running_server():
    server = serve(EchoBackend(), host="127.0.0.1", port=0)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    host, port = server.server_address
    try:
        yield f"http://{host}:{port}"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


def test_server_load_and_generate_round_trip() -> None:
    with running_server() as base_url:
        with httpx.Client(base_url=base_url, timeout=5.0) as client:
            load = client.post("/v1/models/load", json={"model_id": "qwen3.5-35b-a3b"})
            assert load.status_code == 202
            assert load.json()["model_id"] == "qwen3.5-35b-a3b"

            generate = client.post(
                "/v1/jobs/generate",
                json={"job_id": "job-1", "prompt": "hello decentralized world", "max_output_tokens": 4},
            )
            assert generate.status_code == 200
            payload = generate.json()
            assert payload["job_id"] == "job-1"
            assert payload["prompt_tokens"] == 3
            assert payload["completion_tokens"] == 4

            usage = client.get("/v1/jobs/job-1/usage")
            assert usage.status_code == 200
            assert usage.json()["completion_tokens"] == 4


def test_server_rejects_generate_without_loaded_model() -> None:
    with running_server() as base_url:
        with httpx.Client(base_url=base_url, timeout=5.0) as client:
            response = client.post(
                "/v1/jobs/generate",
                json={"job_id": "job-1", "prompt": "hello", "max_output_tokens": 2},
            )
            assert response.status_code == 400
            assert response.json()["code"] == "model_not_loaded"
