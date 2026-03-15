# DGInf — Decentralized Inference Platform

## Table of Contents

1. [Product Thesis](#1-product-thesis)
2. [Supported Hardware](#2-supported-hardware)
3. [Trust Anchor: YubiHSM 2](#3-trust-anchor-yubihsm-2)
4. [Privacy Architecture](#4-privacy-architecture)
5. [Trust Model](#5-trust-model)
6. [Provider Experience](#6-provider-experience)
7. [Product Scope](#7-product-scope)
8. [Actors](#8-actors)
9. [System Architecture](#9-system-architecture)
10. [Runtime Strategy](#10-runtime-strategy)
11. [Capacity Planning and Scheduling](#11-capacity-planning-and-scheduling)
12. [Job Lifecycle](#12-job-lifecycle)
13. [Networking](#13-networking)
14. [Pricing and Payments](#14-pricing-and-payments)
15. [Provider Agent — Detailed Design](#15-provider-agent--detailed-design)
16. [Coordinator — Detailed Design](#16-coordinator--detailed-design)
17. [Security and Compliance](#17-security-and-compliance)
18. [Competitive Landscape](#18-competitive-landscape)
19. [Feasibility and Risk Assessment](#19-feasibility-and-risk-assessment)
20. [Roadmap](#20-roadmap)
21. [Tech Stack Summary](#21-tech-stack-summary)
22. [Open Questions](#22-open-questions)
23. [Sources](#23-sources)

---

## 1. Product Thesis

DGInf is a control plane and marketplace for high-memory unified-memory machines: NVIDIA DGX Spark and Apple Mac Studio. It lets owners expose idle capacity and lets consumers rent that capacity for inference workloads.

Both devices are unusually interesting because they combine compact desktop form factors with large coherent unified memory pools (128–256 GB), making them useful for high-memory inference workloads that do not fit on typical consumer GPUs. Neither device has hardware TEE support suitable for blind confidential execution. DGInf addresses this gap with the **YubiHSM 2** ($650 USB hardware security module) as a universal trust anchor, combined with an **Apple Private Cloud Compute-inspired privacy architecture** that eliminates software access paths to inference data.

The platform is built around three honest principles:

- These machines are a good fit for open-model inference, private fleets, allowlisted-provider deployments, and HSM-attested public marketplace workloads.
- Neither machine supports full confidential computing. The YubiHSM provides hardware-rooted identity, controlled model decryption, and tamper-evident audit — not runtime memory encryption.
- Privacy is achieved through architectural elimination of access vectors (no shell, no debug, immutable OS, crypto-shredding) rather than hardware enclaves — the same approach Apple uses in Private Cloud Compute.

**One-liner:** Marketplace and fleet manager for DGX Spark and Mac Studio inference, with YubiHSM-rooted trust and PCC-inspired privacy.

---

## 2. Supported Hardware

### 2.1 NVIDIA DGX Spark

| Item | Value |
|------|-------|
| SoC | GB10 Grace Blackwell Superchip (NVIDIA + MediaTek, TSMC 2.5D packaging) |
| GPU | Blackwell architecture, 6,144 CUDA cores, 5th-gen Tensor Cores, 4th-gen RT Cores |
| CPU | 20-core Arm (10x Cortex-X925 + 10x Cortex-A725) |
| AI Performance | Up to 1 PFLOP FP4 sparse, 1,000 TOPS inference |
| Memory | 128 GB unified LPDDR5x (CPU+GPU coherent, 256-bit, 16 channels) |
| Memory Bandwidth | 273 GB/s |
| CPU-GPU Interconnect | NVLink Chip-to-Chip (C2C), 600 GB/s bidirectional |
| Storage | 4 TB NVMe M.2 SSD (hardware self-encryption) |
| Ethernet | 1x 10 GbE RJ-45 |
| Smart NIC | ConnectX-7 (2x QSFP connectors, 2x100G links in multi-host mode, ~160–200 Gbps aggregate) |
| WiFi / Bluetooth | Wi-Fi 7 / Bluetooth 5.4 |
| Copy Engines | 2 (simultaneous data transfers) |
| Video | 1x NVENC, 1x NVDEC, 1x HDMI 2.1a |
| USB | 4x USB Type-C (1x with Power Delivery) |
| SoC TDP | 140 W |
| PSU | 240 W external (included) |
| Form Factor | 150mm x 150mm x 50.5mm, 1.2 kg |
| Operating Temp | 5°C to 30°C |
| OS | DGX OS 7.4 (Ubuntu 24.04, ARM64, kernel 6.17, CUDA 13.0.2, driver 580.126.09) |
| Price | ~$4,699 (increased from $3,999 due to LPDDR5x supply constraints, Feb 2026) |
| Strength | Compute (up to 1 PFLOP FP4). Best at prefill / prompt processing. |
| Weakness | Memory bandwidth (273 GB/s). Decode speed is the bottleneck. |

#### DGX Spark Software Stack

Pre-installed and validated:

- **Inference:** TensorRT-LLM, NVIDIA NIM, vLLM, SGLang, llama.cpp, Ollama
- **ML Frameworks:** PyTorch, TensorFlow, JAX (all Blackwell-optimized via NGC containers)
- **Data Science:** cuDF, cuML, cuGraph, XGBoost, Apache Spark RAPIDS
- **Tools:** JupyterLab, Nsight Systems/Compute/Graphics, NVIDIA AI Workbench, Docker + NVIDIA Container Toolkit
- **Official Playbooks:** [github.com/NVIDIA/dgx-spark-playbooks](https://github.com/NVIDIA/dgx-spark-playbooks)

#### DGX Spark Hardware Limitations

| Limitation | Details | Source |
|-----------|---------|--------|
| **No Confidential Computing** | GB10 does not support NVIDIA CC. Grace CPU lacks Arm CCA (design locked before CCA ratification). Hardware limitation, cannot be enabled via firmware. | [NVIDIA Forums](https://forums.developer.nvidia.com/t/confidential-computing-support-for-dgx-spark-gb10/347945) |
| **No MIG** | Unified memory architecture prevents GPU partitioning. `nvidia-smi` reports "Not Supported." Hardware limitation. | [NVIDIA Forums](https://forums.developer.nvidia.com/t/is-the-nvidia-dgx-spark-system-compatible-with-mig-technology/340089) |
| **No vGPU** | Not available on DGX Spark. | [NVIDIA Forums](https://forums.developer.nvidia.com/t/mig-vgpu-support/347828) |
| **ConnectX-7 is 2x100G, not 200G** | Two PCIe Gen5 x4 links in multi-host mode. Requires correct interface pairing for full throughput. Misconfiguration drops to 80–100 Gbps. | [LMSYS Review](https://lmsys.org/blog/2025-10-13-nvidia-dgx-spark/) |

#### DGX Spark Model Capacity

| Quantization | Max Model Size | Examples |
|-------------|---------------|---------|
| BF16 (full precision) | ~60B params | — |
| FP8 | ~70–80B params | Llama 3.1 70B (~70 GB at FP8) |
| NVFP4 | ~200B params | Qwen-235B, Jamba-v1.5-Large |
| Dual Spark (256 GB) | ~405B params | Llama 3.1 405B |

#### DGX Spark Benchmarks (Single Node)

| Model | Framework | Precision | Prefill tok/s | Decode tok/s |
|-------|-----------|-----------|--------------|-------------|
| Llama 3.1 8B | SGLang (B1) | FP8 | 7,991 | 20.5 |
| Llama 3.1 8B | SGLang (B32) | FP8 | 7,949 | 368 |
| Llama 3.1 70B | SGLang (B1) | FP8 | 803 | 2.7 |
| DeepSeek-R1 14B | SGLang (B8) | FP8 | 2,074 | 83.5 |
| Qwen3 14B | TRT-LLM | NVFP4 | 5,929 | 22.7 |
| Qwen3 32B | — | FP8 | ~483 | — |
| GPT-OSS 20B | Ollama | MXFP4 | 2,053 | 49.7 |
| GPT-OSS 120B | — | MXFP4 | 1,725 | — |

Sources: [LMSYS Review](https://lmsys.org/blog/2025-10-13-nvidia-dgx-spark/), [NVIDIA Perf Blog](https://developer.nvidia.com/blog/how-nvidia-dgx-sparks-performance-enables-intensive-ai-tasks/)

Key performance characteristics:
- Memory bandwidth (273 GB/s) is the primary bottleneck vs. 3.35 TB/s on H100
- Speculative decoding (EAGLE-3) delivers up to 2x speedup
- SGLang shows linear batch scaling up to batch size 32

### 2.2 Apple Mac Studio

| Item | M4 Max | M3 Ultra |
|------|--------|----------|
| CPU | 14-core (10P+4E) or 16-core (12P+4E), 4.5 GHz | 28-core (20P+8E) or 32-core (24P+8E), 4.01 GHz |
| GPU | 32-core or 40-core | 60-core or 80-core |
| Neural Engine | 16-core | 32-core |
| Memory | 36/48/64/128 GB LPDDR5x | 96/192/256 GB LPDDR5 |
| Memory Bandwidth | 410 GB/s (32-core) / 546 GB/s (40-core) | 819 GB/s |
| FP16 Compute | ~26 TFLOPS | ~26 TFLOPS |
| Storage | 512 GB to 8 TB NVMe | 1 TB to 16 TB NVMe |
| Ethernet | 10 Gb Ethernet (Nbase-T: 1/2.5/5/10 Gbps) |
| Thunderbolt | 4x Thunderbolt 5 (up to 120 Gbps bidirectional) |
| WiFi / Bluetooth | Wi-Fi 6E / Bluetooth 5.3 |
| USB-A | 2x ports |
| HDMI | 1x HDMI 2.1 |
| Power | <200 W under heavy load, 4.8–5.4 W idle |
| OS | macOS (ARM64) |
| Strength | Memory bandwidth (819 GB/s on M3 Ultra). Best at token generation / decode. |
| Weakness | Compute (~26 TFLOPS). Prefill is slower than Spark. |

**Important:** M4 Ultra does not exist. Apple skipped it. M5 Ultra expected mid-2026. The 512 GB M3 Ultra option has been discontinued due to DRAM shortages.

#### Mac Studio Pricing

**M4 Max Models:**

| Config | Price |
|--------|-------|
| 14-core CPU / 32-core GPU / 36GB / 512GB | $1,999 |
| 16-core CPU / 40-core GPU / 48GB / 1TB | $2,499 |
| 16-core CPU / 40-core GPU / 64GB / 1TB | ~$2,699 |
| 16-core CPU / 40-core GPU / 128GB / 1TB | $3,699 |

**M3 Ultra Models:**

| Config | Price |
|--------|-------|
| 28-core CPU / 60-core GPU / 96GB / 1TB | $3,999 |
| 32-core CPU / 80-core GPU / 96GB / 1TB | $5,499 |
| 28-core CPU / 60-core GPU / 256GB / 1TB | ~$5,999 |
| Maxed (32-core / 80-core / 256GB / 16TB) | ~$12,299+ |

#### Mac Studio Inference Frameworks

| Framework | Status | Notes |
|-----------|--------|-------|
| **MLX** (Apple) | Native, best performance | Apple's ML framework, optimized for Metal/unified memory |
| **llama.cpp** | Fully supported | Metal backend, widely used |
| **vllm-mlx** | Available (2025+) | 21–87% higher throughput than llama.cpp, continuous batching |
| **Ollama** | Fully supported | Uses llama.cpp backend with Metal |
| **LM Studio** | Fully supported | GUI-based |

#### Mac Studio Benchmarks

**M4 Max (128GB):**

| Model | Quantization | Tokens/sec | Framework |
|-------|-------------|------------|-----------|
| Llama 3.1 70B | Q4_K_M | ~18 tok/s | llama.cpp |
| 33B–70B models | Q4 | 30–45 tok/s | MLX |
| Qwen3-0.6B | Various | Up to 525 tok/s | vllm-mlx |

**M3 Ultra (256GB):**

| Model | Quantization | Tokens/sec | Framework |
|-------|-------------|------------|-----------|
| DeepSeek-V3 671B | 4-bit | ~20 tok/s | mlx-lm (512GB config) |
| Qwen3-30B | 4-bit | ~2,320 tok/s (throughput) | MLX batch |
| Qwen3 235B | Q4_K_M | ~30 tok/s | MLX |
| Gemma-3-27b | Q4 | ~41 tok/s | LM Studio |

#### Mac Studio Security Features (Built-In)

| Feature | Description | Relevance to DGInf |
|---------|-------------|-------------------|
| **Secure Enclave** | Dedicated hardware security subsystem with own Boot ROM, AES engine, TRNG, encrypted memory. FIPS 140-2 Level 2 (Level 3 pending). | **Not accessible** for third-party attestation on macOS. `DCAppAttestService.isSupported` returns `false` on Mac. No PKCS#11. Only P-256 ECC to apps. |
| **Kernel Integrity Protection (KIP)** | Hardware-enforced on Apple Silicon. After kernel init, memory controller denies all writes to kernel code. Cannot be disabled at runtime. | Prevents kernel code modification — stronger than any Linux equivalent. |
| **Signed System Volume (SSV)** | Merkle tree hash over entire system volume. Seal verified by bootloader. Any modification invalidates seal. | Equivalent to dm-verity but Apple-managed. |
| **System Integrity Protection (SIP)** | Restricts root from modifying /System, /usr, Apple-signed binaries. Prevents unsigned kernel extensions. | Strong but owner can disable via Recovery Mode. |
| **IOMMU (DMA Protection)** | Per-device IOMMU for every DMA agent. Default-deny policy. Unauthorized DMA triggers kernel panic. | Prevents DMA-based memory extraction. |
| **No /dev/mem** | Apple Silicon does not expose physical memory to userspace. | Prevents direct memory reads. |
| **Secure Boot** | Boot ROM → LLB → iBoot → kernel. All Apple-signed. Three modes: Full, Reduced, Permissive. | Strong boot chain but owner controls which mode. |

#### Mac Studio Model Capacity (M3 Ultra 192GB)

Metal can access ~155GB of 192GB total. Practical limits:

| Model Size | Quantization | Memory | Fits? |
|-----------|-------------|--------|-------|
| Llama 3.1 70B | FP16 | ~140GB | Yes (tight) |
| Llama 3.1 70B | Q4_K_M | ~40GB | Easily |
| Falcon 180B | Q6_K | ~148GB | Yes (tight) |
| Llama 405B | Q4_K_M | ~210GB | No |
| DeepSeek-V3 671B | 4-bit | ~405GB | No (needs 512GB) |

#### Mac Studio Clustering

- **Thunderbolt 5 RDMA:** macOS 26.2 introduced RDMA over Thunderbolt 5 with <50 us latency
- **EXO framework:** Purpose-built distributed inference for Apple Silicon clusters with day-0 TB5 RDMA support
- **Benchmark:** 4x M3 Ultra Mac Studios (1.5 TB total) running Qwen3 235B achieved 31.9 tok/s (vs 19.5 tok/s single node, 1.6x speedup)
- **Heterogeneous:** EXO demonstrated 2x DGX Spark + 1x M3 Ultra achieving 2.8x speedup over M3 Ultra alone on Llama 3.1 8B
- **Limitation:** No TB5 switches exist — devices must be daisy-chained, limiting practical cluster size to 3–4 nodes

Sources: [Jeff Geerling TB5 RDMA](https://www.jeffgeerling.com/blog/2025/15-tb-vram-on-mac-studio-rdma-over-thunderbolt-5/), [EXO Labs](https://blog.exolabs.net/nvidia-dgx-spark/)

### 2.3 Why Both Platforms

DGX Spark and Mac Studio are complementary, not competing:

| Workload Phase | Better Hardware | Why |
|---------------|----------------|-----|
| Prefill (prompt processing) | DGX Spark | ~100 TFLOPS compute advantage (4x Mac Studio) |
| Decode (token generation) | Mac Studio M3 Ultra | 819 GB/s bandwidth (3x Spark's 273 GB/s) |
| Large models (200B+) | Mac Studio M3 Ultra 256 GB | More memory than Spark's 128 GB |
| Quantized fast models (8B–70B) | Either | Both capable; route by price/availability |
| Batch throughput | DGX Spark | Tensor Cores excel at batched inference |
| Heterogeneous cluster | Both together | EXO demonstrated 2.8x combined speedup |

**Comparison table:**

| Metric | DGX Spark (GB10) | Mac Studio M3 Ultra |
|--------|-----------------|---------------------|
| Price | ~$4,699 | $3,999–$12,299 |
| Memory | 128 GB | 96–256 GB |
| Memory Bandwidth | 273 GB/s | 819 GB/s (3x) |
| FP16 Compute | ~100 TFLOPS | ~26 TFLOPS |
| Prefill (Llama 3.1 8B, 8k ctx) | 1.47s (3.8x faster) | 5.57s |
| Token generation (same) | 2.87s | 0.85s (3.4x faster) |
| Power | 140 W TDP | <200 W |
| OS | Linux (DGX OS) | macOS |

### 2.4 Hardware Limitations Shared by Both

| Limitation | DGX Spark | Mac Studio |
|-----------|-----------|------------|
| Confidential Computing / TEE | Not supported (GB10 lacks CC, Grace lacks Arm CCA) | Not supported (Secure Enclave not exposed to third-party apps) |
| Multi-Instance GPU | Not supported (unified memory) | Not supported |
| Hardware memory encryption | Not available | Not available to third-party apps |
| vGPU | Not supported | Not supported |

**Why no affordable hardware has TEE for AI inference:**

We researched all options under $10K:

| Hardware | TEE Status | Why Not |
|----------|-----------|---------|
| DGX Spark (GB10) | No | Grace CPU lacks Arm CCA. Hardware limitation. |
| Mac Studio (Apple Silicon) | Partial | Secure Enclave exists but App Attest not supported on macOS. No public API for boot measurements. |
| RTX 5090 / RTX 4090 | No | Consumer GPUs — no CC silicon |
| AMD Instinct MI300X/MI350 | No | AMD has no GPU-level TEE. CPU-side SEV-SNP only. |
| Jetson AGX Thor | No | Confirmed not supported by NVIDIA |
| Threadripper PRO 9000 | No | SEV-SNP flags present but microcode-locked to EPYC only |

The cheapest GPU with confirmed CC support is the **RTX PRO 6000 Server Edition** at ~$11,600 (GPU alone), requiring an AMD EPYC or Intel Xeon server CPU. Total system: ~$14,000+.

The cheapest path to TEE-backed AI inference is **CPU-only** (Intel Xeon with TDX + AMX) at ~$3,000–$5,000, but limited to 7B–13B models with 10–20% overhead.

This is why the YubiHSM 2 approach exists — it provides hardware-rooted trust at $650 on hardware that has no TEE.

### 2.5 Licensing Constraints

| Concern | DGX Spark | Mac Studio |
|---------|-----------|------------|
| Hardware rental | No restrictions on renting the physical hardware | No restrictions on renting the physical hardware |
| DGX OS / NVIDIA software | EULA prohibits sublicensing, renting, or transferring software. NIM free tier prohibits multi-user serving. | — |
| macOS EULA | — | Requires: entire machine leased to one customer, 24-hour minimum lease, sole and exclusive control. "Service bureau, time-sharing" prohibited. AI inference not explicitly addressed. |
| Open-source runtimes | vLLM, SGLang, llama.cpp (Apache 2.0/MIT) — no restrictions on commercial multi-tenant use | MLX (Apache 2.0), llama.cpp (MIT) — no restrictions |
| NIM licensing | Free tier prohibits "server used to service multiple users." Marketplace use requires NVIDIA AI Enterprise license or use open-source alternatives. | N/A |

**Implications:**
- DGX Spark: Use **open-source runtimes** (vLLM, SGLang) for marketplace. Multi-job public marketplace is fine.
- Mac Studio: Must be leased as **dedicated machines** (one customer, 24h minimum) unless Apple legal clarifies otherwise. Alternatively: private fleet only.

---

## 3. Trust Anchor: YubiHSM 2

### 3.1 Why YubiHSM 2

Neither DGX Spark nor Mac Studio has a usable hardware TEE for third-party attestation. The Apple Secure Enclave exists but is not accessible — `DCAppAttestService.isSupported` returns `false` on macOS, only P-256 ECC is available to apps, there's no PKCS#11, and no public API for boot measurement sealing. The YubiHSM 2 fills this gap as an external hardware root of trust that works identically on both platforms.

### 3.2 YubiHSM 2 Specifications

| Property | Value |
|----------|-------|
| Form factor | USB-A nano (12mm x 13mm x 3.1mm, 1 gram), IP68 rated |
| Price | $650 |
| Certification | FIPS 140-2 Level 3 |
| Cryptographic capabilities | RSA 2048-4096, ECC (P-224 to P-521, curve25519), Ed25519/EdDSA, AES key wrap, SHA hashing, HMAC |
| Performance | RSA-2048 sign: ~139ms, ECDSA-P256 sign: ~73ms |
| API | PKCS#11 (cross-platform), Yubico native SDK (C, Python), Microsoft CNG |
| Key storage | 256 object slots, 128 KB total. Keys generated on-device, non-extractable. |
| Concurrent sessions | 16 |
| Access control | Role-based per security domain |
| Backup | M-of-N key wrap/unwrap capability |
| Tamper evidence | Hash-chained audit log |
| True RNG | On-chip ring oscillator, post-processed with CTR_DRBG |
| Works on | Linux (DGX OS ARM64) and macOS — identical PKCS#11 API on both |

### 3.3 What the YubiHSM Provides

| Capability | How It Helps DGInf |
|-----------|-------------------|
| **Non-extractable identity key** | Each provider node has a hardware-bound identity that cannot be cloned, transferred, or spoofed. Anti-sybil by construction. $650 economic barrier per identity. |
| **Controlled model decryption** | Consumer encrypts model weights to the HSM's public key. HSM only releases decryption key to a verified runtime process. Provider cannot extract model from disk. |
| **Boot-state key sealing** | HSM releases keys only when combined with valid boot measurements from fTPM (Linux) or macOS measured boot. Tampered OS → no decryption key → no model access. |
| **Tamper-evident audit log** | Hash-chained, hardware-backed log of all key operations. Coordinator can verify audit chain integrity. Cannot be backdated or falsified. |
| **Session key management** | HSM negotiates TLS/session keys for consumer-provider channel. Provider cannot MitM the encrypted connection. |
| **Platform-agnostic** | Same PKCS#11 API on DGX Spark (Linux) and Mac Studio (macOS). One trust model, two hardware platforms. |

### 3.4 What the YubiHSM Does NOT Provide

| Gap | Honest Assessment |
|-----|-------------------|
| Runtime memory encryption | Once model weights are decrypted and loaded into GPU/CPU memory, they are in plaintext. |
| Computation isolation | The inference process runs in normal user space, not an enclave. |
| Side-channel protection | No protection against timing or power analysis on the host. |

These gaps are addressed by the Privacy Architecture (Section 4).

### 3.5 Attestation Flow

```text
Provider Boot                 YubiHSM 2                   Coordinator
   │                              │                            │
   │ 1. System boots              │                            │
   │    (Secure Boot + fTPM       │                            │
   │     or macOS SSV)            │                            │
   │                              │                            │
   │ 2. Agent starts, reads       │                            │
   │    boot measurements         │                            │
   │────────────────────────────>│                            │
   │ 3. Present measurements     │                            │
   │    + request identity       │                            │
   │    attestation              │                            │
   │                              │                            │
   │ 4. HSM verifies boot state  │                            │
   │    matches sealed policy    │                            │
   │<────────────────────────────│                            │
   │ 5. HSM signs attestation    │                            │
   │    blob (identity +         │                            │
   │    boot measurements +      │                            │
   │    timestamp)               │                            │
   │                              │                            │
   │ 6. Send signed attestation ─────────────────────────────>│
   │                              │                            │
   │                              │   7. Verify HSM signature  │
   │                              │      against registered    │
   │                              │      public key            │
   │                              │                            │
   │                              │   8. Mark node as          │
   │                              │      "HSM-attested"        │
   │<──────────────────────────────────────────────────────────│
   │ 9. Node can accept jobs     │                            │
```

### 3.6 Model Decryption Flow (Private Models)

```text
Consumer                        Coordinator                  Provider + HSM
   │                                │                            │
   │ 1. Encrypt model weights       │                            │
   │    with HSM's public key       │                            │
   │    + upload to registry        │                            │
   │──────────────────────────────>│                            │
   │                                │                            │
   │                                │ 2. Route job to provider  │
   │                                │──────────────────────────>│
   │                                │                            │
   │                                │    3. Provider pulls       │
   │                                │       encrypted model      │
   │                                │                            │
   │                                │    4. Agent requests       │
   │                                │       decryption from HSM  │
   │                                │    ──────────────────────>│
   │                                │                           HSM
   │                                │    5. HSM checks:          │
   │                                │       - boot state valid?  │
   │                                │       - key policy met?    │
   │                                │       - audit log clean?   │
   │                                │                            │
   │                                │    6. HSM decrypts model   │
   │                                │       key → agent loads    │
   │                                │       weights into GPU     │
   │                                │    <──────────────────────│
   │                                │                            │
   │ 7. Inference proceeds          │                            │
   │<────────────────────────────────────────────────────────────│
   │                                │                            │
   │                                │    8. On job complete:     │
   │                                │       wipe model from      │
   │                                │       GPU memory + disk    │
```

---

## 4. Privacy Architecture

### 4.1 Design Inspiration: Apple Private Cloud Compute

Apple solved the same problem DGInf faces — strong privacy on Apple Silicon without hardware memory encryption — with their Private Cloud Compute (PCC) system. Their approach: **eliminate every software access path to data** rather than relying on hardware enclaves.

PCC's five requirements (which DGInf adopts):
1. **Stateless computation** — data used only for the request, never retained
2. **Enforceable guarantees** — technically enforced, not policy-dependent
3. **No privileged runtime access** — no interfaces bypass privacy guarantees
4. **Non-targetability** — can't target a specific user without compromising the whole system
5. **Verifiable transparency** — security researchers can independently verify

Apple achieved this with: no persistent storage, no SSH/shell/debug, immutable OS (SSV), crypto-shredding on reboot, OHTTP request anonymization, blind signatures, published software images, and a transparency log.

Source: [Apple PCC Security Blog](https://security.apple.com/blog/private-cloud-compute/), [PCC Documentation](https://security.apple.com/documentation/private-cloud-compute/)

**Key insight from PCC:** Apple does NOT use hardware memory encryption. Their threat model accepts that memory is plaintext but eliminates all software paths to read it. This is the same approach DGInf takes with the YubiHSM + locked-down OS.

An open-source implementation of PCC concepts exists: **[OpenPCC](https://github.com/openpcc/openpcc)** (Apache 2.0, built by engineers from Databricks and Apple) — implements OHTTP, encrypted streaming, and GPU attestation.

### 4.2 Provider Operating Modes

DGInf offers three operating modes with different privacy levels:

#### Open Mode (Sleep Mode)

```
What happens:
- DGInf runs as a normal background app/daemon
- Accepts jobs only when machine is idle
- Pauses when provider uses the machine
- Serves public open-source models only
- Provider has full access to their machine

Privacy level: Standard
- Provider COULD theoretically see inference data
- Consumers are warned
- Only public models — no secrets to protect

Provider experience:
- Invisible. Like a screensaver that earns money.
- Menu bar icon (macOS) or system tray (Linux)
- Zero friction
```

#### Guarded Mode (Mac Studio maximum)

```
What happens:
- Dedicated-lease or fleet-managed Macs run under Device Enrollment + supervision
- Personally owned Macs normally stay in Open Mode with User Enrollment
- SIP, KIP, SSV all active (default on macOS)
- YubiHSM provides identity and model decryption
- Terminal/SSH restricted via configuration profile
- Provider has limited access

Privacy level: Strong
- macOS built-in protections (KIP hardware-enforced, SSV Merkle tree, SIP, IOMMU)
- HSM-gated model decryption
- No /dev/mem on Apple Silicon
- Provider would need to reboot into Recovery Mode to bypass (which invalidates HSM attestation)

Limitation: Provider can bypass by rebooting into Recovery Mode and disabling SIP.
Mitigation: HSM detects changed boot state → refuses keys → node drops off network. No data flows to compromised machine.
```

#### Vault Mode (DGX Spark maximum — PCC-inspired)

```
What happens:
- Machine reboots into a hardened, immutable OS image
- Provider has ZERO interactive access
- Machine becomes a locked inference appliance
- Serves both public AND private encrypted models
- Premium earnings rate

Privacy level: Extreme (Apple PCC-equivalent)
- Full privacy stack engaged (see Section 4.3)
```

### 4.3 Vault Mode — Full Privacy Stack (DGX Spark)

Vault Mode on DGX Spark implements an Apple PCC-equivalent privacy architecture:

```text
Layer 0: Hardware Trust Anchor
  └─ YubiHSM 2 (identity keys, model decrypt keys, boot-state sealing)
  └─ fTPM (measured boot chain, PCR values)

Layer 1: Immutable Verified OS
  └─ dm-verity root filesystem (Merkle-tree verified, read-only)
  └─ UEFI Secure Boot (signed bootloader + kernel)
  └─ Kernel lockdown LSM (confidentiality mode)
  └─ Module loading disabled after boot (kernel.modules_disabled=1)
  └─ YubiHSM gates keys to fTPM PCR values
      → Tampered OS = changed PCRs = HSM refuses keys = no inference

Layer 2: Zero Access
  └─ No SSH server (openssh-server removed entirely)
  └─ No console / serial / emergency shell (all getty services masked)
  └─ No ptrace (kernel.yama.ptrace_scope=3)
  └─ No core dumps (fs.suid_dumpable=0, limits.conf hard core 0)
  └─ No debug tools in image (gdb, strace, ltrace, tcpdump removed)
  └─ No JIT compilation (W^X enforced via SELinux/AppArmor)
  └─ No arbitrary package installation (read-only root)
  └─ Provider literally cannot interact with machine except power on/off

Layer 3: Stateless Computation
  └─ Encrypted swap with random per-boot key (/dev/urandom in crypttab)
      → Crypto-shredded on every reboot — swap data unrecoverable
  └─ All writable data on tmpfs (RAM-backed, lost on reboot)
  └─ Model wiped from GPU memory after job completion
  └─ Process address space recycled periodically
  └─ No persistent writable filesystems

Layer 4: Controlled Model Access
  └─ Model weights encrypted to YubiHSM public key
  └─ HSM decrypts only with valid boot state (fTPM PCRs match)
  └─ Decrypted weights exist only in process memory (never on disk)
  └─ Wipe-on-stop enforced

Layer 5: Request Anonymization
  └─ OHTTP relay (RFC 9458) — coordinator never sees consumer's source IP
  └─ RSA blind signatures (RFC 9474) — auth without identity linkage
  └─ E2E encryption from consumer device to inference process
  └─ Coordinator relays only opaque encrypted bytes

Layer 6: Attestation + Transparency
  └─ Remote attestation via YubiHSM-signed boot measurements
  └─ Transparency log for all DGInf OS images (Sigstore/Rekor)
  └─ Consumer SDK verifies attestation BEFORE encrypting request
  └─ Published OS images for independent binary audit
  └─ Reproducible builds for verifiability

Layer 7: Monitoring (Privacy-Preserving)
  └─ Structured, audited-only logging (no user data in logs)
  └─ Log filter daemon prevents accidental data disclosure
  └─ Metrics-only observability pipeline (throughput, latency, not content)
```

### 4.4 Privacy Levels Achieved

| Data | Open Mode | Guarded Mode | Vault Mode |
|------|-----------|-------------|------------|
| **Model weights at rest** | Not protected (public models) | **Strong** (HSM-encrypted) | **Extreme** (HSM-encrypted, dm-verity OS) |
| **Model weights in memory** | Not protected | **Moderate** (KIP prevents kernel modification, but owner could reboot to Recovery) | **Strong** (no tools to dump memory, no shell, no ptrace, no debug) |
| **Prompts in transit** | TLS encrypted | E2E encrypted (HSM keys) | **Extreme** (E2E + OHTTP anonymized) |
| **Prompts during computation** | Visible to provider | **Moderate** (restricted access) | **Strong** (no access path — no shell, no debugger, no ptrace) |
| **Outputs in transit** | TLS encrypted | E2E encrypted | **Extreme** (E2E encrypted) |
| **User identity** | Known to coordinator | Known to coordinator | **Extreme** (OHTTP + blind signatures) |
| **Data after job** | Persists until overwritten | Encrypted cache, wipe-on-stop | **Extreme** (crypto-shredded, tmpfs, no persistent storage) |

**Privacy level definitions:**
- **Extreme** = Cryptographic guarantee. Requires breaking the crypto to defeat.
- **Strong** = Requires a physical hardware attack (cold boot, memory bus probing) on a fully locked-down system. No software-only attack works. This is the same level Apple PCC claims.
- **Moderate** = Requires OS-level bypass (Recovery Mode, SIP disable). Detectable by HSM attestation.
- **Not protected** = Provider can access.

### 4.5 The Remaining Privacy Gap (Honest)

The only attack that works against Vault Mode: **physical lab-grade memory probing on a running, locked-down system.**

An attacker would need to:
1. Open the DGX Spark chassis (could be detected with tamper switch)
2. Probe the LPDDR5x memory chips with specialized equipment
3. While inference is actively running
4. Before the job completes and memory is wiped

**Mitigating factors:**
- LPDDR5x in DGX Spark is **soldered into the GB10 SoC package** — physically desoldering it destroys it
- Same for Mac Studio — memory is soldered to the Apple Silicon die
- Cold boot attacks are very difficult on soldered memory vs. removable DIMMs
- A $20 tamper switch + HSM policy ("if tamper detected, wipe all keys") could further mitigate

This is the **exact same threat model as Apple PCC.** Apple's additional mitigation: armed security guards at data centers + hardware tamper switches. DGInf's mitigation: soldered memory + optional tamper switch + HSM key destruction.

### 4.6 Platform Privacy Comparison

| Capability | DGX Spark (Linux) | Mac Studio (macOS) |
|-----------|-------------------|-------------------|
| **Maximum privacy mode** | **Vault Mode (Extreme)** | **Guarded Mode (Strong)** |
| Boot measurement access | fTPM — public API, standard tpm2-tools | Secure Enclave — **no public API** |
| OS lockdown | dm-verity + kernel lockdown — battle-tested | SIP + KIP + SSV — strong but owner can disable via Recovery |
| Shell removal | Complete (remove binary, mask services) | Incomplete (Recovery Mode always accessible) |
| HSM boot-state sealing | Full chain (fTPM PCR → HSM verification) | **Partial chain** (Managed Device Attestation provides Secure Enclave-attested SIP/SecureBoot/kext status, but no arbitrary PCR sealing) |
| Owner bypass | Very hard (dm-verity detects any change, HSM refuses) | Possible (Recovery Mode → disable SIP → reboot) |
| Kernel integrity | Kernel lockdown LSM (software-enforced) | KIP (hardware-enforced — stronger) |
| DMA protection | Must configure IOMMU | Default-deny IOMMU from boot (stronger) |
| Physical memory access | /dev/mem blocked by kernel lockdown | No /dev/mem on Apple Silicon (stronger) |

**Summary:** Mac Studio has stronger individual security features (hardware KIP, default IOMMU) but weaker attestation chain (no boot measurement API). DGX Spark has weaker individual features but complete attestation chain. Vault Mode is only achievable on DGX Spark.

### 4.7 Guarded Mode — macOS Implementation Details

Since macOS doesn't expose boot measurements to third-party apps, the Mac strategy splits into two enrollment tiers:

| Approach | How It Works | Best Fit | Strength / Tradeoff |
|----------|-------------|----------|---------------------|
| **Account-driven User Enrollment** | Default for personally owned Macs. DGInf manages only its own app, settings, and accounts. Managed Device Attestation is still available, but serial number and UDID are omitted to preserve user privacy. Unenrolling removes DGInf-managed apps/settings. | Personal Macs in Open Mode, private fleets, allowlisted providers | **Best provider privacy, lower control.** Good for management + attestation, but not enough for full-machine lockdown. |
| **Account-driven Device Enrollment** | Mac is enrolled as a managed device and becomes supervised. DGInf can apply broader restrictions, enforce FileVault/update posture, and use Recovery Lock / remote erase where appropriate. | Dedicated lease, managed fleets, stronger Mac Guarded Mode | **Higher control, lower provider privacy.** Required when DGInf needs system-wide restrictions. |
| **Managed Device Attestation** | Secure Enclave-backed attestation of device identity and posture. On Mac this can include security posture such as Secure Boot and SIP state; under User Enrollment the attestation remains privacy-preserving because serial/UDID are omitted. | Both enrollment tiers | Strong evidence the device is genuine and in the expected posture, but still not TPM-style PCR sealing. |
| **YubiHSM identity** | Same as DGX Spark — non-extractable identity key for the DGInf node. | Both enrollment tiers | Strong node identity, independent of Apple APIs. |
| **Software integrity checks** | Agent verifies: SIP enabled, KIP active, SSV seal valid, expected processes running. | Both enrollment tiers | Moderate; useful posture signal, but root on the host can still spoof some checks. |

**Default macOS policy:**
- Personally owned Mac Studio nodes use **account-driven User Enrollment** by default.
- DGInf uses User Enrollment when the goal is privacy-preserving management and the provider continues using the Mac as a personal workstation.
- **Device Enrollment** is reserved for dedicated-lease or fleet-managed Macs where the operator explicitly accepts stronger control and lower local privacy.

**Important:** User Enrollment is the right default for personal Macs, but it does **not** replace full Guarded Mode controls. The broad restrictions described elsewhere in this spec (for example restricting Terminal, SSH, developer tools, or Recovery behavior) require Device Enrollment and supervision, not User Enrollment.

**Managed Device Attestation — Verified Properties (All Enrollment Types Including User Enrollment)**

Managed Device Attestation uses the Secure Enclave to generate hardware-bound attestation certificates. The cryptographic mechanism is identical across all enrollment types — the Secure Enclave generates a private key using its hardware TRNG, binds it to the physical device, and provides non-exportable attestation. Even if the application processor OS is compromised, the attestation remains reliable because it only requires the Secure Enclave to be intact.

The following properties are Secure Enclave-attested and available under **all enrollment types including User Enrollment** (macOS 14+, Apple Silicon only):

| OID | Property | Why It Matters for DGInf |
|-----|----------|-------------------------|
| `1.2.840.113635.100.8.13.1` | **SIP status** | Verify System Integrity Protection is enabled — not self-reported, hardware-attested |
| `1.2.840.113635.100.8.13.2` | **Secure Boot status** | Verify Full Security boot mode is active |
| `1.2.840.113635.100.8.13.3` | **Third-party kernel extensions allowed** | Verify no unsigned kexts are loaded |
| `1.2.840.113635.100.8.10.2` | sepOS version | Verify Secure Enclave firmware is current |
| `1.2.840.113635.100.8.10.1` | OS version | Verify macOS version matches expectations |
| `1.2.840.113635.100.8.10.3` | LLB (bootloader) version | Verify boot chain integrity |
| `1.2.840.113635.100.8.9.4` | Software Update Device ID | Device class identification |
| `1.2.840.113635.100.8.11.1` | Freshness code (nonce) | Prevent replay attacks |
| — | Secure Enclave Enrollment ID | Correlate attestations to the same device without revealing identity |

Properties **omitted under User Enrollment** (for BYOD privacy — DGInf does not need these):

| OID | Property | Notes |
|-----|----------|-------|
| `1.2.840.113635.100.8.9.1` | Serial number | Not needed — YubiHSM provides node identity |
| `1.2.840.113635.100.8.9.2` | UDID | Not needed — Secure Enclave Enrollment ID suffices |

Sources: [Apple Platform Security Guide](https://support.apple.com/guide/security/sec8a37b4cb2/web), [ACME Payload Settings](https://support.apple.com/guide/deployment/depb95c66a07/web), [WWDC22: Managed Device Attestation](https://developer.apple.com/videos/play/wwdc2022/10143/), [WWDC23: What's New in Managing Apple Devices](https://developer.apple.com/videos/play/wwdc2023/10040/)

**This is stronger than originally assessed.** The Secure Enclave-attested SIP status, Secure Boot status, and kernel extension status mean DGInf can cryptographically verify the Mac's security posture — not through self-reported software checks, but through hardware-backed attestation rooted in the Secure Enclave. Combined with the YubiHSM identity key, the macOS attestation chain is:

```
Secure Enclave (hardware) → attests SIP + Secure Boot + kext status
                          → generates device-bound enrollment ID
YubiHSM 2 (hardware)      → provides non-extractable node identity
                          → gates model decryption keys
Combined                   → coordinator verifies both before routing jobs
```

**Revised boot measurement gap assessment:** The gap is narrower than Section 4.6 suggests. macOS cannot do arbitrary PCR-style sealing (true), but Managed Device Attestation provides hardware-attested security posture verification for the specific properties DGInf cares about (SIP, Secure Boot, kext status). This is not equivalent to Linux fTPM PCR sealing, but it is significantly stronger than "software integrity checks" alone.

**What remains weaker than DGX Spark Vault Mode:**
- Cannot seal HSM keys to arbitrary boot measurements (only to MDA-attested properties)
- Provider can still reboot into Recovery Mode and disable SIP (but next MDA attestation will reveal SIP=disabled → coordinator rejects node)
- No immutable root filesystem equivalent (SSV is Apple-managed, not DGInf-controlled)
- Cannot eliminate all interactive access (Recovery Mode always available to owner)

---

## 5. Trust Model

### 5.1 Trust Tiers

| Tier | When to Use | Data Assumption | What DGInf Guarantees | YubiHSM Role | Privacy Mode |
|------|-------------|-----------------|----------------------|-------------|-------------|
| Tier 0: Self-owned | Same org owns node and workload | Same trust boundary | Transport security, rollout control, wipe-on-stop | Optional (recommended for audit) | Any |
| Tier 1: Allowlisted | Trusted partner hosts workloads | Contractually trusted | Transport security, artifact integrity, HSM identity, audit log | Identity + audit | Open or Guarded |
| Tier 2: Public (open models) | Unknown provider, public model | Provider can see runtime data | HSM identity, benchmark verification, reputation | Identity + boot check | Open, Guarded, or Vault |
| Tier 3: Public (private models) | Unknown provider, private model | Model encrypted at rest. Runtime I/O visible (Open/Guarded) or protected (Vault). | HSM-controlled decryption, boot verification, wipe-on-stop | Full: identity, boot seal, decryption, audit | Guarded or Vault |

### 5.2 What Each Trust Layer Provides

| Layer | Mechanism | Protects Against |
|-------|-----------|-----------------|
| HSM identity | Non-extractable key in YubiHSM | Sybil attacks, identity spoofing, node cloning |
| Boot verification | fTPM PCRs (Linux) gating HSM key release | OS tampering, rootkit injection, boot-chain attacks |
| Model encryption | Consumer encrypts to HSM public key; HSM decrypts only with valid boot state | Model theft from disk, unauthorized model access |
| Signed runtimes | OCI image signatures (Sigstore Cosign) verified before launch | Malicious inference backend substitution |
| Audit chain | Hash-chained HSM audit log, coordinator-verifiable | Backdated or falsified operational records |
| Wipe-on-stop | Agent + OS-level cleanup after job completion | Data persistence after job ends |
| Transport encryption | mTLS with HSM-managed session keys | Man-in-the-middle, coordinator snooping |
| Request anonymization | OHTTP relay + blind signatures (Vault Mode) | User identity linkage, traffic analysis |
| Immutable OS | dm-verity Merkle tree (Vault Mode) | Runtime OS modification |
| Zero access | No shell/debug/ptrace (Vault Mode) | Software-based memory inspection |
| Stateless computation | Encrypted swap + tmpfs + crypto-shredding (Vault Mode) | Post-job data recovery |

### 5.3 Honest Limitations

- **Tier 2 and 3 in Open/Guarded Mode:** Runtime data (prompts, outputs) is visible to the provider process. Disclosed to consumers.
- **Tier 3 in Vault Mode:** Model weights protected at rest, runtime I/O protected by zero-access architecture. Remaining attack: physical memory probing (requires lab-grade equipment on soldered memory).
- **The YubiHSM is a trust floor, not a TEE ceiling.** It raises the bar far above reputation-only systems but does not match Intel TDX, AMD SEV-SNP, or NVIDIA CC.
- **Vault Mode is only achievable on DGX Spark.** Mac Studio's maximum is Guarded Mode due to the boot measurement API gap on macOS.

---

## 6. Provider Experience

### 6.1 What a Provider Needs

| Item | Cost | Notes |
|------|------|-------|
| DGX Spark or Mac Studio | $2,499–$12,299 | Provider already owns it |
| YubiHSM 2 | $650 | USB-A nano, shipped from Yubico |
| DGInf app/agent | Free | Download from dginf.io |
| Internet connection | Existing | Outbound-only, works behind NAT |
| **Total extra cost** | **$650** | **~10 minutes setup** |

### 6.2 Setup Flow (Mac Studio — GUI App)

**Step 1: Plug in YubiHSM + download app**
```
Plug YubiHSM 2 into any USB-A port on back of Mac Studio.
Download DGInf.app from dginf.io (~50 MB).
Open the app.
```

**Step 2: Setup wizard (~5 minutes)**

The app auto-detects hardware and walks through setup:

1. **Hardware detection:** "Mac Studio M4 Max 128GB detected. YubiHSM 2 detected."
2. **Enrollment choice:** For a personally owned Mac, the app recommends **account-driven User Enrollment**. This lets DGInf manage its app/profile and receive privacy-preserving attestation without taking full-device visibility. Dedicated-lease Macs can opt into **Device Enrollment** for stronger controls.
3. **HSM initialization:** Generates identity key on-device (~30 seconds). Displays node ID.
4. **Mode selection:** Choose Open (Sleep) Mode or Guarded Mode. On personal Macs, Open Mode + User Enrollment is the default. Mac Guarded Mode with broader restrictions requires Device Enrollment. Vault Mode available on DGX Spark only.
5. **Benchmarks:** Automatic (~3 minutes). Tests Llama 3.1 8B, Qwen3 14B, Llama 3.1 70B. Results determine pricing and model catalog eligibility.
6. **Pricing:** Set hourly rate. App shows market average and estimated monthly earnings.
7. **Start earning:** One click.

**Step 3: Daily operation**

The app sits in the macOS **menu bar**:
```
◉ DGInf — Earning $0.50/hr | 18 tok/s | Llama 70B
```

- **Provider sits down to work:** DGInf detects keyboard/mouse activity, pauses after current job finishes (~seconds). Provider uses Mac normally.
- **Provider walks away:** After configurable idle timeout (5/15/30 min), DGInf auto-resumes serving.
- **Provider goes to bed:** DGInf serves overnight. Wake up to "$4.00 earned overnight."
- **Provider goes on vacation:** Toggle Guarded Mode for premium earnings on a Device-Enrolled Mac, or leave the Mac in Open Mode under User Enrollment if it remains a personal workstation.

**Earnings dashboard:**
```
┌─────────────────────────────────────────────────────────┐
│  DGInf                                     ● Online     │
│                                                          │
│  Node: dginf-node-7f3a2b                                │
│  Hardware: Mac Studio M4 Max (128 GB)                    │
│  Mode: Open (Sleep)         HSM: ✅ Attested             │
│                                                          │
│  Today:      6.2 hrs active    $3.10 earned              │
│  This month: 112 hrs active    $47.30 earned             │
│                                                          │
│  Currently: Llama 3.1 70B (Q4) — 18 tok/s               │
│                                                          │
│  [Pause]  [Switch to Guarded Mode]  [Settings]           │
└─────────────────────────────────────────────────────────┘
```

### 6.3 Setup Flow (DGX Spark — CLI Agent)

```bash
# 1. Plug YubiHSM 2 into USB-A port
# 2. Install agent
curl -fsSL https://dginf.io/install.sh | bash

# 3. Initialize
dginf-agent init              # Detect hardware, install dependencies
dginf-agent hsm init          # Generate identity key on HSM
dginf-agent hsm seal-boot-policy  # Record boot measurements as policy

# 4. Register + benchmark
dginf-agent register          # Register with coordinator
dginf-agent benchmark         # Run hardware-specific benchmarks (~3 min)

# 5. Start serving
dginf-agent serve             # Open Mode (background daemon)
# OR
dginf-agent vault             # Vault Mode (reboots into hardened OS)
```

**Vault Mode on DGX Spark:**
- `dginf-agent vault` reboots into a hardened, immutable DGInf OS image
- Screen shows only a status dashboard (earnings, uptime, model, throughput)
- Provider cannot interact except power on/off
- To exit: hold power button 5 seconds → reboots to normal DGX OS
- Personal data untouched (Vault Mode uses separate boot partition)

### 6.4 Provider Earnings Estimates

| Scenario | Hours/day | Rate | Monthly Earnings |
|----------|-----------|------|-----------------|
| Open Mode, light personal use | 12 hrs idle | $0.40/hr | ~$146 |
| Open Mode, mostly away | 18 hrs idle | $0.40/hr | ~$219 |
| Guarded Mode weeknights | 10 hrs/night | $0.60/hr | ~$183 |
| Vault Mode weeknights (DGX Spark) | 10 hrs/night | $0.75/hr | ~$228 |
| Vault Mode full-time (vacation) | 24 hrs | $0.75/hr | ~$547 |
| Mixed (Open day + Vault night) | 18 hrs total | ~$0.55 avg | ~$300 |

**ROI estimates:**

| Hardware | Total Investment | Monthly Earnings (50% util) | Break-even |
|----------|-----------------|---------------------------|------------|
| Mac Studio M4 Max 128GB + HSM | $4,349 | ~$146–$219 | 20–30 months |
| Mac Studio M3 Ultra 256GB + HSM | $6,649 | ~$219–$365 (premium for 256GB) | 18–30 months |
| DGX Spark + HSM | $5,349 | ~$182–$292 | 18–29 months |

### 6.5 What Providers Don't Need to Do

- No command line knowledge (Mac Studio — GUI app handles everything)
- No Linux expertise (DGX Spark — installer handles everything)
- No Docker configuration
- No port forwarding (outbound-only connectivity)
- No crypto wallet (paid in USD via Stripe)
- No understanding of HSMs, attestation, dm-verity, or OHTTP
- No OS reinstall (Vault Mode uses separate boot partition)

---

## 7. Product Scope

### 7.1 In Scope for V1

- Open-model inference on DGX Spark public marketplace nodes (multi-job)
- Open-model inference on Mac Studio dedicated lease nodes (single-customer, 24h minimum)
- Private-fleet orchestration for teams that own their own nodes
- Allowlisted-provider deployments
- YubiHSM-based node identity and attestation
- Account-driven **User Enrollment** as the default management path for personally owned Mac Studio nodes
- Open Mode and Guarded Mode
- Curated hardware-specific runtime stacks and model catalog
- Benchmark-driven scheduling and pricing
- macOS GUI app and Linux CLI agent

### 7.2 In Scope for V1.5

- Vault Mode on DGX Spark (hardened OS image, full privacy stack)
- Private model uploads with HSM-controlled decryption (Tier 3)
- OHTTP relay for request anonymization
- Mixed hardware scheduling

### 7.3 Explicitly Out of Scope for V1

- Runtime memory encryption / full confidential execution
- Multi-tenant GPU slicing
- Distributed training across WAN
- On-chain payments as a launch requirement
- Mac Studio multi-job marketplace (Apple EULA constraint)
- Full-device lockdown on personally owned Macs enrolled only via User Enrollment
- Vault Mode on Mac Studio (boot measurement API gap)

### 7.4 Product Modes by Hardware

| Mode | DGX Spark | Mac Studio |
|------|-----------|------------|
| Private fleet | Yes | Yes |
| Allowlisted providers | Yes | Yes |
| Public marketplace (multi-job) | Yes | No (EULA constraint) |
| Dedicated lease (single customer) | Yes | Yes (24h minimum) |
| Open Mode (Sleep) | Yes | Yes |
| Guarded Mode | Yes | Yes |
| Vault Mode (Extreme Privacy) | **Yes** | **No** (boot measurement API gap) |

---

## 8. Actors

### 8.1 Provider
- Owns one or more DGX Spark or Mac Studio systems
- Purchases and installs a YubiHSM 2 ($650) per node
- Runs the DGInf provider agent (GUI app on macOS, CLI daemon on Linux)
- Chooses operating mode and privacy level
- Sets availability, runtime backends, and pricing
- Earns revenue when matched jobs run

### 8.2 Consumer
- Uses the CLI, SDK, or API to deploy models and invoke endpoints
- Chooses a trust tier, hardware preference, and privacy level requirement
- Pays for reserved capacity, usage, and optional relay/storage overhead
- Can upload private models (Tier 3) encrypted to specific HSM public keys
- SDK auto-verifies HSM attestation before encrypting requests

### 8.3 Fleet Owner
- Operates multiple nodes under one account (can mix Spark and Mac Studio)
- Uses DGInf primarily for scheduling, rollout, and observability
- May be the same party as the consumer

### 8.4 Coordinator
- Runs the control plane
- Maintains node registry, HSM public key store, health, routing, scheduling, billing
- Provides public ingress and relay services for nodes behind NAT
- Verifies HSM attestation signatures on registration and periodically
- Operates OHTTP relay infrastructure (Vault Mode)
- Maintains transparency log for OS images (Sigstore/Rekor)
- Tracks runtime digests, benchmark profiles, and provider reputation

---

## 9. System Architecture

```text
┌────────────────────┐        Control Plane         ┌──────────────────────┐
│ Consumer / Fleet   │ ───────────────────────────> │ Coordinator           │
│ Owner CLI / SDK    │                              │ Registry / HSM Keys   │
└─────────┬──────────┘ <─────────────────────────── │ Billing / Routing     │
          │                                         │ OHTTP Relay           │
          │             End-to-end mTLS / QUIC      │ Transparency Log      │
          │           (HSM-managed session keys,    └─────────┬────────────┘
          │            relayed as opaque bytes)                │
          ▼                                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Provider Agent (DGX Spark or Mac Studio)                                    │
│ ┌───────────┐  ┌──────────┐                                                │
│ │ YubiHSM 2 │  │ fTPM     │  (Linux: fTPM PCR sealing)                     │
│ │ (USB-A)   │  │ (Linux)  │  (macOS: MDM + software checks)                │
│ └───────────┘  └──────────┘                                                │
│ - HSM-attested registration                                                 │
│ - boot-state verification                                                   │
│ - outbound control connection                                               │
│ - runtime supervisor                                                        │
│ - encrypted model cache (HSM-gated decryption)                              │
│ - benchmark collector + usage meter                                          │
│ - idle detection (macOS) / always-on (Linux)                                 │
└─────────────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Inference Runtime (platform-specific)                                       │
│                                                                             │
│ DGX Spark: TensorRT-LLM / vLLM / SGLang (OCI containers, NVIDIA runtime)   │
│ Mac Studio: MLX / llama.cpp / vllm-mlx (native macOS processes)             │
│                                                                             │
│ Signed runtime artifacts, verified model digests                            │
│                                                                             │
│ In Vault Mode (DGX Spark only):                                             │
│ - Runs on immutable dm-verity root filesystem                               │
│ - No shell, no debug, no ptrace                                             │
│ - Encrypted swap with random per-boot key                                   │
│ - All writable data on tmpfs                                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 9.1 Key Architectural Decisions

1. **YubiHSM as universal trust anchor.** Same device, same PKCS#11 API, same attestation flow on both platforms.
2. **PCC-inspired privacy architecture.** Eliminate access vectors rather than encrypt memory. Vault Mode on DGX Spark achieves Apple PCC-equivalent privacy.
3. **Control plane and data plane stay separate.** Coordinator never touches inference data. HSM-managed session keys ensure even relayed traffic is opaque.
4. **Outbound-first connectivity.** Provider agent maintains outbound persistent connection. No inbound ports required.
5. **Relay-first MVP.** Direct P2P / QUIC is an optimization, not a prerequisite.
6. **Benchmark-driven scheduling.** Match on measured throughput, memory high-water mark, cold-start behavior, and cache status.
7. **One workload per node in public mode.** No multi-tenant promises (no MIG on either platform).
8. **Hardware-specific runtime stacks.** DGX Spark uses CUDA/TensorRT containers. Mac Studio uses MLX/Metal native processes. The agent abstracts the difference.
9. **Curated catalog first.** Public marketplace accepts only approved model/runtime combinations at launch.
10. **Progressive privacy.** Open Mode → Guarded Mode → Vault Mode. Providers choose their privacy level and earning potential.

---

## 10. Runtime Strategy

### 10.1 Backend Matrix

| Backend | DGX Spark | Mac Studio | Role |
|---------|-----------|------------|------|
| TensorRT-LLM | Primary | N/A | Highest performance on Blackwell. NVFP4 quantization. |
| vLLM / SGLang | Supported | N/A (vllm-mlx in development) | Broad open-model compatibility. SGLang has RadixAttention for superior batching. |
| MLX | N/A | Primary | Apple-native, best Metal performance. MLX distributed for multi-node. |
| llama.cpp | Supported (CUDA) | Supported (Metal) | Cross-platform fallback. GGUF models. MoE optimization. |
| Ollama | Supported | Supported | Convenience/hobbyist layer over llama.cpp. |
| NIM | Conditional (licensing) | N/A | Only if NVIDIA AI Enterprise license obtained. |

### 10.2 Recommended Backend Policy

- **Public marketplace default (Spark):** TensorRT-LLM for curated catalog models
- **Public marketplace default (Mac Studio):** MLX for curated catalog models
- **Compatibility fallback:** vLLM/SGLang (Spark), llama.cpp (both)
- **Private fleet:** Any backend allowed, including llama.cpp and Ollama for convenience
- **NIM:** Only if licensing story is resolved for commercial marketplace use

### 10.3 Model Packaging Policy

Public marketplace uses a curated catalog:
- Each model bundle tested on each supported hardware platform
- Associated with a specific runtime image digest
- Benchmark profiles stored per hardware + model + quantization + backend
- Prebuilt engines preferred over first-run engine builds on provider machines

Raw model pulls allowed only in: private-fleet mode, allowlisted-provider mode, or explicit experimental opt-in pools.

---

## 11. Capacity Planning and Scheduling

### 11.1 Hardware-Aware Scheduling

| Workload Type | Preferred Hardware | Reasoning |
|--------------|-------------------|-----------|
| Short prompts, fast response | DGX Spark | Compute advantage for prefill (100 TFLOPS vs 26) |
| Long generation, streaming | Mac Studio M3 Ultra | Bandwidth advantage for decode (819 GB/s vs 273) |
| Very large models (200B+) | Mac Studio M3 Ultra 256 GB | More memory |
| Quantized 8B–70B models | Either | Both capable; route by price/availability |
| Batch throughput | DGX Spark | Tensor Cores + SGLang linear batch scaling |
| Privacy-sensitive workloads | DGX Spark in Vault Mode | Only platform with extreme privacy |

### 11.2 Unified Memory Rules

Both platforms use unified memory and should NOT be scheduled like discrete-GPU servers:
- Do not rely on `nvidia-smi` alone for memory accounting (Spark)
- Do not hardcode "usable GPU memory" numbers
- Track host memory telemetry, cgroup limits, swap pressure, and measured high-water marks
- Keep configurable system reserve (OS + agent + HSM daemon overhead)
- Reject workloads based on observed safe envelopes, not marketing totals

### 11.3 Capacity Classes

| Class | Memory Requirement | Target |
|-------|-------------------|--------|
| Small | Fits comfortably in 64 GB | Either platform, fast interactive models (8B–14B) |
| Medium | Requires 64–110 GB | Single Spark or M4 Max 128 GB (32B–70B quantized) |
| Large | Requires 110–220 GB | M3 Ultra 256 GB, or dual-Spark (70B FP16, 200B quantized) |
| XL | Requires 220+ GB | Dual-Spark, or clustered Mac Studios via TB5 (405B+) |

### 11.4 Concurrency Policy

- **Public marketplace / dedicated lease:** one active workload per node
- **Private fleet:** controlled overcommit at operator's risk
- **No public SLA** on concurrent multi-model serving until real isolation data exists

---

## 12. Job Lifecycle

### 12.1 Public Marketplace Inference

```text
Consumer                     Coordinator                    Provider Agent
   │                              │                               │
   │ 1. Request deployment/job    │                               │
   │    (catalog ID, tier,        │                               │
   │     hardware pref, SLA)      │                               │
   │─────────────────────────────>│                               │
   │                              │ 2. Match node using           │
   │                              │    benchmark + policy +       │
   │                              │    HSM attestation check      │
   │                              │──────────────────────────────>│
   │                              │                               │
   │ 3. Receive route + manifest  │                               │
   │    + provider HSM public key │                               │
   │<─────────────────────────────│                               │
   │                              │                               │
   │ 4. SDK verifies HSM          │                               │
   │    attestation proof         │                               │
   │                              │                               │
   │ 5. Open E2E session (direct or relayed, HSM-keyed)           │
   │──────────────────────────────────────────────────────────────>│
   │                              │                               │
   │ 6. Invoke model              │                               │
   │──────────────────────────────────────────────────────────────>│
   │                              │                               │
   │ 7. Stream result + usage     │                               │
   │<──────────────────────────────────────────────────────────────│
   │                              │                               │
   │ 8. Metering + settlement     │                               │
   │─────────────────────────────>│                               │
   │                              │                               │
   │                              │ 9. On completion: wipe model  │
   │                              │    from memory (if not cached)│
```

### 12.2 Vault Mode Inference (Enhanced Privacy)

Same as above, with additional steps:
- Step 4 includes: verify boot measurements in attestation match transparency log for blessed DGInf OS image
- Step 5 includes: OHTTP relay for request anonymization + blind signature authentication
- Step 6: Request encrypted to HSM public key, decrypted only inside locked-down node
- Step 9: Crypto-shredding — model wiped, process address space recycled

### 12.3 Model Caching

- Cache by model digest + runtime digest on provider's NVMe storage
- Encrypt cache at rest with HSM-managed key
- Charge optional storage/cache fees for warm deployments
- Evict by LRU plus safety reserve
- In Vault Mode: cache on encrypted tmpfs or LUKS volume with ephemeral key

---

## 13. Networking

### 13.1 Home-Network Reality

Most providers sit behind NAT, dynamic IPs, consumer upload bottlenecks (asymmetric bandwidth), and variable latency.

### 13.2 MVP Design

- Provider agent maintains outbound persistent mTLS control channel (HSM-managed keys)
- Coordinator or regional relay exposes public ingress
- Data plane encrypted end-to-end (relay sees only opaque bytes)
- Model artifacts pulled by provider from registries (HuggingFace, S3), not uploaded by consumer
- In Vault Mode: OHTTP relay adds request anonymization layer

### 13.3 Hardware-Specific Networking

| Feature | DGX Spark | Mac Studio |
|---------|-----------|------------|
| Primary LAN | 10 GbE RJ-45 | 10 GbE (Nbase-T) |
| High-speed inter-node | ConnectX-7 (2x100G Ethernet, ~160–200 Gbps) | Thunderbolt 5 (120 Gbps, RDMA <50us) |
| Clustering latency | Standard TCP | <50 us (RDMA over TB5, macOS 26.2+) |
| Cluster topology | Point-to-point or Ethernet switch | Daisy-chain (no TB5 switches exist) |
| Practical bottleneck | 10 GbE for home internet serving | 10 GbE for home internet serving |

### 13.4 Networking Upgrades (Later)

- Direct QUIC when connectivity allows (hole-punching)
- Regional relay POPs for latency reduction
- Route selection based on measured RTT and uplink bandwidth
- Heterogeneous clusters (Spark + Mac Studio via EXO-style orchestration)

---

## 14. Pricing and Payments

### 14.1 Billing Units

- **Reserved endpoint hours** as primary provider-facing unit
- **Per-request minimum** for short interactive calls
- **Optional output-token surcharge** for streaming generations
- **Cache/storage charges** for warm model persistence
- **Privacy premium** — Vault Mode nodes command higher rates

### 14.2 Provider Economics

| Item | DGX Spark | Mac Studio M4 Max 128GB | Mac Studio M3 Ultra 256GB |
|------|-----------|------------------------|--------------------------|
| Hardware | ~$4,699 | $3,699 | ~$5,999 |
| YubiHSM 2 | $650 | $650 | $650 |
| Total upfront | ~$5,349 | $4,349 | ~$6,649 |
| Power (continuous) | ~$15–26/mo | ~$15–22/mo | ~$15–22/mo |
| Break-even (50% util) | ~15 mo | ~20 mo | ~18 mo |

### 14.3 Marketplace Presentation

Display per provider:
- Hardware type, memory, and privacy mode
- Hourly price
- Measured prefill throughput (tok/s)
- Measured decode throughput (tok/s)
- Warm/cold start behavior
- Effective cost per million output tokens
- HSM attestation status and privacy mode badge
- Route type (direct/relayed)
- Uptime and reputation score

### 14.4 Payments Roadmap

**Phase 1 (MVP):** Stripe fiat payments + internal ledger. Simple, works globally.

**Phase 2 (If demand exists):** Stablecoin settlement on **Base L2** (cheapest EVM L2, sub-cent fees, Coinbase fiat on-ramp, native USDC).

Researched EVM payment architecture for Phase 2:

| Layer | Choice | Rationale |
|-------|--------|-----------|
| L2 Chain | **Base** | Lowest fees ($0.001–0.005/tx), highest L2 volume, native USDC |
| Stablecoin | **USDC (native on Base)** | Regulated, fiat-redeemable, deep liquidity |
| Streaming payments | **Superfluid** | Per-second continuous flow with zero gas between start/stop |
| High-freq settlement | **ERC-7824 state channels** | Zero per-payment cost for bilateral sessions |
| Gas abstraction | **ERC-4337 + EIP-7702** | Users pay gas in USDC, never hold ETH |
| Escrow | **iExec PoCo-inspired** | Dual-deposit escrow, open-source, audited (Consensys) |

Sources: [L2Fees.info](https://l2fees.info/), [Superfluid Docs](https://docs.superfluid.org/), [iExec PoCo](https://github.com/iExecBlockchainComputing/PoCo), [ERC-7824](https://ethereum-magicians.org/t/erc-7824-state-channels-framework/22566)

Crypto should not be a launch priority. It becomes relevant only if product-market fit exists and provider demand justifies it.

---

## 15. Provider Agent — Detailed Design

### 15.1 Configuration

```yaml
# ~/.dginf/config.yaml
provider:
  name: "node-01"
  hardware: "dgx_spark"           # dgx_spark | mac_studio_m4max | mac_studio_m3ultra
  mode: "public_marketplace"      # public_marketplace | dedicated_lease | allowlisted | private_fleet
  privacy_mode: "vault"           # open | guarded | vault (vault = DGX Spark only)
  region: "us-west"

hsm:
  type: "yubihsm2"
  connector: "usb"
  attestation_interval_sec: 300
  audit_log_export: true

availability:
  schedule: "always"              # or cron: "0 22 * * * - 0 8 * * *" (10pm-8am)
  max_active_workloads: 1
  idle_timeout_min: 15            # macOS: resume after N minutes idle

runtime:
  backends:
    - "trtllm"                    # DGX Spark
    # - "mlx"                     # Mac Studio
    # - "llamacpp"                # Both
  allow_custom_images: false
  allow_raw_model_refs: false

catalog:
  classes:
    - "small"
    - "medium"

storage:
  cache_path: "/var/lib/dginf/cache"
  cache_max_gb: 600
  encrypted_cache: true

network:
  relay_only: true
  direct_quic_opt_in: false

security:
  signed_images_only: true
  wipe_on_stop: true
  no_prompt_logging: true
  allowed_registries:
    - "huggingface.co"
    - "nvcr.io"

pricing:
  endpoint_hourly_usd: 0.50
  request_minimum_usd: 0.001
  output_token_1m_usd: 20.00
```

### 15.2 Platform-Specific Behavior

| Concern | DGX Spark (Linux) | Mac Studio (macOS) |
|---------|-------------------|-------------------|
| Agent form factor | CLI daemon (systemd) | macOS app (menu bar + GUI) |
| Runtime isolation | OCI containers + NVIDIA Container Toolkit | Native processes + macOS sandbox profiles |
| Boot measurement | fTPM PCR values via tpm2-tools | MDM attestation + software integrity checks |
| Vault Mode OS | Custom dm-verity image, kernel lockdown | Not available (boot measurement gap) |
| GPU access | CUDA / TensorRT via container GPU passthrough | Metal via native framework |
| Model cache encryption | LUKS volume, key in HSM | APFS encrypted volume, key in HSM |
| Process supervision | systemd | launchd |
| Idle detection | N/A (always-on daemon) | NSWorkspace notifications for user activity |

### 15.3 Resource Management

- Backend-specific memory envelopes (measured, not theoretical)
- Cache quotas and LRU eviction
- cgroup isolation (Linux) / sandbox profiles (macOS) for runtime process
- Thermal and power guardrails (throttle if GPU temp exceeds threshold)
- Model-digest verification before loading
- Cooldown and health backoff after repeated OOM or driver faults

### 15.4 Public-Mode Safety Rules

If `mode=public_marketplace` or `mode=dedicated_lease`, enforce:
- `max_active_workloads=1`
- `allow_custom_images=false`
- `allow_raw_model_refs=false`
- `signed_images_only=true`
- No prompt logging
- HSM attestation required
- Wipe-on-stop mandatory

---

## 16. Coordinator — Detailed Design

### 16.1 API Surface

```text
Provider APIs:
  POST   /v1/providers/register         — HSM public key + signed attestation
  POST   /v1/providers/heartbeat        — health, capacity, HSM audit hash
  POST   /v1/providers/benchmarks       — upload benchmark profiles
  POST   /v1/providers/attest           — periodic re-attestation

Consumer APIs:
  POST   /v1/deployments                — deploy model (catalog ID, tier, hardware pref, privacy req)
  POST   /v1/jobs                       — submit inference job
  GET    /v1/jobs/{id}                  — job status + HSM attestation proof
  GET    /v1/deployments/{id}           — deployment status
  DELETE /v1/deployments/{id}           — undeploy + trigger wipe
  GET    /v1/models/catalog             — browse catalog by hardware
  GET    /v1/providers/marketplace      — browse providers + pricing + privacy mode + HSM status
  POST   /v1/models/encrypt             — encrypt model to specific HSM public key (Tier 3)

Payments APIs:
  POST   /v1/payments/deposit
  GET    /v1/payments/balance
  POST   /v1/payments/withdraw
```

### 16.2 Matching Algorithm

1. Filter by: trust tier, hardware type, online status, HSM-attested, provider mode, privacy mode
2. Filter by: compatible runtime/backend, catalog class, sufficient memory
3. Filter by: benchmark envelope, route health
4. Prefer: warm cache hits, lower cold-start cost
5. Rank by: price, region, reputation, expected latency, hardware match for workload type
6. For Vault Mode requests: only match DGX Spark nodes in Vault Mode
7. Allocate top candidate; retry on reject or route failure

### 16.3 Reputation System

Score based on operational reality:
- Uptime (% of advertised availability)
- Job success rate
- Actual vs. claimed performance (benchmark drift detection)
- Route reliability
- HSM attestation consistency (never failed re-attestation)
- Policy violations

---

## 17. Security and Compliance

### 17.1 Baseline Controls (All Tiers)

- YubiHSM-rooted node identity
- End-to-end mTLS (HSM-managed session keys)
- Signed runtime artifacts (Sigstore Cosign)
- Verified model digests
- Encrypted cache (HSM-gated)
- Wipe-on-stop
- No prompt logging by default
- Rate limits and balance requirements

### 17.2 Vault Mode Additional Controls

- Immutable dm-verity root filesystem
- UEFI Secure Boot + fTPM measured boot
- Kernel lockdown (confidentiality mode)
- No SSH, no console, no serial, no emergency shell
- No ptrace, no core dumps, no debug tools
- Encrypted swap with random per-boot key (crypto-shredding)
- All writable data on tmpfs
- OHTTP relay for request anonymization
- RSA blind signatures for auth
- Transparency log for OS images
- Log filter daemon (no user data in logs)

### 17.3 Public Marketplace Policy

- Approved open/public models only (Tier 2) or HSM-encrypted private models (Tier 3)
- Clear disclosure of privacy level per provider (Open / Guarded / Vault badge)
- Region filtering
- HSM attestation required for all marketplace providers

### 17.4 Legal / Licensing Workstreams

- macOS EULA: validate dedicated-lease model for Mac Studio inference (seek Apple legal guidance)
- NVIDIA NIM: validate commercial marketplace use or stay on open-source runtimes
- Export control: obligations for certain models or destinations
- Data processing: privacy obligations if prompts/outputs contain regulated data
- YubiHSM FIPS: validate whether FIPS 140-2 Level 3 certification is sufficient for target markets
- Apple MDM: validate DGInf operating as MDM provider and Managed Apple Account workflow for User Enrollment / Device Enrollment on customer-owned Macs

---

## 18. Competitive Landscape

### 18.1 Detailed Competitor Analysis

| Platform | Focus | Verification Model | TEE Support | Payment | Hardware | Status | Revenue |
|----------|-------|-------------------|-------------|---------|----------|--------|---------|
| **Akash** | General compute | Reputation (TEE in dev via AEP-29) | AMD SEV-SNP, Intel TDX, NVIDIA NVTrust (by May 2026) | AKT (4% take) / USDC (20% take) | H100, H200, A100, RTX 4090 | Production, most mature | Daily fees >$13K |
| **io.net** | GPU aggregation | PoW + PoTL + staking + slashing | None | IO / USDC (2% fee) | 327K GPUs, Apple M-series | Production | — |
| **Render** | Rendering + AI | Reputation + output check | None | RENDER token (BME) | Consumer to H200, MI300X | Production, pivoting to AI | 35% from Hollywood studios |
| **Gensyn** | ML training | Verde (RepOps + refereed delegation) | None (math-based) | AI token | Heterogeneous | Testnet (Mar 2025) | Pre-revenue |
| **Ritual** | AI-native blockchain | ZK + TEE + optimistic + probabilistic | Intel SGX, AWS Nitro | On-chain | Heterogeneous | Production (8K+ nodes) | — |
| **Bittensor** | AI quality market | Yuma Consensus (stake-weighted) | None | TAO + subnet alpha tokens | RTX 4090 to B200 | Production, 128+ subnets | — |
| **Nosana** | AI inference | PoW + escrow | None | NOS token (Solana) | NVIDIA RTX 30+, AMD RX 6000+ | Production, 50K+ nodes | — |
| **Aethir** | Enterprise GPU | Zero-trust Checker Nodes (91K) | Via iExec partnership | ATH + AUSD stablecoin | Enterprise GPUs (435K containers) | Production | ~$40M/quarter |
| **Hyperbolic** | Verifiable inference | Proof of Sampling (PoSP) | None | Own token (planned) | Aggregated | Production | 1B+ tokens/day |
| **Fluence** | Enterprise cloud | Crypto proofs + cross-validation | None | USDC + FLT | VMs, GPU containers | Production | $4M+ customer savings |

Sources: Platform websites, documentation, and announcements (2025–2026).

### 18.2 DGInf Differentiators

| Differentiator | What It Means | Why Competitors Don't Have It |
|---------------|--------------|------------------------------|
| **Unified-memory-specific scheduling** | Route based on prefill vs decode, memory bandwidth vs compute | Every competitor treats GPUs as generic FLOPS. None optimizes for 128–256 GB unified memory. |
| **YubiHSM trust anchor** | Hardware-rooted identity + model decryption at $650/node | Akash's TEE roadmap is in development. Most use reputation only. No competitor uses HSM. |
| **Cross-platform (CUDA + Metal)** | Route to DGX Spark for compute, Mac Studio for bandwidth | No competitor supports Apple Silicon + NVIDIA in the same marketplace. |
| **PCC-inspired privacy (Vault Mode)** | Apple PCC-equivalent privacy without hardware TEE | No competitor has this. Closest is Aethir via iExec TEE partnership (enterprise-only, not hobbyist). |
| **Hobbyist economics** | $3K–5K device + $650 HSM = earning machine | Most competitors target data centers or require $20K+ GPUs. |
| **Honest trust model** | Tiered trust with transparent limitations | Many competitors claim "decentralized" or "secure" without hardware backing. |

### 18.3 Things That Are Not Differentiators Today

- Confidential computing claims on Spark or Mac Studio (neither supports CC)
- Tokenomics (crypto should follow product-market fit, not lead)
- "Decentralization" as a brand word (meaningless without hardware reality)

---

## 19. Feasibility and Risk Assessment

### 19.1 What Is Real and Proven

| Component | Status | Evidence |
|-----------|--------|---------|
| YubiHSM 2 | Ships today, $650, FIPS 140-2 Level 3 | Yubico product page, PKCS#11 on Linux + macOS |
| dm-verity | Production | Powers Android Verified Boot and ChromeOS (billions of devices) |
| UEFI Secure Boot | Production | Standard on all modern hardware |
| Kernel lockdown LSM | Production | In Linux kernel since 5.4. DGX OS runs kernel 6.17. |
| Encrypted swap (random per-boot key) | Production | Standard Linux crypttab with /dev/urandom |
| No SSH / no shell | Trivial | Remove openssh-server, mask getty services |
| ptrace disabled | Trivial | sysctl kernel.yama.ptrace_scope=3 |
| OHTTP relay | RFC 9458, production | Cloudflare runs these. OpenPCC implements it. |
| RSA Blind Signatures | RFC 9474, production | Standard crypto, libraries exist |
| Sigstore / Rekor transparency log | Production | Used by Kubernetes, npm, PyPI ecosystems |
| E2E encryption | Trivial | Standard TLS 1.3 + HSM-managed keys |
| macOS SIP + KIP + SSV | Production | Built into every Apple Silicon Mac, hardware-enforced |
| MLX inference | Production | Apple's framework, active development |
| vLLM / SGLang on DGX Spark | Validated | Official NVIDIA playbooks |

### 19.2 Needs Verification on DGX Spark

| Component | Risk Level | What Could Go Wrong |
|-----------|-----------|-------------------|
| **NVIDIA GPU driver + kernel lockdown** | **HIGH** | #1 unknown. NVIDIA's proprietary kernel module must load under lockdown. Nobody has publicly tested dm-verity + kernel lockdown + NVIDIA GPU driver together. If driver breaks, Vault Mode architecture needs rethinking. |
| **fTPM on DGX Spark** | MEDIUM | Confirmed on Jetson (same NVIDIA lineage). NVIDIA confirmed Secure Boot on Spark. But fTPM specifically needs hands-on verification. |
| **dm-verity with DGX OS + CUDA stack** | MEDIUM | dm-verity works on Linux. But building a dm-verity image with the full DGX stack (CUDA 13.0, driver 580.x, container toolkit) is uncharted territory. |
| **YubiHSM sealing to fTPM PCR values** | MEDIUM | YubiHSM doesn't natively read TPM PCRs. Need software bridge: agent reads PCRs → presents to HSM → HSM validates. This is custom integration, not out-of-box. |

### 19.3 Significant Engineering Required

| Component | Effort | Notes |
|-----------|--------|-------|
| Custom hardened DGX OS image (Vault Mode) | 2–3 months | Custom Linux distro with dm-verity, NVIDIA drivers, CUDA, inference frameworks, zero interactive access. Effectively becoming a distro maintainer. |
| OTA update mechanism | 4–8 weeks | A/B partition scheme + remote-triggered updates + rollback. Providers can't SSH in to update. |
| Provider UX with zero shell access | 4–8 weeks | Provider's $5K machine becomes appliance. Need: LAN web dashboard, LED status, self-healing, error communication. |
| macOS GUI app (menu bar + dashboard) | 4–6 weeks | Native macOS app with idle detection, HSM integration, benchmark runner. |
| Consumer attestation verification SDK | 4–6 weeks | Client verifies HSM signature → boot measurements → transparency log → encrypts to specific node. |
| OHTTP relay infrastructure | 2–4 weeks | OpenPCC provides starting point. Need deployment + operations. |
| Reproducible builds | 4–6 weeks | Deterministic image builds for transparency log. NVIDIA proprietary blobs complicate this. |
| Mac Studio Guarded Mode (MDM) | 3–6 weeks | MDM enrollment flow, configuration profiles, attestation integration. |

### 19.4 Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|-----------|
| NVIDIA driver breaks under kernel lockdown | 30–50% | Showstopper for Vault Mode | **Validate in week 1.** If fails: negotiate with NVIDIA, or limit Vault Mode to inference-only containers with minimal driver surface. |
| fTPM not available on DGX Spark | 20% | High | Fall back to software measurements + YubiHSM-only attestation. Weaker but functional. |
| Provider UX unacceptable ("I can't use my $5K machine") | 40% | High | Offer clear mode switching: Open Mode (full access, lower earnings) vs Vault Mode (appliance, higher earnings). Provider chooses. |
| Cold boot attack on unified memory | 5% | Medium | LPDDR5x soldered into SoC — physically near-impossible to probe. Low real-world risk. |
| Custom OS maintenance burden | 60% | High | Start with DGX Spark only. Defer Mac Studio Vault Mode. Partner with NVIDIA for driver compatibility. |
| Apple EULA blocks Mac Studio marketplace | 30% | Medium | Stay safe with dedicated-lease only. Seek legal guidance. |
| YubiHSM supply/pricing at scale | 20% | Low | Negotiate bulk pricing. Alternative: explore cheaper HSMs if Yubico can't meet demand. |

### 19.5 Week-1 Validation Plan

Before writing production code, validate these three critical unknowns on a real DGX Spark:

```bash
# Test 1: NVIDIA Driver + Kernel Lockdown
# Can NVIDIA GPU driver function under kernel lockdown?
echo "lockdown=confidentiality" >> /etc/default/grub
update-grub && reboot
nvidia-smi                    # Does driver load?
python3 -c "import torch; print(torch.cuda.is_available())"  # Does CUDA work?
# Launch vLLM with a small model — does inference work?

# Test 2: fTPM Verification
# Does DGX Spark have a TPM?
ls /dev/tpm*
tpm2_getcap properties-fixed
tpm2_pcrread                  # Can we read boot measurements?

# Test 3: YubiHSM on ARM64 Linux
# Does YubiHSM work on DGX Spark?
apt install yubihsm-connector yubihsm-shell
yubihsm-connector -d
yubihsm-shell                 # Generate key, sign data, verify PKCS#11
```

**If Test 1 fails:** Vault Mode architecture needs rethinking (possibly run NVIDIA driver in a minimal container with reduced lockdown, or negotiate with NVIDIA for signed-module support under lockdown).

**If Test 2 fails:** Fall back to YubiHSM-only attestation (software boot measurements presented to HSM instead of TPM PCR sealing).

**If Test 3 fails:** YubiHSM SDK may need ARM64 porting (unlikely — PKCS#11 is platform-independent, and Yubico supports Linux ARM).

---

## 20. Roadmap

### Phase 1: Private Fleet + Curated Serving + HSM Trust (0–3 months)

- [ ] **Week 1: Hardware validation** — NVIDIA driver + lockdown, fTPM, YubiHSM on DGX Spark
- [ ] Provider agent for DGX Spark (Linux) — registration, health, relay, HSM integration
- [ ] Provider agent for Mac Studio (macOS) — GUI app, menu bar, idle detection
- [ ] YubiHSM PKCS#11 integration — identity, attestation
- [ ] Open Mode on both platforms
- [ ] Curated model catalog (TensorRT-LLM for Spark, MLX for Mac Studio)
- [ ] Benchmark harness and benchmark-driven scheduling
- [ ] Consumer CLI / SDK (Python)
- [ ] Relay-first networking
- [ ] Fiat payments (Stripe) and internal ledger

**Goal:** Best way to operate self-owned and allowlisted DGX Spark + Mac Studio nodes.

### Phase 2: Public Marketplace + Guarded Mode (3–6 months)

- [ ] Public marketplace mode for DGX Spark (multi-job, open models)
- [ ] Dedicated lease mode for Mac Studio (single-customer, 24h minimum)
- [ ] Guarded Mode on Mac Studio (MDM profiles, restricted access)
- [ ] Warm cache management and deployment persistence
- [ ] Marketplace pages with benchmark-derived economics, HSM status, privacy badges
- [ ] vLLM/SGLang compatibility pool (Spark), vllm-mlx (Mac Studio)
- [ ] Provider dashboard and earnings tracking
- [ ] Provider companion app (mobile — monitor earnings, toggle availability)

**Goal:** Validate whether HSM-attested supply has real consumer demand.

### Phase 3: Vault Mode + Private Models (6–9 months)

- [ ] **Vault Mode on DGX Spark** — full PCC-inspired privacy stack
  - [ ] Custom hardened DGX OS image (dm-verity, kernel lockdown, no shell)
  - [ ] fTPM PCR sealing to YubiHSM
  - [ ] Encrypted swap with crypto-shredding
  - [ ] OTA update mechanism (A/B partitions)
  - [ ] Full-screen status dashboard for locked-down mode
- [ ] HSM-controlled model decryption (Tier 3 private model uploads)
- [ ] OHTTP relay for request anonymization
- [ ] RSA blind signatures for auth
- [ ] Transparency log for OS images (Sigstore/Rekor)
- [ ] Consumer SDK attestation verification
- [ ] Reproducible builds

**Goal:** Apple PCC-equivalent privacy on DGX Spark. Enable private model marketplace.

### Phase 4: Premium Capacity + Enterprise (9–12 months)

- [ ] Dual-Spark and dual-Mac-Studio dedicated endpoints
- [ ] Heterogeneous scheduling (route prefill→Spark, decode→Mac Studio)
- [ ] Direct QUIC where connectivity allows
- [ ] Regional relay POPs
- [ ] Reputation system tied to performance and HSM audit consistency
- [ ] Single-node LoRA / QLoRA fine-tuning
- [ ] Private network / VPC-style routing
- [ ] Audit exports and policy packs
- [ ] Team accounts and fleet controls

**Goal:** Premium high-memory deployments and enterprise controls.

### Phase 5: Clusters + Payments + Expansion (12+ months)

- [ ] Cross-hardware clusters (Spark + Mac Studio via EXO-style orchestration)
- [ ] Thunderbolt 5 RDMA for Mac Studio clusters
- [ ] ConnectX-7 networking for Spark clusters
- [ ] Optional stablecoin settlement on Base L2 (Superfluid, ERC-4337)
- [ ] Support for future CC-capable hardware (NVIDIA Vera Rubin) as Tier 4
- [ ] Explore Vault Mode on Mac Studio if Apple adds attestation API

**Goal:** Rack-scale inference across mixed hardware with optimal routing.

### Deliberately Deferred

- Public-marketplace fine-tuning (Phase 4 is private fleet only)
- WAN-distributed training across home nodes
- Smart-contract-heavy settlement at launch
- Full confidential execution claims (until hardware supports it)
- Vault Mode on Mac Studio (until macOS exposes boot measurement API)

---

## 21. Tech Stack Summary

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Provider Agent (Linux) | Rust | Cross-platform, performance, PKCS#11 bindings (yubihsm-rs) |
| Provider Agent (macOS) | Swift + Rust core | Native macOS UI (SwiftUI menu bar app) + Rust core for HSM/crypto |
| HSM Integration | YubiHSM 2 via PKCS#11 | Standard API, identical on Linux and macOS |
| DGX Spark Runtime | Docker/containerd + NVIDIA Container Toolkit | GPU passthrough, OCI packaging |
| Mac Studio Runtime | Native processes + macOS sandbox | Metal requires native access, no Docker GPU passthrough on macOS |
| Primary Backend (Spark) | TensorRT-LLM | Best Blackwell performance, NVFP4 quantization |
| Primary Backend (Mac Studio) | MLX | Best Metal performance, Apple-native |
| Cross-Platform Backend | llama.cpp | CUDA + Metal, GGUF models |
| Coordinator | Rust or Go | Control-plane concurrency and networking |
| Database | PostgreSQL | Providers, deployments, billing, HSM public keys |
| Eventing | Redis Streams or NATS | Async job/control events |
| Consumer SDK / CLI | Python (Typer) | ML ecosystem standard |
| OHTTP Relay | OpenPCC-derived (Rust) | Request anonymization for Vault Mode |
| Transparency Log | Sigstore / Rekor | OS image version auditing |
| Monitoring | Prometheus + Grafana | Fleet observability |
| Artifact Signing | Sigstore Cosign | Runtime and model integrity |
| Vault Mode OS | Custom DGX OS image (dm-verity, kernel lockdown) | PCC-equivalent privacy |
| Payments (Phase 1) | Stripe + internal ledger | Fastest sane MVP path |
| Payments (Later) | Base L2 + USDC + Superfluid + ERC-4337 | If crypto settlement is demanded |
| Escrow (Later) | iExec PoCo-inspired Solidity | Battle-tested compute marketplace escrow |

---

## 22. Open Questions

1. **NVIDIA driver + kernel lockdown:** Does the NVIDIA GPU driver work under Linux kernel lockdown (confidentiality mode) on DGX Spark? This is the #1 technical risk. Must validate in week 1.
2. **fTPM on DGX Spark:** Is fTPM available and functional? Can we read PCR values and seal HSM keys to boot state?
3. **Wedge product:** Is the real wedge a public marketplace or a private-fleet control plane? The HSM + Vault Mode story may make public marketplace viable earlier.
4. **Mac Studio EULA:** Should we seek Apple legal guidance on AI inference as a "permitted developer service"? Or stay safe with dedicated-lease-only?
5. **Catalog strategy:** Which models create strongest early demand — small fast (8B), high-memory (70B), or 200B+ that only M3 Ultra 256GB can serve?
6. **YubiHSM supply chain:** Bulk pricing? Alternative HSMs if Yubico can't meet demand?
7. **HSM key backup:** YubiHSM supports M-of-N key wrap. Mandate backup, or accept lost HSM = new identity?
8. **Heterogeneous routing:** "Route prefill to Spark, decode to Mac Studio" — worth the orchestration cost in V1 or later optimization?
9. **NIM licensing:** Worth pursuing NVIDIA AI Enterprise for marketplace, or stay on open-source runtimes?
10. **Vault Mode UX:** How do providers feel about their $5K machine becoming an appliance? Need user research.
11. **OHTTP relay operator:** Run our own or partner with Cloudflare/Fastly? Cost and trust implications.
12. **Tamper switch:** Add $20 hardware tamper detection (chassis open = HSM key wipe) as optional provider upgrade?

---

## 23. Sources

### Hardware — DGX Spark
- [DGX Spark Release Notes](https://docs.nvidia.com/dgx/dgx-spark/release-notes.html)
- [DGX Spark Porting Guide — Software Stack](https://docs.nvidia.com/dgx/dgx-spark-porting-guide/porting/software-requirements.html)
- [DGX Spark Playbooks](https://github.com/NVIDIA/dgx-spark-playbooks)
- [LMSYS In-Depth Review](https://lmsys.org/blog/2025-10-13-nvidia-dgx-spark/)
- [NVIDIA Performance Blog](https://developer.nvidia.com/blog/how-nvidia-dgx-sparks-performance-enables-intensive-ai-tasks/)
- [NVIDIA Software Optimizations Blog](https://developer.nvidia.com/blog/new-software-and-model-optimizations-supercharge-nvidia-dgx-spark/)
- [NVIDIA DGX Spark CC Limitation](https://forums.developer.nvidia.com/t/confidential-computing-support-for-dgx-spark-gb10/347945)
- [NVIDIA Grace CPU Arm CCA Limitation](https://forums.developer.nvidia.com/t/does-the-grace-cpu-support-arm-cca/272111)
- [NVIDIA MIG on DGX Spark](https://forums.developer.nvidia.com/t/is-the-nvidia-dgx-spark-system-compatible-with-mig-technology/340089)
- [Tom's Hardware — Price Increase](https://www.tomshardware.com/desktops/mini-pcs/nvidia-dgx-spark-gets-18-percent-price-increase)

### Hardware — Mac Studio
- [Apple Mac Studio Specs](https://www.apple.com/mac-studio/specs/)
- [Apple Mac Studio Newsroom](https://www.apple.com/newsroom/2025/03/apple-unveils-new-mac-studio-the-most-powerful-mac-ever/)
- [EXO Labs: DGX Spark + Mac Studio Benchmarks](https://blog.exolabs.net/nvidia-dgx-spark/)
- [Jeff Geerling: 1.5 TB VRAM Mac Studio RDMA over TB5](https://www.jeffgeerling.com/blog/2025/15-tb-vram-on-mac-studio-rdma-over-thunderbolt-5/)
- [vllm-mlx GitHub](https://github.com/waybarrios/vllm-mlx)
- [EXO GitHub](https://github.com/exo-explore/exo)

### Trust / Security
- [YubiHSM 2 Product Page](https://www.yubico.com/product/yubihsm-2/)
- [YubiHSM 2 SDK](https://developers.yubico.com/YubiHSM2/)
- [Apple Secure Enclave Security Guide](https://support.apple.com/guide/security/the-secure-enclave-sec59b0b31ff/web)
- [Apple Signed System Volume](https://support.apple.com/guide/security/signed-system-volume-security-secd698747c9/web)
- [Apple Kernel Integrity Protection](https://support.apple.com/guide/security/secb7ea06b49/web)
- [Apple DMA Protections](https://support.apple.com/guide/security/direct-memory-access-protections-seca4960c2b5/web)
- [Apple Private Cloud Compute](https://security.apple.com/blog/private-cloud-compute/)
- [Apple PCC Documentation](https://security.apple.com/documentation/private-cloud-compute/)
- [Apple PCC Security Research](https://security.apple.com/blog/pcc-security-research/)
- [OpenPCC GitHub (Apache 2.0)](https://github.com/openpcc/openpcc)
- [IronCore Labs: Apple PCC Analysis](https://ironcorelabs.com/blog/2024/apple-confidential-ai/)

### Confidential Computing Research
- [NVIDIA CC on H100](https://developer.nvidia.com/blog/confidential-computing-on-h100-gpus-for-secure-and-trustworthy-ai/)
- [GPU CC Demystified (arXiv)](https://arxiv.org/html/2507.02770v1)
- [Confidential LLM Inference: CPU and GPU TEEs (arXiv)](https://arxiv.org/abs/2509.18886)
- [Super Protocol: GPU+CPU TEE Requirements](https://superprotocol.com/resources/gpu-cpu-tee-requirements)
- [Level1Techs: Threadripper PRO 9000 SEV-SNP (Fails)](https://forum.level1techs.com/t/enabling-confidential-computing-features-on-threadripper-pro-9000-series/234586)
- [NVIDIA Vera Rubin NVL72](https://www.nvidia.com/en-us/data-center/vera-rubin-nvl72/)

### Privacy Alternatives Research
- [TZ-LLM: ARM TrustZone for LLMs (arXiv)](https://arxiv.org/html/2511.13717v1)
- [EncryptedLLM: FHE on GPT-2 (ICML 2025)](https://proceedings.mlr.press/v267/de-castro25a.html)
- [Zama/HuggingFace: Encrypted LLM](https://huggingface.co/blog/encrypted-llm)
- [PermLLM: Private Inference in 3s (arXiv)](https://arxiv.org/html/2405.18744v1)
- [zkLLM: ZK Proofs for LLMs (arXiv)](https://arxiv.org/abs/2404.16109)
- [DeepProve-1: First Full LLM zkML Proof](https://lagrange.dev/blog/deepprove-1)
- [Linux kernel_lockdown(7)](https://www.man7.org/linux/man-pages/man7/kernel_lockdown.7.html)
- [Keylime Remote Attestation](https://keylime.dev/)

### Competitive
- [Akash Network](https://akash.network/) — [AEP-29 TEE Roadmap](https://akash.network/roadmap/aep-29/)
- [io.net](https://io.net/)
- [Render Network](https://rendernetwork.com/)
- [Gensyn](https://www.gensyn.ai/) — [Verde Verification](https://www.gensyn.ai/articles/verde)
- [Ritual](https://www.ritualfoundation.org/) — [Infernet SDK](https://github.com/ritual-net/infernet-sdk)
- [Bittensor Docs](https://docs.learnbittensor.org/)
- [Nosana](https://nosana.com/)
- [Aethir](https://aethir.com/)
- [Hyperbolic](https://hyperbolic.xyz/)
- [Fluence Network](https://www.fluence.network/)

### Payments (Future Reference)
- [L2Fees.info](https://l2fees.info/)
- [Superfluid Docs](https://docs.superfluid.org/)
- [ERC-7824 State Channels](https://ethereum-magicians.org/t/erc-7824-state-channels-framework/22566)
- [iExec PoCo Escrow (GitHub)](https://github.com/iExecBlockchainComputing/PoCo)
- [ERC-4337 Account Abstraction](https://docs.erc4337.io/)
- [Circle: Native USDC](https://www.circle.com/usdc)
- [Sablier Token Streaming](https://blog.sablier.com/overview-token-streaming-models/)
