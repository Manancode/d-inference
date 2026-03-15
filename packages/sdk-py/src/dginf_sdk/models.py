from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime


def parse_datetime(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(UTC)


@dataclass(slots=True)
class AuthChallenge:
    message: str
    nonce: str
    expires_at: datetime


@dataclass(slots=True)
class AuthSession:
    session_token: str
    wallet: str


@dataclass(slots=True)
class RateCard:
    min_job_usdc: int
    input_1m_usdc: int
    output_1m_usdc: int


@dataclass(slots=True)
class ModelCatalogEntry:
    model_id: str
    minimum_memory_gb: int
    description: str


@dataclass(slots=True)
class ProviderEntry:
    provider_wallet: str
    node_id: str
    selected_model_id: str
    status: str
    memory_gb: int
    hardware_profile: str
    posture: dict[str, object] | None = None


@dataclass(slots=True)
class JobQuoteRequest:
    consumer_wallet: str
    model_id: str
    estimated_input_tokens: int
    max_output_tokens: int

    def to_payload(self) -> dict[str, object]:
        return {
            "consumerWallet": self.consumer_wallet,
            "modelId": self.model_id,
            "estimatedInputTokens": self.estimated_input_tokens,
            "maxOutputTokens": self.max_output_tokens,
        }


@dataclass(slots=True)
class JobQuote:
    quote_id: str
    provider_id: str
    reservation_usdc: int
    expires_at: datetime
    rate_card: RateCard
    provider_signing_pubkey: str
    provider_session_pubkey: str
    provider_session_signature: str


@dataclass(slots=True)
class JobCreateRequest:
    quote_id: str
    client_ephemeral_pubkey: str
    encrypted_job_envelope: str
    max_spend_usdc: int

    def to_payload(self) -> dict[str, object]:
        return {
            "quoteId": self.quote_id,
            "clientEphemeralPubkey": self.client_ephemeral_pubkey,
            "encryptedJobEnvelope": self.encrypted_job_envelope,
            "maxSpendUsdc": self.max_spend_usdc,
        }


@dataclass(slots=True)
class SessionDescriptor:
    job_id: str
    session_id: str
    relay_url: str
    provider_node_id: str
    provider_signing_pubkey: str
    provider_session_pubkey: str
    provider_session_signature: str
    expires_at: datetime


@dataclass(slots=True)
class JobStatus:
    job_id: str
    status: str
    billed_usdc: int | None = None


@dataclass(slots=True)
class JobRunRequest:
    prompt: str
    max_output_tokens: int

    def to_payload(self) -> dict[str, object]:
        return {
            "prompt": self.prompt,
            "maxOutputTokens": self.max_output_tokens,
        }


@dataclass(slots=True)
class JobRunResult:
    job_id: str
    output_text: str
    prompt_tokens: int
    completion_tokens: int
    billed_usdc: int
    status: str


@dataclass(slots=True)
class SettlementVoucher:
    consumer: str
    provider: str
    amount: int
    platform_fee: int
    nonce: int
    job_id_hash: str
    deadline: int


@dataclass(slots=True)
class SettlementVoucherResponse:
    voucher: SettlementVoucher
    signature: str
    signer_address: str
    verifying_chain: int
    contract: str


@dataclass(slots=True)
class BalanceSnapshot:
    wallet: str
    available_usdc: int
    reserved_usdc: int
    withdrawable_usdc: int


@dataclass(slots=True)
class JobCompletionRequest:
    prompt_tokens: int
    completion_tokens: int

    def to_payload(self) -> dict[str, object]:
        return {
            "promptTokens": self.prompt_tokens,
            "completionTokens": self.completion_tokens,
        }


@dataclass(slots=True)
class SeedBalanceRequest:
    wallet: str
    available_usdc: int
    withdrawable_usdc: int = 0

    def to_payload(self) -> dict[str, object]:
        return {
            "wallet": self.wallet,
            "availableUsdc": self.available_usdc,
            "withdrawableUsdc": self.withdrawable_usdc,
        }
