// Package billing provides unified payment processing for the DGInf coordinator.
//
// Three payment methods are supported as parallel work streams:
//   - Stripe: Fiat checkout sessions with webhook confirmation
//   - EVM:    On-chain deposits/withdrawals on Ethereum-compatible chains (Tempo, Ethereum, Base)
//   - Solana: On-chain deposits/withdrawals using SPL tokens (USDC)
//
// All payment methods ultimately credit the same internal micro-USD ledger.
// A referral system allows accounts to earn a share of platform fees for
// users they refer.
package billing

import (
	"fmt"
	"log/slog"

	"github.com/dginf/coordinator/internal/payments"
	"github.com/dginf/coordinator/internal/store"
)

// PaymentMethod identifies the payment rail used for a transaction.
type PaymentMethod string

const (
	MethodStripe PaymentMethod = "stripe"
	MethodEVM    PaymentMethod = "evm"
	MethodSolana PaymentMethod = "solana"
)

// Chain identifies the specific blockchain network.
type Chain string

const (
	ChainEthereum Chain = "ethereum"
	ChainTempo    Chain = "tempo"
	ChainBase     Chain = "base"
	ChainSolana   Chain = "solana"
)

// Config holds billing service configuration, typically from environment variables.
type Config struct {
	// Stripe
	StripeSecretKey    string
	StripeWebhookSecret string
	StripeSuccessURL   string // redirect URL after successful payment
	StripeCancelURL    string // redirect URL after cancelled payment

	// EVM chains
	EVMChains []EVMChainConfig

	// Solana
	SolanaRPCURL        string
	SolanaDepositAddress string
	SolanaUSDCMint      string
	SolanaPrivateKey    string // hot wallet for withdrawals (base58)

	// Referral
	ReferralSharePercent int64 // percentage of platform fee going to referrer (default 20)
}

// EVMChainConfig configures a single EVM-compatible chain.
type EVMChainConfig struct {
	Chain          Chain
	RPCURL         string
	DepositAddress string
	USDCContract   string // ERC-20 contract address for the stablecoin
	PrivateKey     string // hot wallet for withdrawals (hex, no 0x prefix)
	ChainID        int64
}

// DepositAddresses returns the deposit addresses for all configured chains.
type DepositAddresses struct {
	EVM    map[Chain]string `json:"evm,omitempty"`
	Solana string           `json:"solana,omitempty"`
}

// Service is the unified billing orchestrator. It delegates to chain-specific
// processors and manages the referral reward flow.
type Service struct {
	store    store.Store
	ledger   *payments.Ledger
	logger   *slog.Logger
	config   Config

	stripe   *StripeProcessor
	evm      map[Chain]*EVMProcessor
	solana   *SolanaProcessor
	referral *ReferralService

	// processedTxHashes prevents double-crediting the same on-chain tx.
	// In production, this should be backed by the database.
	processedTxHashes map[string]bool
}

// NewService creates a new billing service from the given configuration.
func NewService(st store.Store, ledger *payments.Ledger, logger *slog.Logger, cfg Config) *Service {
	if cfg.ReferralSharePercent == 0 {
		cfg.ReferralSharePercent = 20 // default: referrer gets 20% of platform fee
	}

	svc := &Service{
		store:             st,
		ledger:            ledger,
		logger:            logger,
		config:            cfg,
		evm:               make(map[Chain]*EVMProcessor),
		referral:          NewReferralService(st, ledger, logger, cfg.ReferralSharePercent),
		processedTxHashes: make(map[string]bool),
	}

	// Initialize Stripe if configured
	if cfg.StripeSecretKey != "" {
		svc.stripe = NewStripeProcessor(cfg.StripeSecretKey, cfg.StripeWebhookSecret,
			cfg.StripeSuccessURL, cfg.StripeCancelURL, logger)
		logger.Info("billing: Stripe processor enabled")
	}

	// Initialize EVM chain processors
	for _, chainCfg := range cfg.EVMChains {
		if chainCfg.RPCURL != "" {
			svc.evm[chainCfg.Chain] = NewEVMProcessor(chainCfg, logger)
			logger.Info("billing: EVM processor enabled", "chain", chainCfg.Chain)
		}
	}

	// Initialize Solana processor
	if cfg.SolanaRPCURL != "" {
		svc.solana = NewSolanaProcessor(cfg.SolanaRPCURL, cfg.SolanaDepositAddress,
			cfg.SolanaUSDCMint, cfg.SolanaPrivateKey, logger)
		logger.Info("billing: Solana processor enabled")
	}

	return svc
}

// Stripe returns the Stripe processor, or nil if not configured.
func (s *Service) Stripe() *StripeProcessor { return s.stripe }

// EVM returns the EVM processor for a specific chain, or nil if not configured.
func (s *Service) EVM(chain Chain) *EVMProcessor { return s.evm[chain] }

// Solana returns the Solana processor, or nil if not configured.
func (s *Service) Solana() *SolanaProcessor { return s.solana }

// Referral returns the referral service.
func (s *Service) Referral() *ReferralService { return s.referral }

// Store returns the underlying store for direct access.
func (s *Service) Store() store.Store { return s.store }

// Ledger returns the underlying ledger for direct access.
func (s *Service) Ledger() *payments.Ledger { return s.ledger }

// DepositAddresses returns all configured deposit addresses.
func (s *Service) DepositAddresses() DepositAddresses {
	addrs := DepositAddresses{
		EVM: make(map[Chain]string),
	}
	for chain, proc := range s.evm {
		addrs.EVM[chain] = proc.DepositAddress()
	}
	if s.solana != nil {
		addrs.Solana = s.solana.DepositAddress()
	}
	return addrs
}

// SupportedMethods returns which payment methods are configured and available.
func (s *Service) SupportedMethods() []PaymentMethodInfo {
	var methods []PaymentMethodInfo

	if s.stripe != nil {
		methods = append(methods, PaymentMethodInfo{
			Method:      MethodStripe,
			DisplayName: "Credit/Debit Card (Stripe)",
			Currencies:  []string{"USD"},
		})
	}

	for chain, proc := range s.evm {
		methods = append(methods, PaymentMethodInfo{
			Method:         MethodEVM,
			Chain:          chain,
			DisplayName:    fmt.Sprintf("Stablecoin on %s", chain),
			DepositAddress: proc.DepositAddress(),
			Currencies:     []string{"USDC", "pathUSD"},
		})
	}

	if s.solana != nil {
		methods = append(methods, PaymentMethodInfo{
			Method:         MethodSolana,
			Chain:          ChainSolana,
			DisplayName:    "USDC on Solana",
			DepositAddress: s.solana.DepositAddress(),
			Currencies:     []string{"USDC"},
		})
	}

	return methods
}

// CheckProcessedTx returns true if this tx hash has already been credited.
func (s *Service) CheckProcessedTx(txHash string) bool {
	return s.processedTxHashes[txHash]
}

// MarkProcessedTx marks a tx hash as processed to prevent double-crediting.
func (s *Service) MarkProcessedTx(txHash string) {
	s.processedTxHashes[txHash] = true
}

// CreditDeposit credits a consumer's balance after a verified deposit and
// handles referral rewards if applicable.
func (s *Service) CreditDeposit(accountID string, amountMicroUSD int64, entryType store.LedgerEntryType, reference string) error {
	return s.store.Credit(accountID, amountMicroUSD, entryType, reference)
}

// PaymentMethodInfo describes a supported payment method for the API.
type PaymentMethodInfo struct {
	Method         PaymentMethod `json:"method"`
	Chain          Chain         `json:"chain,omitempty"`
	DisplayName    string        `json:"display_name"`
	DepositAddress string        `json:"deposit_address,omitempty"`
	Currencies     []string      `json:"currencies"`
}
