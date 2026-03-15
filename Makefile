GO_MODULES := ./services/coordinator ./services/providerd
GO_CACHE_DIR := $(abspath $(CURDIR)/.local/go-build-cache)
GO_MOD_CACHE := $(abspath $(CURDIR)/.local/gomodcache)
UV_CACHE_DIR := $(abspath $(CURDIR)/.local/uv-cache)
HOME_DIR := $(abspath $(CURDIR)/.home)
CLANG_CACHE_DIR := $(abspath $(CURDIR)/.cache/clang)
SWIFTPM_CACHE_DIR := $(abspath $(CURDIR)/.cache/swiftpm)

.PHONY: bootstrap test-all test-go test-swift test-py test-contracts test-e2e fmt-go clean-generated

bootstrap:
	@mkdir -p $(GO_CACHE_DIR) $(GO_MOD_CACHE) $(UV_CACHE_DIR) $(HOME_DIR) $(CLANG_CACHE_DIR) $(SWIFTPM_CACHE_DIR)

test-all: bootstrap test-go test-swift test-py test-contracts test-e2e

test-go:
	@for module in $(GO_MODULES); do \
		echo "==> $$module"; \
		(cd $$module && GOCACHE=$(GO_CACHE_DIR) GOMODCACHE=$(GO_MOD_CACHE) go test ./...); \
	done

test-swift:
	cd apps/provider-mac && HOME=$(HOME_DIR) CLANG_MODULE_CACHE_PATH=$(CLANG_CACHE_DIR) SWIFTPM_MODULECACHE_OVERRIDE=$(SWIFTPM_CACHE_DIR) SWIFTPM_DISABLE_SANDBOX=1 swift test --disable-sandbox

test-py:
	HOME=$(HOME_DIR) UV_CACHE_DIR=$(UV_CACHE_DIR) uv run pytest packages/runtime-mlx/tests packages/sdk-py/tests

test-e2e:
	HOME=$(HOME_DIR) UV_CACHE_DIR=$(UV_CACHE_DIR) uv run pytest tests/e2e

test-contracts:
	cd packages/contracts && HOME=$(HOME_DIR) forge test

fmt-go:
	@for module in $(GO_MODULES); do \
		echo "==> $$module"; \
		(cd $$module && gofmt -w $$(find . -name '*.go' -print)); \
	done

clean-generated:
	-rm -rf .cache .home .venv .pytest_cache .local/go-build-cache .local/gomodcache .local/uv-cache packages/contracts/cache packages/contracts/out uv.lock
