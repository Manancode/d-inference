#!/bin/bash
set -euo pipefail

# DGInf Provider Installer
# Usage: curl -fsSL https://inference-test.openinnovation.dev/install.sh | bash
#
# Installs:
#   - dginf-provider (Rust binary, Apple Silicon)
#   - dginf-enclave (Swift CLI, Secure Enclave attestation)
#   - vllm-mlx + mlx + mlx-lm (via pip, matched to your Python version)
#
# Requires: macOS with Apple Silicon, Python 3.10+

BASE_URL="https://inference-test.openinnovation.dev"
DGINF_DIR="$HOME/.dginf"
BIN_DIR="$DGINF_DIR/bin"
COORDINATOR_WS="wss://inference-test.openinnovation.dev/ws/provider"

echo "╔══════════════════════════════════════════════╗"
echo "║  DGInf — Decentralized Private Inference     ║"
echo "║  Provider Installer                          ║"
echo "╚══════════════════════════════════════════════╝"
echo ""

# Check macOS + Apple Silicon
if [ "$(uname)" != "Darwin" ]; then
    echo "Error: DGInf provider requires macOS with Apple Silicon."
    exit 1
fi

ARCH=$(uname -m)
if [ "$ARCH" != "arm64" ]; then
    echo "Error: DGInf provider requires Apple Silicon (arm64). Detected: $ARCH"
    exit 1
fi

echo "→ Detected: macOS $(sw_vers -productVersion) on $ARCH"

# Detect hardware
CHIP=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "Unknown")
MEM=$(sysctl -n hw.memsize 2>/dev/null | awk '{printf "%.0f", $1/1073741824}')
echo "→ Hardware: $CHIP, ${MEM}GB RAM"

# Check Python
if ! command -v python3 >/dev/null 2>&1; then
    echo ""
    echo "Error: python3 not found."
    echo "Install Python 3.10+ from https://python.org or: brew install python@3.12"
    exit 1
fi
PYVER=$(python3 -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
echo "→ Python: $PYVER"
echo ""

# Download binaries
echo "→ Downloading dginf-provider..."
mkdir -p "$BIN_DIR"
curl -fSL "$BASE_URL/dl/dginf-provider" -o "$BIN_DIR/dginf-provider"
chmod +x "$BIN_DIR/dginf-provider"

echo "→ Downloading dginf-enclave..."
curl -fSL "$BASE_URL/dl/dginf-enclave" -o "$BIN_DIR/dginf-enclave"
chmod +x "$BIN_DIR/dginf-enclave"

# Add to PATH if needed
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    SHELL_RC="$HOME/.zshrc"
    if [ -f "$HOME/.bashrc" ] && [ ! -f "$HOME/.zshrc" ]; then
        SHELL_RC="$HOME/.bashrc"
    fi
    echo "" >> "$SHELL_RC"
    echo "# DGInf provider" >> "$SHELL_RC"
    echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$SHELL_RC"
    export PATH="$BIN_DIR:$PATH"
    echo "→ Added $BIN_DIR to PATH"
fi

# Install inference engine
echo ""
echo "→ Installing inference engine..."
if python3 -c "import vllm_mlx" 2>/dev/null; then
    VVER=$(python3 -c "import vllm_mlx; print(vllm_mlx.__version__)" 2>/dev/null || echo "?")
    echo "  ✓ vllm-mlx $VVER already installed"
else
    echo "  Installing vllm-mlx (this may take a minute)..."
    pip3 install vllm-mlx --break-system-packages 2>/dev/null \
        || pip3 install vllm-mlx 2>/dev/null \
        || { echo "  ⚠ pip install failed. Try: pip3 install vllm-mlx"; }
    if python3 -c "import vllm_mlx" 2>/dev/null; then
        VVER=$(python3 -c "import vllm_mlx; print(vllm_mlx.__version__)" 2>/dev/null)
        echo "  ✓ vllm-mlx $VVER installed"
    fi
fi

# Setup Secure Enclave identity
echo ""
echo "→ Setting up Secure Enclave identity..."
if "$BIN_DIR/dginf-enclave" info >/dev/null 2>&1; then
    echo "  ✓ Secure Enclave ready"
else
    echo "  ⚠ Secure Enclave not available (attestation will be skipped)"
fi

# Suggest models
echo ""
echo "→ Recommended models for ${MEM}GB RAM:"
if [ "$MEM" -ge 64 ]; then
    echo "  • mlx-community/Qwen3.5-32B-Instruct-4bit"
    echo "  • mlx-community/Qwen3.5-14B-Instruct-4bit"
elif [ "$MEM" -ge 32 ]; then
    echo "  • mlx-community/Qwen3.5-14B-Instruct-4bit"
    echo "  • mlx-community/Qwen3.5-9B-MLX-4bit"
elif [ "$MEM" -ge 16 ]; then
    echo "  • mlx-community/Qwen3.5-9B-MLX-4bit"
    echo "  • mlx-community/Qwen2.5-3B-Instruct-4bit"
else
    echo "  • mlx-community/Qwen2.5-0.5B-Instruct-4bit"
fi

echo ""
echo "════════════════════════════════════════════════"
echo "  Installation complete!"
echo ""
echo "  Start serving:"
echo "    dginf-provider serve --coordinator $COORDINATOR_WS"
echo ""
echo "  With a specific model:"
echo "    dginf-provider serve --coordinator $COORDINATOR_WS \\"
echo "      --model mlx-community/Qwen3.5-9B-MLX-4bit"
echo ""
echo "  Run diagnostics:"
echo "    dginf-provider doctor"
echo "════════════════════════════════════════════════"
