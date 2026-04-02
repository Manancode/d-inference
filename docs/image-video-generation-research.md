# Open-Source Image & Video Generation Research (March 2026)

Research into integrating image/video generation capabilities. Covers best open-source models, VRAM requirements, performance on consumer hardware, and MLX ecosystem readiness.

---

## Image Generation Models (2025/2026)

### Top Picks by VRAM Tier

**8 GB or less:**
| Model | Params | VRAM | Speed (4090) | License |
|-------|--------|------|-------------|---------|
| FLUX.2 Klein 4B | 4B | ~8 GB (FP8) | **~1 sec** | Apache 2.0 |
| Z-Image-Turbo | 6B | ~6 GB (GGUF) | ~4-6 sec | Apache 2.0 |
| PixArt-Sigma | 0.6B | <8 GB native | ~3 sec | Open |
| SDXL | 2.6B | 8 GB native | ~8 sec | RAIL-M |

**12-24 GB:**
| Model | Params | VRAM | Speed (4090) | License |
|-------|--------|------|-------------|---------|
| FLUX.2 Klein 9B | 9B | ~12 GB (FP8) | **~2 sec** | Apache 2.0 |
| FLUX.1 Dev | 12B | ~13 GB (FP8) | ~14 sec | Non-commercial |
| SD 3.5 Large | 8B | ~18 GB | ~8-12 sec | Free < $1M rev |

**Frontier (24+ GB):**
| Model | Params | VRAM | Notes |
|-------|--------|------|-------|
| FLUX.2 Dev | 32B | ~64 GB | Best overall open-weight quality |
| Qwen-Image 2.0 | 20B | ~40 GB (4 GB w/ offload!) | #1 on AI Arena, Apache 2.0 |
| HunyuanImage-3.0 | 80B MoE | ~240 GB full / 24 GB quant | Largest open image model |

### Quality Ranking (community consensus, early 2026)
1. **FLUX.2 Dev** (32B) - highest visual fidelity
2. **Qwen-Image 2.0 Pro** - best text rendering, multilingual
3. **GLM-Image** - 91% text accuracy vs FLUX's 50%
4. **Z-Image-Turbo** - matches Tier 1 quality at fraction of resources

### Detailed Model Specs

#### FLUX Family (Black Forest Labs)

**FLUX.1 (August 2024):**
- Architecture: Flow Matching (Rectified Flow Transformer)
- FLUX.1 Dev: 12B params, ~24 GB FP16, ~13 GB Q8/FP8, ~8 GB GGUF Q4
- FLUX.1 Schnell: 12B params, Apache 2.0, only 4 inference steps (~5.5 sec on 4090)

**FLUX.2 (November 2025):**
- FLUX.2 Dev: 32B, coupled with Mistral-3 24B VLM, ~90 GB full / ~64 GB lowVRAM
- FLUX.2 Klein 4B: ~13 GB native, ~8 GB FP8, ~1-1.2 sec on 4090, Apache 2.0
- FLUX.2 Klein 9B: ~20 GB native, ~12 GB FP8, ~2 sec on 4090, Apache 2.0

#### Stable Diffusion 3.5 (October 2024)
- SD 3.5 Large: 8B params, MMDiT architecture, ~18 GB FP16, ~11 GB TensorRT FP8
- SD 3.5 Medium: 2.6B, ~10 GB, quantized ~8 GB
- SD 3.5 Large Turbo: ~3-5 sec on 4090

#### Chinese Lab Models

**Qwen-Image (Alibaba):** 20B MMDiT, ~40-48 GB full, ~4 GB with CPU offload (DiffSynth-Studio), native 1328x1328, Apache 2.0. #1 on AI Arena for text-to-image.

**Z-Image-Turbo (Alibaba, Dec 2025):** 6B S3-DiT, 8-step inference, 12-16 GB native, ~6 GB GGUF, sub-second to 4-6 sec. Matches or exceeds FLUX.2 dev quality. Apache 2.0.

**HunyuanImage-3.0 (Tencent, Sep 2025):** 80B MoE (64 experts, ~13B active), 3x 80 GB GPUs recommended, 24 GB quantized FP8. Excels at knowledge-intensive generation.

**GLM-Image (Zhipu AI, Jan 2026):** 16B (9B AR + 7B DiT), 91.16% text rendering accuracy vs FLUX's 49.65%. Best for dense text, typography, infographics.

#### Lightweight Models
- **PixArt-Sigma:** 0.6B, under 8 GB, up to 4K natively, ~3 sec per image
- **Playground 2.5:** Based on SDXL, ~8-12 GB, trained to match Midjourney aesthetic

### Architecture Summary

| Architecture | Models | Key Trait |
|-------------|--------|-----------|
| Flow Matching (Rectified Flow) | FLUX.1, FLUX.2 | Straight-line noise-to-image paths; fewer steps |
| MMDiT (Multimodal Diffusion Transformer) | SD 3.5, Qwen-Image | Superior prompt following; text rendering |
| Latent Diffusion (U-Net) | SDXL, SD 1.5 | Mature; largest ecosystem |
| Hybrid Autoregressive + Diffusion | GLM-Image | LLM understands prompt, diffusion decodes |
| MoE Autoregressive | HunyuanImage-3.0 | Massive capacity with sparse activation |
| DiT (Diffusion Transformer) | PixArt-Sigma, HunyuanDiT, Z-Image-Turbo | Efficient transformer-based diffusion |

---

## Video Generation Models (2025/2026)

### Consumer-Friendly (24 GB or less)

| Model | Params | Min VRAM | Resolution | Length | Speed (4090) |
|-------|--------|----------|-----------|--------|-------------|
| **LTX-2.3** | 22B | 6 GB (512p) / 24 GB (4K FP8) | up to 4K | 20s | **Faster than real-time** |
| **HunyuanVideo 1.5** | 8.3B | 8 GB (offload) / 14 GB | 1080p | 10s | ~75 sec |
| **Wan 2.1 (1.3B)** | 1.3B | **8 GB** | 480p | 5s | ~4 min |
| **CogVideoX-2B** | 2B | 12 GB | 768x1360 | 6s | ~90s |
| **AnimateDiff** | ~1B | 8 GB | 1024x1024 | 16 frames | seconds |

### Datacenter-Scale

| Model | Params | VRAM | Resolution | Notes |
|-------|--------|------|-----------|-------|
| Wan 2.1/2.2 14B | 14B | 24 GB (480p) / 65 GB (720p) | 720p | Most versatile |
| Mochi 1 | 10B | 60 GB | 480p | Novel AsymmDiT arch |
| MAGI-1 | 14.5B | multi-H100 | 720p | Autoregressive, infinite length |

### Detailed Model Specs

#### LTX Video (Lightricks)
- **LTX-2.3 (March 2026):** 22B params, rebuilt VAE, 4K at 50fps, up to 20 seconds
- First open-source model generating synchronized video AND audio in single pass
- Hierarchical temporal attention at three scales
- RTX 4090: 121 frames in 11 seconds. **5 seconds of video generated in 2 seconds**
- The fastest open-source video model by a significant margin

#### HunyuanVideo (Tencent)
- **v1.5 (Nov 2025):** 8.3B, "Dual-stream to Single-stream" hybrid, 1080p
- Selective and Sliding Tile Attention (SSTA) for 1.87x speedup
- ~13.6 GB with offloading for 720p/121 frames
- Can run on 8 GB VRAM with HunyuanVideoGP project
- Step-distilled on RTX 4090: ~75 seconds

#### Wan Video (Alibaba)
- **Wan 2.1 (Feb 2025):** DiT + Flow Matching, 1.3B and 14B variants
- **Wan 2.2 (Jul 2025):** First open-source video model with MoE architecture
- 1.3B: 480p, 5s, ~8.2 GB -- runs on nearly any modern GPU
- 14B at 480p: ~24 GB (RTX 4090 viable with offloading)
- 14B at 720p: 65-80 GB natively, needs datacenter GPU
- Most versatile: T2V, I2V, video editing, T2I, V2A

#### CogVideoX (Zhipu AI)
- 2B and 5B variants, 3D Causal VAE, Expert Transformer
- 2B: ~18 GB (A100), can run with 12 GB optimized
- 5B: ~24 GB recommended
- 768x1360, 10 seconds at 16fps

#### Mochi 1 (Genmo)
- 10B, Asymmetric Diffusion Transformer (AsymmDiT)
- Visual stream has 4x more params than text stream
- ~60 GB full precision, ~22 GB BFloat16
- Apache 2.0

#### MAGI-1 (Sand AI, Apr 2025)
- Autoregressive Diffusion Transformer
- Block-Causal Attention enables chunk-by-chunk generation
- Can theoretically generate infinite-length video
- 4.5B variant fits on RTX 4090

#### SkyReels V2 (Skywork AI, Apr 2025)
- Based on Wan 2.1 DiT + Flow Matching
- First open-source model using AutoRegressive Diffusion-Forcing for infinite-length generation
- Specialized in human-centric video: 33 expressions, 400+ movement combinations

### Key Takeaways
- **Fastest:** LTX-2.3 -- faster-than-real-time generation
- **Best quality per VRAM:** HunyuanVideo 1.5 -- competitive with 13B+ at 8.3B
- **Most versatile:** Wan 2.1/2.2 -- broadest capabilities
- **Most accessible:** Wan 2.1 1.3B at 8 GB, LTX at 6 GB minimum

---

## MLX Ecosystem (Apple Silicon)

### Image Generation on MLX

| Library | Models Supported | Stars |
|---------|-----------------|-------|
| **mflux** (filipstrand) | FLUX.1, FLUX.2 Klein, Z-Image, Qwen Image, FIBO, SeedVR2 | 1,954 |
| **ml-explore/mlx-examples** | SD 2.1, SDXL-Turbo, FLUX | 8,419 |
| **DiffusionKit** (argmaxinc) | SD3, FLUX.1 | 702 |
| **Draw Things** (app) | SD, SDXL, SD3, FLUX, Z-Image, Qwen, Wan, LTX | Fastest |

### Video Generation on MLX

| Library | Models | Notes |
|---------|--------|-------|
| **mlx-video** (Blaizzy) | LTX-2/2.3, Wan 2.1/2.2 | Primary MLX video lib |
| **ltx-2-mlx** (dgrauet) | LTX-2/2.3 | Pure MLX, LoRA finetuning |
| **MLX-GenAI** | LTX-2.3 22B | 8-bit ~14 GB, 4-bit ~10 GB |
| **Wan2.2-mlx** | Wan 2.2 | Pure MLX port |

### Apple Silicon Performance

**Image gen (FLUX.1 schnell, 1024x1024, 2 steps, q8):**

| Chip | Time | Notes |
|------|------|-------|
| M4 Max 128 GB | **~7 sec** | Interactive use viable |
| M4 Pro 64 GB | ~30-38 sec | |
| M2 Max 96 GB | ~26 sec | |
| M2 Pro 32 GB | ~54 sec | |

Apple Silicon is roughly **3-5x slower** than RTX 4090 for equivalent workloads.

### Memory Requirements on Unified Memory
- **16 GB**: FLUX.1 at 4-bit, LTX-2.3 at 4-bit (tight)
- **32 GB**: FLUX.1 full precision, LTX-2.3 int8
- **64 GB+**: Comfortable for everything including video
- **128 GB**: Multiple models, high-res video generation

### Key MLX Libraries

| Repo | Focus |
|------|-------|
| `filipstrand/mflux` | Primary MLX image gen (7 model families, LoRA, quantization) |
| `Blaizzy/mlx-video` | MLX video gen (LTX-2, Wan) |
| `argmaxinc/DiffusionKit` | Core ML + MLX diffusion |
| `dgrauet/ltx-2-mlx` | Pure MLX LTX-2/2.3 with finetuning |
| `appautomaton/MLX-GenAI` | Quantized LTX-2.3 on MLX |

### MLX-Specific Advantages for Diffusion
- **Unified memory / zero-copy**: No PCIe transfer overhead
- **Lazy evaluation**: Fused computation graph reduces kernel launch overhead
- **Native quantization**: Built-in 3/4/6/8-bit with efficient dequantization kernels
- **Draw Things Metal FlashAttention**: Custom Metal shaders, ~25% faster than mflux

---

## Integration Considerations

### Distributed Diffusion Inference Research

**DistriFusion (MIT HAN Lab, CVPR 2024):** Spatial patch parallelism, 6.1x speedup on 8 A100s.

**PipeFusion:** Pipeline parallelism for DiT, works on PCIe-linked GPUs (no NVLink needed).

**xDiT:** Unified engine supporting 30+ DiT models including FLUX, HunyuanVideo, Wan. Combines USP + PipeFusion + CFG parallelism.

### Why Distributed DiT is Tractable
- DiT transformer blocks use standard Linear layers (Q/K/V/O projections, FFN) -- same pattern already sharded for LLMs
- MLX has native `AllToShardedLinear` / `ShardedToAllLinear`
- FLUX.1: 57 blocks, hidden_dim=3072 → 114 all-reduces per denoising step
- LTX-2.3: 30 blocks → 60 all-reduces per step

### Three Parallelism Options
1. **Tensor Parallel** (best for high-bandwidth interconnects) - Shard transformer weights across nodes
2. **CFG Parallel** (free 2x) - Run positive/negative guidance branches on separate nodes
3. **PipeFusion** (best for slower interconnects) - Split transformer blocks across nodes, reuse stale activations

### Best Candidates for Integration
**Image gen:**
- FLUX.2 Klein 4B/9B -- fastest, Apache 2.0, mflux supports it
- Z-Image-Turbo -- great quality/VRAM ratio, mflux supports it
- FLUX.1 Dev/Schnell -- 12B, proven, massive ecosystem

**Video gen:**
- LTX-2.3 (22B) -- MLX ports exist, fastest model, joint audio+video
- Wan 2.1/2.2 -- MLX ports exist, most versatile
- HunyuanVideo 1.5 (8.3B) -- best quality-to-VRAM ratio

### The Big Opportunity
Video models like LTX-2.3 (22B) and Wan 14B are too large for a single consumer Mac but could be sharded across a heterogeneous cluster -- exactly what asymmetric tensor parallelism enables.
