package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/api"
	"github.com/dginf/dginf/services/coordinator/internal/app"
	"github.com/dginf/dginf/services/coordinator/internal/store"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	addr := envOrDefault("DGINF_COORDINATOR_ADDR", ":8080")
	relayURL := envOrDefault("DGINF_RELAY_URL", "quic://relay.dginf.local")
	contract := envOrDefault("DGINF_LEDGER_CONTRACT", "0x0000000000000000000000000000000000000000")
	signerKey := mustLoadSigner(envOrDefault("DGINF_LEDGER_SIGNER_KEY", strings.Repeat("1", 64)))

	service := app.NewServiceWithSigner(store.NewMemory(), relayURL, time.Now, signerKey, 8453, contract)
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewServer(service).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("coordinator listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func mustLoadSigner(hexKey string) *ecdsa.PrivateKey {
	decoded, err := hex.DecodeString(hexKey)
	if err != nil {
		log.Fatalf("decode signer key: %v", err)
	}
	key, err := crypto.ToECDSA(decoded)
	if err != nil {
		log.Fatalf("load signer key: %v", err)
	}
	return key
}
