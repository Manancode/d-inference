# Upstream Runtime References

This worktree keeps shallow local clones of the upstream projects we expect to
borrow from or benchmark against under `.local/upstream/`.

## Pinned Revisions

| Project | URL | Commit | Intended use |
| --- | --- | --- | --- |
| MLX | https://github.com/ml-explore/mlx | `5d1700493a2d199e4f3aac7c5a8df4925c05039f` | Core Apple Silicon tensor/runtime substrate |
| mlx-lm | https://github.com/ml-explore/mlx-lm | `332d94ca6f7e12bd4350ebe5a21f3188c42f19f0` | Model loading, generation, tokenizer, quantized model handling |
| llama.cpp | https://github.com/ggml-org/llama.cpp | `d23355afc319f598d0e588a2d16a4da82e14ff41` | Fallback backend, API patterns, local OpenAI-compatible serving |
| vLLM | https://github.com/vllm-project/vllm | `458c1a4b2d21965ecd41b76ec0506ffe5ed8c8a1` | Scheduler and batching reference for later phases |
| vLLM Metal | https://github.com/vllm-project/vllm-metal | `0e7a105e50088fc97aff6035d7975a38c38d95ae` | Apple Silicon vLLM compatibility reference |
| Exo | https://github.com/exo-explore/exo | `7ed46395405cd4e848982c228f79e687ec1e1d89` | Multi-node and cluster design reference |
| vllm-mlx | https://github.com/waybarrios/vllm-mlx | `22dcbf87f31daf1182c5092c42c476f4e8a24110` | Unofficial Apple Silicon batching reference |

## Current Borrowing Stance

- Use `MLX` and `mlx-lm` directly for the first production runtime adapter.
- Treat `llama.cpp` as a planned fallback runtime once the core MLX path is stable.
- Treat `vLLM`, `vllm-metal`, `vllm-mlx`, and `Exo` as design/reference inputs only for now.
- Do not fork or patch upstream code in this worktree until a concrete adapter gap is identified.
