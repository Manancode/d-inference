# DGInf — Decentralized Private Inference

A platform for private, decentralized AI inference on Apple Silicon Macs. Mac owners rent out idle compute. Consumers get private inference on open-source models with hardware-backed trust from Apple's Secure Enclave.

**Privacy claim:** Nobody in the chain can see your prompts — not the coordinator (runs in a hardware-encrypted Confidential VM), not the provider (SIP + Hardened Runtime + in-process inference prevent memory inspection), and not DGInf as a company.

## How It Works

```
Consumer (Python SDK / CLI)
    │
    │  HTTPS + OpenAI-compatible API
    ▼
Coordinator (Go, GCP Confidential VM — AMD SEV-SNP)
    │
    │  WebSocket (outbound from provider, no port forwarding needed)
    ▼
Provider Agent (Rust, hardened process)
    │
    │  In-process Python (PyO3) — no subprocess, no IPC
    ▼
MLX inference engine → Metal → Apple Silicon GPU
```

## Quick Start

### Consumer

```bash
pip install dginf

# One-shot inference
dginf configure --url https://coordinator.dginf.io --api-key dginf-...
dginf ask "Explain quantum computing"

# Or use the Python SDK (OpenAI-compatible drop-in)
from dginf import DGInf
client = DGInf()
response = client.chat.completions.create(
    model="mlx-community/Qwen2.5-7B-Instruct-4bit",
    messages=[{"role": "user", "content": "Hello"}],
    stream=True,
)
```

### Provider

```bash
# Install and initialize
curl -fsSL https://dginf.io/install.sh | bash
dginf-provider init

# Start serving (in-process inference, auto-installs mlx-lm if needed)
dginf-provider serve --coordinator wss://coordinator.dginf.io/ws/provider
```

The provider agent:
1. Detects your Apple Silicon hardware
2. Generates a Secure Enclave identity key
3. Loads the model directly in-process via MLX
4. Connects to the coordinator and starts accepting jobs

## Architecture

| Component | Language | What It Does |
|-----------|----------|-------------|
| **Coordinator** (`coordinator/`) | Go | Control plane: routing, attestation verification, payments, API |
| **Provider Agent** (`provider/`) | Rust + Python (PyO3) | Inference agent: in-process MLX, security hardening, attestation |
| **Consumer SDK** (`sdk/`) | Python | OpenAI-compatible client library and CLI |
| **macOS App** (`app/DGInf/`) | Swift/SwiftUI | Menu bar app with idle detection, earnings dashboard |
| **Secure Enclave** (`enclave/`) | Swift | Hardware-bound P-256 identity, signed attestation blobs |
| **Scripts** (`scripts/`) | Bash | Hardened Runtime signing, app bundling |

## Security Model

DGInf prevents providers from reading consumer prompts through multiple layers:

| Protection | What It Blocks |
|---|---|
| **PT_DENY_ATTACH** | Debugger attachment (lldb, dtrace, Instruments) |
| **Hardened Runtime** | External process memory reads (task_for_pid, mach_vm_read) |
| **SIP enforcement** | Kernel-level enforcement of the above; cannot be disabled without reboot |
| **In-process inference** | No subprocess or IPC channel to sniff — all inference inside hardened process |
| **Python path locking** | Only loads from signed app bundle, not system packages (prevents malicious vllm-mlx) |
| **Signed app bundle** | Any file modification breaks code signature; SIP refuses to run modified bundle |
| **Binary hash attestation** | Coordinator verifies provider runs the expected blessed binary version |
| **SIP re-verification** | Checked at startup, before every request, and in every 5-min challenge-response |
| **Memory wiping** | Volatile-zeros prompt/response buffers after each request |
| **MDM SecurityInfo** | Hardware-verified SIP, Secure Boot, and system integrity via Apple MDM |

**Remaining attack surface:** Physical memory probing on soldered LPDDR5x — same threat model as Apple Private Cloud Compute.

### Trust Levels

| Level | Name | How Achieved |
|-------|------|-------------|
| `none` | Open Mode | No attestation |
| `self_signed` | Self-Attested | Secure Enclave P-256 signature verified + periodic challenge-response |
| `hardware` | Hardware-Attested | MDA certificate chain from Apple Enterprise Attestation Root CA (via MDM) |

## MDM Infrastructure

DGInf uses Apple MDM (MicroMDM) to query provider security posture:

- **Enrollment:** Provider installs a `.mobileconfig` profile (one click, minimal permissions)
- **Access Rights:** Query device info + security info only (AccessRights=1041). No erase, lock, or app management.
- **SecurityInfo query returns:** SIP status, Secure Boot level, Authenticated Root Volume, FileVault status
- **Push notifications:** APNs for on-demand attestation queries

Infrastructure: MicroMDM + SCEP + step-ca (ACME with device-attest-01) on AWS.

## Inference

DGInf runs inference **in-process** via PyO3 (embedded Python). The MLX engine loads directly inside the hardened Rust process — no subprocess, no HTTP, no Unix socket.

| Backend | Status | Use |
|---------|--------|-----|
| **mlx-lm** (in-process) | Primary | Embedded via PyO3, single-process security |
| **vllm-mlx** (in-process) | Preferred when available | Continuous batching + prefix caching |

Auto-installs mlx-lm if not present. No subprocess fallback — in-process is the only mode.

## Payments

- Internal micro-USD ledger (1 USD = 1,000,000 micro-USD)
- $0.50 per 1M output tokens, 10% platform fee
- Consumer deposits via Stripe (MVP) or Tempo blockchain settlement
- Provider payouts: 90% of inference charges

## Development

```bash
# Build all components
cd coordinator && go build ./...
cd provider && cargo build --release
cd enclave && swift build -c release
cd sdk && pip install -e .

# Run tests
cd coordinator && go test ./...
cd provider && cargo test
cd enclave && swift test

# Sign binaries with Hardened Runtime
./scripts/sign-hardened.sh

# Build signed app bundle
./scripts/bundle-app.sh
```

## Hardware Support

Any Apple Silicon Mac (M1 or later):

| Chip | Memory | Bandwidth | Best Models |
|------|--------|-----------|-------------|
| M1 | 8-16 GB | 68 GB/s | 3B-8B |
| M1 Pro/Max | 16-64 GB | 200-400 GB/s | 8B-33B |
| M2 Pro/Max | 16-96 GB | 200-400 GB/s | 8B-70B |
| M3 Pro/Max | 18-128 GB | 150-400 GB/s | 8B-122B |
| M3 Ultra | 96-256 GB | 819 GB/s | 8B-230B |
| M4 Pro/Max | 24-128 GB | 273-546 GB/s | 8B-122B |

## License

Proprietary. All rights reserved.
