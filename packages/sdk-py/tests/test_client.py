from __future__ import annotations

import httpx
import pytest

from dginf_sdk.client import CoordinatorClient
from dginf_sdk.errors import CoordinatorClientError
from dginf_sdk.models import JobCompletionRequest, JobCreateRequest, JobQuoteRequest, JobRunRequest, SeedBalanceRequest


def make_client(handler) -> CoordinatorClient:
    transport = httpx.MockTransport(handler)
    return CoordinatorClient("https://coordinator.test", transport=transport, token="session-token")


def test_get_auth_challenge_parses_response() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/auth/challenge"
        return httpx.Response(
            200,
            json={
                "message": "Sign in to DGInf",
                "nonce": "abc123",
                "expiresAt": "2026-03-14T12:05:00Z",
            },
        )

    client = make_client(handler)
    challenge = client.get_auth_challenge("0xabc")

    assert challenge.message == "Sign in to DGInf"
    assert challenge.nonce == "abc123"
    client.close()


def test_get_models_parses_catalog_entries() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={
                "models": [
                    {
                        "modelId": "qwen35",
                        "minimumMemoryGb": 64,
                        "description": "Default 64GB tier model",
                    }
                ]
            },
        )

    client = make_client(handler)
    catalog = client.get_models()

    assert len(catalog) == 1
    assert catalog[0].minimum_memory_gb == 64
    client.close()


def test_get_providers_parses_provider_entries() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={
                "providers": [
                    {
                        "providerWallet": "0xprovider",
                        "nodeId": "node-1",
                        "selectedModelId": "qwen35",
                        "status": "healthy",
                        "memoryGb": 64,
                        "hardwareProfile": "M3 Max",
                    }
                ]
            },
        )

    client = make_client(handler)
    providers = client.get_providers()

    assert len(providers) == 1
    assert providers[0].node_id == "node-1"
    client.close()


def test_get_job_quote_posts_expected_payload() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers["Authorization"] == "Bearer session-token"
        assert request.read() == (
            b'{"consumerWallet":"0xabc","modelId":"qwen35","estimatedInputTokens":12,'
            b'"maxOutputTokens":400}'
        )
        return httpx.Response(
            200,
            json={
                "quoteId": "quote-1",
                "providerId": "provider-1",
                "reservationUsdc": 1500000,
                "expiresAt": "2026-03-14T12:10:00Z",
                "minJobUsdc": 10000,
                "input1mUsdc": 20000000,
                "output1mUsdc": 30000000,
                "providerSigningPubkey": "signing-pub",
                "providerSessionPubkey": "session-pub",
                "providerSessionSignature": "session-sig",
            },
        )

    client = make_client(handler)
    quote = client.get_job_quote(
        JobQuoteRequest(
            consumer_wallet="0xabc",
            model_id="qwen35",
            estimated_input_tokens=12,
            max_output_tokens=400,
        )
    )

    assert quote.quote_id == "quote-1"
    assert quote.rate_card.min_job_usdc == 10000
    assert quote.rate_card.input_1m_usdc == 20000000
    assert quote.provider_session_pubkey == "session-pub"
    client.close()


def test_create_job_parses_session_descriptor() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/jobs"
        return httpx.Response(
            200,
            json={
                "jobId": "job-1",
                "sessionId": "session-1",
                "relayUrl": "quic://relay.test/session-1",
                "providerNodeId": "node-1",
                "providerSigningPubkey": "signing-pub",
                "providerSessionPubkey": "pub",
                "providerSessionSignature": "sig",
                "expiresAt": "2026-03-14T12:10:00Z",
            },
        )

    client = make_client(handler)
    session = client.create_job(
        JobCreateRequest(
            quote_id="quote-1",
            client_ephemeral_pubkey="client-pub",
            encrypted_job_envelope="ciphertext",
            max_spend_usdc=1500000,
        )
    )

    assert session.relay_url.startswith("quic://")
    assert session.provider_node_id == "node-1"
    assert session.provider_signing_pubkey == "signing-pub"
    client.close()


def test_error_response_raises_typed_exception() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(409, json={"error": "insufficient_funds"})

    client = make_client(handler)

    with pytest.raises(CoordinatorClientError) as exc:
        client.get_balances("0xabc")

    assert exc.value.status_code == 409
    assert exc.value.request_path == "/v1/balances?wallet=0xabc"
    client.close()


def test_non_object_payload_raises_exception() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json=["not", "an", "object"])

    client = make_client(handler)

    with pytest.raises(CoordinatorClientError) as exc:
        client.get_balances("0xabc")

    assert "non-object" in exc.value.message
    client.close()


def test_estimate_token_count_counts_whitespace_separated_words() -> None:
    assert CoordinatorClient.estimate_token_count("hello   decentralized inference world") == 4


def test_complete_job_posts_usage_payload() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/jobs/job-1/complete"
        assert request.read() == b'{"promptTokens":12,"completionTokens":34}'
        return httpx.Response(
            200,
            json={
                "jobId": "job-1",
                "state": "completed",
                "billedUsdc": 100,
            },
        )

    client = make_client(handler)
    status = client.complete_job("job-1", JobCompletionRequest(prompt_tokens=12, completion_tokens=34))

    assert status.status == "completed"
    assert status.billed_usdc == 100
    client.close()


def test_seed_balance_posts_payload() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/dev/seed-balance"
        assert request.read() == b'{"wallet":"0xabc","availableUsdc":5000,"withdrawableUsdc":25}'
        return httpx.Response(
            200,
            json={
                "wallet": "0xabc",
                "availableUsdc": 5000,
                "reservedUsdc": 0,
                "withdrawableUsdc": 25,
            },
        )

    client = make_client(handler)
    balance = client.seed_balance(SeedBalanceRequest(wallet="0xabc", available_usdc=5000, withdrawable_usdc=25))

    assert balance.available_usdc == 5000
    assert balance.withdrawable_usdc == 25
    client.close()


def test_run_job_posts_payload() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/jobs/job-1/run"
        assert request.read() == b'{"prompt":"hello world","maxOutputTokens":12}'
        return httpx.Response(
            200,
            json={
                "jobId": "job-1",
                "outputText": "hello",
                "promptTokens": 2,
                "completionTokens": 1,
                "billedUsdc": 100,
                "status": "completed",
            },
        )

    client = make_client(handler)
    result = client.run_job("job-1", JobRunRequest(prompt="hello world", max_output_tokens=12))

    assert result.output_text == "hello"
    assert result.billed_usdc == 100
    client.close()


def test_get_settlement_voucher_parses_response() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/jobs/job-1/settlement-voucher"
        return httpx.Response(
            200,
            json={
                "voucher": {
                    "consumer": "0xconsumer",
                    "provider": "0xprovider",
                    "amount": 100,
                    "platformFee": 0,
                    "nonce": 1,
                    "jobIdHash": "0xabc",
                    "deadline": 1234,
                },
                "signature": "0xsig",
                "signerAddress": "0xsigner",
                "verifyingChain": 8453,
                "contract": "0xcontract",
            },
        )

    client = make_client(handler)
    response = client.get_settlement_voucher("job-1")

    assert response.signature == "0xsig"
    assert response.voucher.amount == 100
    client.close()
