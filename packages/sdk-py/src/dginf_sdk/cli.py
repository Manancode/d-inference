from __future__ import annotations

import argparse
from dataclasses import asdict
import json
import sys

from .client import CoordinatorClient
from .crypto import encrypt_job_envelope, verify_provider_session_key
from .models import JobCompletionRequest, JobCreateRequest, JobQuoteRequest, JobRunRequest, SeedBalanceRequest


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="DGInf CLI")
    parser.add_argument("--base-url", default="http://127.0.0.1:8080")

    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("models", help="List models")
    subparsers.add_parser("providers", help="List registered providers")

    balances = subparsers.add_parser("balances", help="Get wallet balances")
    balances.add_argument("wallet")

    seed = subparsers.add_parser("seed-balance", help="Seed a wallet balance in dev mode")
    seed.add_argument("wallet")
    seed.add_argument("available_usdc", type=int)
    seed.add_argument("--withdrawable-usdc", type=int, default=0)

    quote = subparsers.add_parser("quote", help="Get a job quote")
    quote.add_argument("wallet")
    quote.add_argument("model_id")
    quote.add_argument("prompt")
    quote.add_argument("--max-output-tokens", type=int, default=128)

    complete = subparsers.add_parser("complete-job", help="Complete a job with usage")
    complete.add_argument("job_id")
    complete.add_argument("prompt_tokens", type=int)
    complete.add_argument("completion_tokens", type=int)

    voucher = subparsers.add_parser("settlement-voucher", help="Fetch a settlement voucher for a completed job")
    voucher.add_argument("job_id")

    run = subparsers.add_parser("run", help="Quote, create, and run a job in one command")
    run.add_argument("wallet")
    run.add_argument("model_id")
    run.add_argument("prompt")
    run.add_argument("--max-output-tokens", type=int, default=128)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    client = CoordinatorClient(args.base_url)

    try:
        if args.command == "models":
            print(json.dumps([asdict(entry) for entry in client.get_models()], default=str))
        elif args.command == "providers":
            print(json.dumps([asdict(entry) for entry in client.get_providers()], default=str))
        elif args.command == "balances":
            print(json.dumps(asdict(client.get_balances(args.wallet)), default=str))
        elif args.command == "seed-balance":
            result = client.seed_balance(
                SeedBalanceRequest(
                    wallet=args.wallet,
                    available_usdc=args.available_usdc,
                    withdrawable_usdc=args.withdrawable_usdc,
                )
            )
            print(json.dumps(asdict(result), default=str))
        elif args.command == "quote":
            estimated = CoordinatorClient.estimate_token_count(args.prompt)
            result = client.get_job_quote(
                JobQuoteRequest(
                    consumer_wallet=args.wallet,
                    model_id=args.model_id,
                    estimated_input_tokens=estimated,
                    max_output_tokens=args.max_output_tokens,
                )
            )
            print(json.dumps(_job_quote_to_dict(result), default=str))
        elif args.command == "complete-job":
            result = client.complete_job(
                args.job_id,
                JobCompletionRequest(
                    prompt_tokens=args.prompt_tokens,
                    completion_tokens=args.completion_tokens,
                ),
            )
            print(json.dumps(asdict(result), default=str))
        elif args.command == "settlement-voucher":
            print(json.dumps(asdict(client.get_settlement_voucher(args.job_id)), default=str))
        elif args.command == "run":
            estimated = CoordinatorClient.estimate_token_count(args.prompt)
            quote = client.get_job_quote(
                JobQuoteRequest(
                    consumer_wallet=args.wallet,
                    model_id=args.model_id,
                    estimated_input_tokens=estimated,
                    max_output_tokens=args.max_output_tokens,
                )
            )
            verify_provider_session_key(
                quote.provider_signing_pubkey,
                quote.provider_session_pubkey,
                quote.provider_session_signature,
            )
            envelope = encrypt_job_envelope(
                quote.provider_session_pubkey,
                args.prompt,
                args.max_output_tokens,
            )
            session = client.create_job(
                JobCreateRequest(
                    quote_id=quote.quote_id,
                    client_ephemeral_pubkey=envelope.ephemeral_pubkey,
                    encrypted_job_envelope=envelope.to_json(),
                    max_spend_usdc=quote.reservation_usdc,
                )
            )
            result = client.run_job(
                session.job_id,
                JobRunRequest(prompt="", max_output_tokens=args.max_output_tokens),
            )
            print(json.dumps(asdict(result), default=str))
        return 0
    finally:
        client.close()


def _job_quote_to_dict(result) -> dict[str, object]:
    return {
        "quote_id": result.quote_id,
        "provider_id": result.provider_id,
        "reservation_usdc": result.reservation_usdc,
        "expires_at": result.expires_at.isoformat(),
        "rate_card": asdict(result.rate_card),
    }


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
