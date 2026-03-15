# DGInf

DGInf is a Mac-first decentralized private inference network. This branch contains
the initial implementation work for the provider app, provider daemon,
coordinator, runtime adapters, SDK, and onchain settlement contract.

## Layout

- `apps/provider-mac`: SwiftUI shell plus the Secure Enclave helper tool.
- `services/providerd`: macOS daemon that owns node lifecycle.
- `services/coordinator`: control plane, quote engine, and run/settlement flow.
- `packages/runtime-mlx`: Python runtime adapter around `mlx-lm`.
- `packages/sdk-py`: Python SDK and CLI surface.
- `packages/contracts`: Base USDC ledger contract and tests.
- `packages/proto`: shared data-plane protobuf definitions.
- `packages/schema`: shared control-plane OpenAPI schema.
- `docs/UPSTREAMS.md`: pinned upstream repos we are borrowing from.

## Local Commands

Use the root `Makefile` to run the most important test suites:

```sh
make test-all
```

## Notes

- The upstream research repos live under `.local/upstream` and are intentionally
  ignored by git.
- The implementation currently targets allowlisted provider Macs first.
- The current branch includes encrypted job envelopes, signed settlement
  vouchers, a Secure Enclave helper tool, signed posture telemetry, and real
  MLX-backed end-to-end tests.
