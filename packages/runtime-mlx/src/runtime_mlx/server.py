from __future__ import annotations

import argparse
import json
import os
from dataclasses import asdict
from datetime import datetime
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Callable
from urllib.parse import urlparse

from .backends import build_backend
from .errors import RuntimeServiceError
from .models import GenerateRequest, LoadModelRequest
from .service import RuntimeBackend, RuntimeService


def create_handler(service: RuntimeService) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:  # noqa: N802
            parsed = urlparse(self.path)
            if parsed.path == "/healthz":
                self._write_json(HTTPStatus.OK, {"status": "ok"})
                return
            if parsed.path == "/v1/health":
                self._write_json(HTTPStatus.OK, _to_jsonable(service.health_check()))
                return
            if parsed.path.startswith("/v1/jobs/") and parsed.path.endswith("/usage"):
                job_id = parsed.path.removeprefix("/v1/jobs/").removesuffix("/usage").strip("/")
                try:
                    report = service.usage_report(job_id)
                except RuntimeServiceError as exc:
                    self._write_runtime_error(exc)
                    return
                self._write_json(HTTPStatus.OK, _to_jsonable(report))
                return
            self._write_json(HTTPStatus.NOT_FOUND, {"error": "not_found"})

        def do_POST(self) -> None:  # noqa: N802
            parsed = urlparse(self.path)
            if parsed.path == "/v1/models/load":
                payload = self._decode_json()
                if payload is None:
                    return
                try:
                    result = service.load_model(
                        LoadModelRequest(
                            model_id=payload["model_id"],
                            model_path=payload.get("model_path", payload["model_id"]),
                            revision=payload.get("revision"),
                        )
                    )
                except KeyError:
                    self._write_json(HTTPStatus.BAD_REQUEST, {"error": "model_id is required"})
                    return
                except RuntimeServiceError as exc:
                    self._write_runtime_error(exc)
                    return
                self._write_json(HTTPStatus.ACCEPTED, _to_jsonable(result))
                return
            if parsed.path == "/v1/jobs/generate":
                payload = self._decode_json()
                if payload is None:
                    return
                try:
                    result = service.generate(
                        GenerateRequest(
                            job_id=payload["job_id"],
                            prompt=payload["prompt"],
                            max_output_tokens=int(payload["max_output_tokens"]),
                        )
                    )
                except KeyError as exc:
                    self._write_json(HTTPStatus.BAD_REQUEST, {"error": f"missing field: {exc.args[0]}"})
                    return
                except RuntimeServiceError as exc:
                    self._write_runtime_error(exc)
                    return
                self._write_json(HTTPStatus.OK, _to_jsonable(result))
                return
            if parsed.path.startswith("/v1/jobs/") and parsed.path.endswith("/cancel"):
                job_id = parsed.path.removeprefix("/v1/jobs/").removesuffix("/cancel").strip("/")
                try:
                    result = service.cancel_job(job_id)
                except RuntimeServiceError as exc:
                    self._write_runtime_error(exc)
                    return
                self._write_json(HTTPStatus.OK, _to_jsonable(result))
                return
            self._write_json(HTTPStatus.NOT_FOUND, {"error": "not_found"})

        def log_message(self, format: str, *args: object) -> None:  # noqa: A003
            return

        def _decode_json(self) -> dict[str, object] | None:
            try:
                length = int(self.headers.get("Content-Length", "0"))
                raw = self.rfile.read(length)
                return json.loads(raw or b"{}")
            except json.JSONDecodeError:
                self._write_json(HTTPStatus.BAD_REQUEST, {"error": "invalid_json"})
                return None

        def _write_runtime_error(self, exc: RuntimeServiceError) -> None:
            status = HTTPStatus.CONFLICT if exc.retryable else HTTPStatus.BAD_REQUEST
            self._write_json(status, {"code": exc.code, "message": exc.message})

        def _write_json(self, status: HTTPStatus, payload: dict[str, object]) -> None:
            body = json.dumps(payload).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return Handler


def serve(backend: RuntimeBackend, host: str = "127.0.0.1", port: int = 8089) -> ThreadingHTTPServer:
    server = ThreadingHTTPServer((host, port), create_handler(RuntimeService(backend)))
    return server


def _to_jsonable(value: object) -> dict[str, object]:
    if hasattr(value, "__dataclass_fields__"):
        return _convert_datetimes(asdict(value))
    raise TypeError(f"unsupported value: {type(value)!r}")


def _convert_datetimes(payload: dict[str, object]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in payload.items():
        if isinstance(value, datetime):
            result[key] = value.isoformat()
        else:
            result[key] = value
    return result

def main() -> None:
    parser = argparse.ArgumentParser(description="Run the DGInf runtime-mlx HTTP server.")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8089)
    parser.add_argument("--backend", choices=["echo", "mlx-lm"], default=os.environ.get("DGINF_RUNTIME_BACKEND", "echo"))
    args = parser.parse_args()

    server = serve(build_backend(args.backend), host=args.host, port=args.port)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
