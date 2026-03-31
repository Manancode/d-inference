// Command coordinator runs the DGInf coordinator control plane.
//
// The coordinator is the central routing and trust layer in the DGInf network.
// It accepts provider WebSocket connections, verifies their Secure Enclave
// attestations, and routes OpenAI-compatible HTTP requests from consumers
// to appropriate providers based on model availability and trust level.
//
// Deployment: The coordinator runs in a GCP Confidential VM (AMD SEV-SNP)
// with hardware-encrypted memory. Consumer traffic arrives over HTTPS/TLS.
// The coordinator can read requests for routing purposes but never logs
// prompt content.
//
// Configuration (environment variables):
//   DGINF_PORT         - HTTP listen port (default: "8080")
//   DGINF_ADMIN_KEY    - Pre-seeded API key for bootstrapping
//   DGINF_DATABASE_URL - PostgreSQL connection string (omit for in-memory store)
//
// Graceful shutdown: The coordinator handles SIGINT/SIGTERM, stops the
// eviction loop, and drains active connections with a 15-second deadline.
package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strconv"

	"github.com/dginf/coordinator/internal/api"
	"github.com/dginf/coordinator/internal/attestation"
	"github.com/dginf/coordinator/internal/billing"
	"github.com/dginf/coordinator/internal/mdm"
	"github.com/dginf/coordinator/internal/payments"
	"github.com/dginf/coordinator/internal/registry"
	"github.com/dginf/coordinator/internal/store"
)

func main() {
	// Structured logging.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Configuration from environment.
	port := envOr("DGINF_PORT", "8080")
	adminKey := os.Getenv("DGINF_ADMIN_KEY")

	if adminKey == "" {
		logger.Warn("DGINF_ADMIN_KEY is not set — no pre-seeded API key available")
	}

	// Create core components.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var st store.Store
	if dbURL := os.Getenv("DGINF_DATABASE_URL"); dbURL != "" {
		pgStore, err := store.NewPostgres(ctx, dbURL)
		if err != nil {
			logger.Error("failed to connect to PostgreSQL", "error", err)
			os.Exit(1)
		}
		defer pgStore.Close()
		st = pgStore
		logger.Info("using PostgreSQL store")

		// If an admin key is set, seed it in the database.
		if adminKey != "" {
			if err := pgStore.SeedKey(adminKey); err != nil {
				logger.Warn("failed to seed admin key (may already exist)", "error", err)
			}
		}
	} else {
		st = store.NewMemory(adminKey)
		logger.Info("using in-memory store")
	}

	reg := registry.New(logger)

	// Set minimum trust level for routing. Default: hardware (production).
	// Set DGINF_MIN_TRUST=none or DGINF_MIN_TRUST=self_signed for testing.
	if minTrust := os.Getenv("DGINF_MIN_TRUST"); minTrust != "" {
		reg.MinTrustLevel = registry.TrustLevel(minTrust)
		logger.Info("minimum trust level override", "level", minTrust)
	}

	srv := api.NewServer(reg, st, logger)

	// Configure billing service (Stripe + EVM + Solana + Referrals).
	billingCfg := billing.Config{
		StripeSecretKey:     os.Getenv("DGINF_STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("DGINF_STRIPE_WEBHOOK_SECRET"),
		StripeSuccessURL:    envOr("DGINF_STRIPE_SUCCESS_URL", "https://inference-test.openinnovation.dev/billing/success"),
		StripeCancelURL:     envOr("DGINF_STRIPE_CANCEL_URL", "https://inference-test.openinnovation.dev/billing/cancel"),
		SolanaRPCURL:        os.Getenv("DGINF_SOLANA_RPC_URL"),
		SolanaDepositAddress: os.Getenv("DGINF_SOLANA_DEPOSIT_ADDRESS"),
		SolanaUSDCMint:      envOr("DGINF_SOLANA_USDC_MINT", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"), // mainnet USDC
		SolanaPrivateKey:    os.Getenv("DGINF_SOLANA_PRIVATE_KEY"),
	}

	// Parse referral share percentage
	if refShareStr := os.Getenv("DGINF_REFERRAL_SHARE_PCT"); refShareStr != "" {
		if v, err := strconv.ParseInt(refShareStr, 10, 64); err == nil {
			billingCfg.ReferralSharePercent = v
		}
	}

	// Configure EVM chains from environment
	// Tempo chain (primary — pathUSD stablecoin)
	if tempoRPC := os.Getenv("DGINF_TEMPO_RPC_URL"); tempoRPC != "" {
		billingCfg.EVMChains = append(billingCfg.EVMChains, billing.EVMChainConfig{
			Chain:          billing.ChainTempo,
			RPCURL:         tempoRPC,
			DepositAddress: os.Getenv("DGINF_TEMPO_DEPOSIT_ADDRESS"),
			USDCContract:   os.Getenv("DGINF_TEMPO_PATHUSD_CONTRACT"),
			PrivateKey:     os.Getenv("DGINF_TEMPO_PRIVATE_KEY"),
		})
	}

	// Ethereum chain (USDC)
	if ethRPC := os.Getenv("DGINF_ETH_RPC_URL"); ethRPC != "" {
		billingCfg.EVMChains = append(billingCfg.EVMChains, billing.EVMChainConfig{
			Chain:          billing.ChainEthereum,
			RPCURL:         ethRPC,
			DepositAddress: os.Getenv("DGINF_ETH_DEPOSIT_ADDRESS"),
			USDCContract:   envOr("DGINF_ETH_USDC_CONTRACT", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // mainnet USDC
			PrivateKey:     os.Getenv("DGINF_ETH_PRIVATE_KEY"),
			ChainID:        1,
		})
	}

	// Base chain (USDC)
	if baseRPC := os.Getenv("DGINF_BASE_RPC_URL"); baseRPC != "" {
		billingCfg.EVMChains = append(billingCfg.EVMChains, billing.EVMChainConfig{
			Chain:          billing.ChainBase,
			RPCURL:         baseRPC,
			DepositAddress: os.Getenv("DGINF_BASE_DEPOSIT_ADDRESS"),
			USDCContract:   envOr("DGINF_BASE_USDC_CONTRACT", "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"), // Base mainnet USDC
			PrivateKey:     os.Getenv("DGINF_BASE_PRIVATE_KEY"),
			ChainID:        8453,
		})
	}

	ledger := payments.NewLedger(st)
	billingSvc := billing.NewService(st, ledger, logger, billingCfg)
	srv.SetBilling(billingSvc)

	// Log which billing methods are active
	methods := billingSvc.SupportedMethods()
	if len(methods) > 0 {
		var names []string
		for _, m := range methods {
			names = append(names, string(m.Method))
		}
		logger.Info("billing enabled", "methods", names, "referral_share_pct", billingCfg.ReferralSharePercent)
	}

	// Configure MDM client for provider security verification.
	// When set, the coordinator independently verifies SIP/SecureBoot via MicroMDM
	// rather than trusting the provider's self-reported attestation.
	if mdmURL := os.Getenv("DGINF_MDM_URL"); mdmURL != "" {
		mdmKey := os.Getenv("DGINF_MDM_API_KEY")
		if mdmKey == "" {
			mdmKey = "dginf-micromdm-api" // default
		}
		mdmClient := mdm.NewClient(mdmURL, mdmKey, logger)

		// Register callback for late-arriving MDA certs — stores them
		// on the provider so users can verify via the attestation API.
		mdmClient.SetOnMDA(func(udid string, certChain [][]byte) {
			// Find the provider with this UDID and store the cert chain
			reg.ForEachProvider(func(p *registry.Provider) {
				if p.AttestationResult == nil {
					return
				}
				// Match by checking if this provider's MDM UDID matches
				// (UDID is set during MDM verification)
				mdaResult, err := attestation.VerifyMDADeviceAttestation(certChain)
				if err != nil {
					logger.Error("late MDA cert parse error", "udid", udid, "error", err)
					return
				}
				if mdaResult.Valid && (mdaResult.DeviceSerial == p.AttestationResult.SerialNumber) {
					p.MDAVerified = true
					p.MDACertChain = certChain
					p.MDAResult = mdaResult
					logger.Info("late MDA cert stored on provider",
						"provider_id", p.ID,
						"serial", mdaResult.DeviceSerial,
						"udid", mdaResult.DeviceUDID,
						"os_version", mdaResult.OSVersion,
					)
				}
			})
		})

		srv.SetMDMClient(mdmClient)
		logger.Info("MDM verification enabled", "url", mdmURL)
	}

	// Configure step-ca root CA for ACME client cert verification.
	// When providers present a TLS client cert issued by step-ca via
	// device-attest-01, the coordinator verifies the chain and grants
	// hardware trust (Apple-attested SE key binding).
	if stepCARoot := os.Getenv("DGINF_STEP_CA_ROOT"); stepCARoot != "" {
		rootPEM, err := os.ReadFile(stepCARoot)
		if err != nil {
			logger.Error("failed to read step-ca root CA", "path", stepCARoot, "error", err)
		} else {
			block, _ := pem.Decode(rootPEM)
			if block != nil {
				rootCert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					logger.Error("failed to parse step-ca root CA", "error", err)
				} else {
					// Try to load intermediate too
					var intCert *x509.Certificate
					stepCAInt := os.Getenv("DGINF_STEP_CA_INTERMEDIATE")
					if stepCAInt != "" {
						intPEM, err := os.ReadFile(stepCAInt)
						if err == nil {
							intBlock, _ := pem.Decode(intPEM)
							if intBlock != nil {
								intCert, _ = x509.ParseCertificate(intBlock.Bytes)
							}
						}
					}
					srv.SetStepCACerts(rootCert, intCert)
					logger.Info("step-ca ACME client cert verification enabled", "root", stepCARoot)
				}
			}
		}
	}

	// Start background eviction of stale providers.
	reg.StartEvictionLoop(ctx, 90*time.Second)

	// HTTP server with graceful shutdown.
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // SSE streaming requires no write timeout
		IdleTimeout:  120 * time.Second,
	}

	// Start listening.
	go func() {
		logger.Info("coordinator starting", "port", port, "admin_key_set", adminKey != "")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("shutting down", "signal", sig.String())

	// Graceful shutdown with a deadline.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	cancel() // Stop the eviction loop.

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("coordinator stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
