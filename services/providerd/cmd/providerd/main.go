package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dginf/dginf/services/providerd/internal/api"
	"github.com/dginf/dginf/services/providerd/internal/app"
	coordclient "github.com/dginf/dginf/services/providerd/internal/coordinator"
	"github.com/dginf/dginf/services/providerd/internal/domain"
	"github.com/dginf/dginf/services/providerd/internal/identity"
	"github.com/dginf/dginf/services/providerd/internal/posture"
	"github.com/dginf/dginf/services/providerd/internal/runtime"
	"github.com/dginf/dginf/services/providerd/internal/store"
)

func main() {
	addr := envOrDefault("DGINF_PROVIDERD_ADDR", "127.0.0.1:8787")
	publicURL := envOrDefault("DGINF_PROVIDERD_PUBLIC_URL", "http://"+addr)
	model := envOrDefault("DGINF_PROVIDERD_MODEL", "qwen3.5-35b-a3b")
	runtimeURL := envOrDefault("DGINF_RUNTIME_URL", "http://127.0.0.1:8089")
	coordinatorURL := os.Getenv("DGINF_COORDINATOR_URL")
	keyTool := os.Getenv("DGINF_PROVIDER_KEY_TOOL")

	var signer identity.Signer
	var err error
	if keyTool != "" {
		signer = identity.NewCommandSigner(keyTool, envOrDefault("DGINF_PROVIDER_KEY_TAG", "com.dginf.provider.signing"))
		if _, err := signer.PublicKey(); err != nil {
			log.Printf("providerd secure-enclave signer unavailable, falling back to software signer: %v", err)
			signer = nil
		}
	}
	if signer == nil {
		signer, err = identity.NewSoftwareSigner()
		if err != nil {
			log.Fatal(err)
		}
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	var coordinator *coordclient.Client
	if coordinatorURL != "" {
		coordinator = coordclient.NewClient(coordinatorURL)
	}
	service := app.NewService(store.NewMemory(), signer, sessionKeys, coordinator, runtime.NewClient(runtimeURL), posture.NewCollector(time.Now), time.Now)
	if _, err := service.Bootstrap(domain.NodeConfig{
		NodeID:          envOrDefault("DGINF_PROVIDERD_NODE_ID", "dev-node"),
		ProviderWallet:  envOrDefault("DGINF_PROVIDERD_WALLET", "0xprovider"),
		PublicURL:       publicURL,
		SelectedModel:   model,
		MemoryGB:        envOrDefaultInt("DGINF_PROVIDERD_MEMORY_GB", 64),
		HardwareProfile: envOrDefault("DGINF_PROVIDERD_HARDWARE_PROFILE", "M3 Max 64GB"),
		MinJobUSDC:      envOrDefaultInt64("DGINF_PROVIDERD_MIN_JOB_USDC", 100),
		Input1MUSDC:     envOrDefaultInt64("DGINF_PROVIDERD_INPUT_1M_USDC", 10_000),
		Output1MUSDC:    envOrDefaultInt64("DGINF_PROVIDERD_OUTPUT_1M_USDC", 20_000),
	}); err != nil {
		log.Fatal(err)
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			if err := service.RegisterWithCoordinator(context.Background()); err != nil {
				log.Printf("providerd register warning: %v", err)
			}
			if err := service.SendHeartbeat(context.Background()); err != nil {
				log.Printf("providerd heartbeat warning: %v", err)
			}
			<-ticker.C
		}
	}()
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewServer(service).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("providerd listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var parsed int
		if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func envOrDefaultInt64(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		var parsed int64
		if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}
