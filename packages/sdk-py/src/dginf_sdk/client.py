from __future__ import annotations

from collections.abc import Mapping
from typing import Any

import httpx

from .errors import CoordinatorClientError
from .models import (
    AuthChallenge,
    AuthSession,
    BalanceSnapshot,
    JobCompletionRequest,
    JobCreateRequest,
    JobQuote,
    JobQuoteRequest,
    ProviderEntry,
    JobRunResult,
    JobRunRequest,
    JobStatus,
    ModelCatalogEntry,
    RateCard,
    SeedBalanceRequest,
    SettlementVoucher,
    SettlementVoucherResponse,
    SessionDescriptor,
    parse_datetime,
)


class CoordinatorClient:
    def __init__(
        self,
        base_url: str,
        *,
        transport: httpx.BaseTransport | None = None,
        timeout: float = 5.0,
        token: str | None = None,
    ) -> None:
        self._client = httpx.Client(
            base_url=base_url,
            timeout=timeout,
            transport=transport,
            headers=self._build_headers(token),
        )

    @staticmethod
    def estimate_token_count(prompt: str) -> int:
        return len([part for part in prompt.split() if part])

    def close(self) -> None:
        self._client.close()

    def get_auth_challenge(self, wallet: str, *, chain_id: int = 8453) -> AuthChallenge:
        payload = self._request("POST", "/v1/auth/challenge", json={"wallet": wallet, "chainId": chain_id})
        return AuthChallenge(
            message=payload["message"],
            nonce=payload["nonce"],
            expires_at=parse_datetime(payload["expiresAt"]),
        )

    def verify_auth(self, wallet: str, signature: str, message: str) -> AuthSession:
        payload = self._request(
            "POST",
            "/v1/auth/verify",
            json={"wallet": wallet, "signature": signature, "message": message},
        )
        return AuthSession(
            session_token=payload["sessionToken"],
            wallet=payload["wallet"],
        )

    def get_models(self) -> list[ModelCatalogEntry]:
        payload = self._request("GET", "/v1/models")
        return [self._parse_catalog_entry(entry) for entry in payload["models"]]

    def get_providers(self) -> list[ProviderEntry]:
        payload = self._request("GET", "/v1/providers")
        return [self._parse_provider_entry(entry) for entry in payload["providers"]]

    def get_balances(self, wallet: str) -> BalanceSnapshot:
        payload = self._request("GET", f"/v1/balances?wallet={wallet}")
        return BalanceSnapshot(
            wallet=payload["wallet"],
            available_usdc=payload["availableUsdc"],
            reserved_usdc=payload["reservedUsdc"],
            withdrawable_usdc=payload["withdrawableUsdc"],
        )

    def get_job_quote(self, request: JobQuoteRequest) -> JobQuote:
        payload = self._request("POST", "/v1/jobs/quote", json=request.to_payload())
        return JobQuote(
            quote_id=payload["quoteId"],
            provider_id=payload["providerId"],
            reservation_usdc=payload["reservationUsdc"],
            expires_at=parse_datetime(payload["expiresAt"]),
            rate_card=self._parse_rate_card(payload),
            provider_signing_pubkey=payload["providerSigningPubkey"],
            provider_session_pubkey=payload["providerSessionPubkey"],
            provider_session_signature=payload["providerSessionSignature"],
        )

    def create_job(self, request: JobCreateRequest) -> SessionDescriptor:
        payload = self._request("POST", "/v1/jobs", json=request.to_payload())
        return SessionDescriptor(
            job_id=payload["jobId"],
            session_id=payload["sessionId"],
            relay_url=payload["relayUrl"],
            provider_node_id=payload["providerNodeId"],
            provider_signing_pubkey=payload["providerSigningPubkey"],
            provider_session_pubkey=payload["providerSessionPubkey"],
            provider_session_signature=payload["providerSessionSignature"],
            expires_at=parse_datetime(payload["expiresAt"]),
        )

    def get_job_status(self, job_id: str) -> JobStatus:
        payload = self._request("GET", f"/v1/jobs/{job_id}")
        return JobStatus(
            job_id=payload["jobId"],
            status=payload.get("status", payload.get("state", "unknown")),
            billed_usdc=payload.get("reservedUsdc"),
        )

    def cancel_job(self, job_id: str) -> JobStatus:
        self._request("POST", f"/v1/jobs/{job_id}/cancel")
        return JobStatus(
            job_id=job_id,
            status="cancelled",
            billed_usdc=None,
        )

    def complete_job(self, job_id: str, request: JobCompletionRequest) -> JobStatus:
        payload = self._request("POST", f"/v1/jobs/{job_id}/complete", json=request.to_payload())
        return JobStatus(
            job_id=payload["jobId"],
            status=payload.get("state", "completed"),
            billed_usdc=payload.get("billedUsdc"),
        )

    def run_job(self, job_id: str, request: JobRunRequest) -> JobRunResult:
        payload = self._request("POST", f"/v1/jobs/{job_id}/run", json=request.to_payload())
        return JobRunResult(
            job_id=payload["jobId"],
            output_text=payload["outputText"],
            prompt_tokens=payload["promptTokens"],
            completion_tokens=payload["completionTokens"],
            billed_usdc=payload["billedUsdc"],
            status=payload["status"],
        )

    def get_settlement_voucher(self, job_id: str) -> SettlementVoucherResponse:
        payload = self._request("GET", f"/v1/jobs/{job_id}/settlement-voucher")
        voucher = payload["voucher"]
        return SettlementVoucherResponse(
            voucher=SettlementVoucher(
                consumer=voucher["consumer"],
                provider=voucher["provider"],
                amount=voucher["amount"],
                platform_fee=voucher["platformFee"],
                nonce=voucher["nonce"],
                job_id_hash=voucher["jobIdHash"],
                deadline=voucher["deadline"],
            ),
            signature=payload["signature"],
            signer_address=payload["signerAddress"],
            verifying_chain=payload["verifyingChain"],
            contract=payload["contract"],
        )

    def seed_balance(self, request: SeedBalanceRequest) -> BalanceSnapshot:
        payload = self._request("POST", "/v1/dev/seed-balance", json=request.to_payload())
        return BalanceSnapshot(
            wallet=payload["wallet"],
            available_usdc=payload["availableUsdc"],
            reserved_usdc=payload["reservedUsdc"],
            withdrawable_usdc=payload["withdrawableUsdc"],
        )

    def _request(self, method: str, path: str, **kwargs: Any) -> Mapping[str, Any]:
        response = self._client.request(method, path, **kwargs)
        if response.status_code >= 400:
            raise CoordinatorClientError(
                message="coordinator request failed",
                status_code=response.status_code,
                response_body=self._maybe_json(response),
                request_path=path,
            )

        payload = self._maybe_json(response)
        if not isinstance(payload, Mapping):
            raise CoordinatorClientError(
                message="coordinator returned a non-object payload",
                status_code=response.status_code,
                response_body=payload,
                request_path=path,
            )
        return payload

    @staticmethod
    def _build_headers(token: str | None) -> dict[str, str]:
        if not token:
            return {}
        return {"Authorization": f"Bearer {token}"}

    @staticmethod
    def _maybe_json(response: httpx.Response) -> Any:
        try:
            return response.json()
        except ValueError:
            return response.text

    @staticmethod
    def _parse_rate_card(payload: Mapping[str, Any]) -> RateCard:
        return RateCard(
            min_job_usdc=int(payload["minJobUsdc"]),
            input_1m_usdc=int(payload["input1mUsdc"]),
            output_1m_usdc=int(payload["output1mUsdc"]),
        )

    def _parse_catalog_entry(self, payload: Mapping[str, Any]) -> ModelCatalogEntry:
        return ModelCatalogEntry(
            model_id=payload["modelId"],
            minimum_memory_gb=int(payload["minimumMemoryGb"]),
            description=payload["description"],
        )

    def _parse_provider_entry(self, payload: Mapping[str, Any]) -> ProviderEntry:
        return ProviderEntry(
            provider_wallet=payload["providerWallet"],
            node_id=payload["nodeId"],
            selected_model_id=payload["selectedModelId"],
            status=payload["status"],
            memory_gb=int(payload["memoryGb"]),
            hardware_profile=payload["hardwareProfile"],
            posture=payload.get("posture"),
        )
