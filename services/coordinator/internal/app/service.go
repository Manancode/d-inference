package app

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/auth"
	"github.com/dginf/dginf/services/coordinator/internal/domain"
	"github.com/dginf/dginf/services/coordinator/internal/providerclient"
	"github.com/dginf/dginf/services/coordinator/internal/store"
	"github.com/google/uuid"
)

type Service struct {
	store     *store.Memory
	auth      *auth.Service
	now       func() time.Time
	relayURL  string
	catalog   []domain.CatalogEntry
	signerKey *ecdsa.PrivateKey
	chainID   uint64
	contract  string
}

func NewService(memory *store.Memory, relayURL string, now func() time.Time) *Service {
	return NewServiceWithSigner(memory, relayURL, now, nil, 8453, "0x0000000000000000000000000000000000000000")
}

func NewServiceWithSigner(memory *store.Memory, relayURL string, now func() time.Time, signerKey *ecdsa.PrivateKey, chainID uint64, contract string) *Service {
	if memory == nil {
		memory = store.NewMemory()
	}
	if now == nil {
		now = time.Now
	}
	if relayURL == "" {
		relayURL = "quic://relay.dginf.local"
	}
	service := &Service{
		store:     memory,
		auth:      auth.NewService(memory, now),
		now:       now,
		relayURL:  relayURL,
		signerKey: signerKey,
		chainID:   chainID,
		contract:  contract,
		catalog: []domain.CatalogEntry{
			{ModelID: "qwen3.5-4b-mlx-4bit", MinimumMemoryGB: 16, Description: "Local MLX smoke model"},
			{ModelID: "qwen3.5-9b", MinimumMemoryGB: 16, Description: "Smoke and dev model"},
			{ModelID: "qwen3.5-35b-a3b", MinimumMemoryGB: 64, Description: "Default 64GB tier model"},
			{ModelID: "qwen3.5-122b-a10b", MinimumMemoryGB: 128, Description: "Premium 128GB tier model"},
		},
	}
	return service
}

func (s *Service) IssueChallenge(req domain.AuthChallengeRequest) (domain.AuthChallengeResponse, error) {
	return s.auth.IssueChallenge(req.Wallet, req.ChainID)
}

func (s *Service) VerifyChallenge(req domain.AuthVerifyRequest) (domain.AuthVerifyResponse, error) {
	return s.auth.VerifyChallenge(req.Wallet, req.Message, req.Signature)
}

func (s *Service) RegisterProvider(req domain.ProviderRegistration) error {
	if req.NodeID == "" || req.ProviderWallet == "" || req.SelectedModelID == "" {
		return errors.New("nodeId, providerWallet, and selectedModelId are required")
	}
	if !s.catalogSupports(req.SelectedModelID, req.MemoryGB) {
		return domain.ErrModelUnavailable
	}
	s.store.UpsertProvider(domain.Provider{
		ProviderRegistration: normalizeRegistration(req),
		Status:               domain.ProviderStatusHealthy,
		Allowlisted:          true,
		LastHeartbeatAt:      s.now().UTC(),
	})
	return nil
}

func (s *Service) Heartbeat(req domain.ProviderHeartbeat) error {
	provider, ok := s.store.GetProvider(req.NodeID)
	if !ok {
		return domain.ErrNoCapacity
	}
	provider.Status = req.Status
	if req.SelectedModelID != "" {
		provider.SelectedModelID = req.SelectedModelID
	}
	provider.Posture = req.Posture
	provider.LastHeartbeatAt = s.now().UTC()
	s.store.UpsertProvider(provider)
	return nil
}

func (s *Service) Models() []domain.CatalogEntry {
	return slices.Clone(s.catalog)
}

func (s *Service) Providers() []domain.Provider {
	return s.store.ListProviders()
}

func (s *Service) SeedBalance(wallet string, availableUSDC int64) {
	s.store.PutBalance(domain.WalletBalance{
		Wallet:        wallet,
		AvailableUSDC: availableUSDC,
	})
}

func (s *Service) SeedWalletBalance(req domain.SeedBalanceRequest) domain.WalletBalance {
	s.store.PutBalance(domain.WalletBalance{
		Wallet:           req.Wallet,
		AvailableUSDC:    req.AvailableUSDC,
		WithdrawableUSDC: req.WithdrawableUSDC,
	})
	return s.store.GetBalance(req.Wallet)
}

func (s *Service) Balance(wallet string) domain.WalletBalance {
	s.expireQuotes()
	return s.store.GetBalance(wallet)
}

func (s *Service) QuoteJob(req domain.JobQuoteRequest) (domain.JobQuote, error) {
	s.expireQuotes()
	if req.ConsumerWallet == "" || req.ModelID == "" {
		return domain.JobQuote{}, domain.ErrModelUnavailable
	}
	entry, ok := s.catalogEntry(req.ModelID)
	if !ok {
		return domain.JobQuote{}, domain.ErrModelUnavailable
	}
	candidates := make([]domain.Provider, 0)
	for _, provider := range s.store.ListProviders() {
		if !provider.Allowlisted {
			continue
		}
		if provider.SelectedModelID != req.ModelID {
			continue
		}
		if provider.MemoryGB < entry.MinimumMemoryGB {
			continue
		}
		if provider.Status != domain.ProviderStatusHealthy {
			continue
		}
		candidates = append(candidates, provider)
	}
	if len(candidates) == 0 {
		return domain.JobQuote{}, domain.ErrNoCapacity
	}
	slices.SortFunc(candidates, func(left, right domain.Provider) int {
		return compareCosts(quoteCost(left.RateCard, req), quoteCost(right.RateCard, req))
	})
	selected := candidates[0]
	reservation := quoteCost(selected.RateCard, req)
	balance := s.store.GetBalance(req.ConsumerWallet)
	if balance.AvailableUSDC < reservation {
		return domain.JobQuote{}, domain.ErrInsufficientFunds
	}
	s.store.UpdateBalance(req.ConsumerWallet, func(current domain.WalletBalance) domain.WalletBalance {
		current.AvailableUSDC -= reservation
		current.ReservedUSDC += reservation
		return current
	})
	quote := domain.JobQuote{
		QuoteID:                  uuid.NewString(),
		ProviderID:               selected.NodeID,
		ReservationUSDC:          reservation,
		MinJobUSDC:               selected.RateCard.MinJobUSDC,
		Input1MUSDC:              selected.RateCard.Input1MUSDC,
		Output1MUSDC:             selected.RateCard.Output1MUSDC,
		ProviderSigningPubkey:    selected.SecureEnclaveSigningPubkey,
		ProviderSessionPubkey:    selected.ProviderSessionPubkey,
		ProviderSessionSignature: selected.ProviderSessionSignature,
		ExpiresAt:                s.now().UTC().Add(2 * time.Minute),
		ConsumerWallet:           strings.ToLower(req.ConsumerWallet),
		ModelID:                  req.ModelID,
	}
	s.store.PutQuote(quote)
	return quote, nil
}

func (s *Service) CreateJob(req domain.JobCreateRequest) (domain.SessionDescriptor, error) {
	s.expireQuotes()
	quote, ok := s.store.GetQuote(req.QuoteID)
	if !ok {
		return domain.SessionDescriptor{}, domain.ErrQuoteNotFound
	}
	if s.now().After(quote.ExpiresAt) {
		return domain.SessionDescriptor{}, domain.ErrQuoteExpired
	}
	if quote.Consumed {
		return domain.SessionDescriptor{}, domain.ErrQuoteConsumed
	}
	if req.MaxSpendUSDC < quote.ReservationUSDC {
		return domain.SessionDescriptor{}, domain.ErrInsufficientFunds
	}
	provider, ok := s.store.GetProvider(quote.ProviderID)
	if !ok || provider.Status != domain.ProviderStatusHealthy {
		return domain.SessionDescriptor{}, domain.ErrNoCapacity
	}
	session := domain.SessionDescriptor{
		JobID:                    uuid.NewString(),
		SessionID:                uuid.NewString(),
		RelayURL:                 s.relayURL,
		ProviderNodeID:           provider.NodeID,
		ProviderSigningPubkey:    provider.SecureEnclaveSigningPubkey,
		ProviderSessionPubkey:    provider.ProviderSessionPubkey,
		ProviderSessionSignature: provider.ProviderSessionSignature,
		ExpiresAt:                s.now().UTC().Add(10 * time.Minute),
	}
	s.store.UpdateQuote(req.QuoteID, func(current domain.JobQuote) domain.JobQuote {
		current.Consumed = true
		return current
	})
	provider.Status = domain.ProviderStatusBusy
	s.store.UpsertProvider(provider)
	s.store.PutJob(domain.JobRecord{
		JobID:                session.JobID,
		SessionID:            session.SessionID,
		State:                domain.JobStateSessionOpen,
		ProviderID:           provider.NodeID,
		ConsumerWallet:       quote.ConsumerWallet,
		ModelID:              quote.ModelID,
		EncryptedJobEnvelope: req.EncryptedJobEnvelope,
		ReservedUSDC:         quote.ReservationUSDC,
		CreatedAt:            s.now().UTC(),
	})
	return session, nil
}

func (s *Service) Job(jobID string) (domain.JobRecord, error) {
	job, ok := s.store.GetJob(jobID)
	if !ok {
		return domain.JobRecord{}, domain.ErrJobNotFound
	}
	return job, nil
}

func (s *Service) CancelJob(jobID string) error {
	job, ok := s.store.GetJob(jobID)
	if !ok {
		return domain.ErrJobNotFound
	}
	if _, ok := s.store.UpdateJob(jobID, func(current domain.JobRecord) domain.JobRecord {
		current.State = domain.JobStateCancelled
		return current
	}); !ok {
		return domain.ErrJobNotFound
	}
	provider, ok := s.store.GetProvider(job.ProviderID)
	if ok {
		provider.Status = domain.ProviderStatusHealthy
		s.store.UpsertProvider(provider)
	}
	s.store.UpdateBalance(job.ConsumerWallet, func(balance domain.WalletBalance) domain.WalletBalance {
		balance.ReservedUSDC -= job.ReservedUSDC
		balance.AvailableUSDC += job.ReservedUSDC
		return balance
	})
	return nil
}

func (s *Service) CompleteJob(jobID string, req domain.JobCompletionRequest) (domain.JobRecord, error) {
	job, ok := s.store.GetJob(jobID)
	if !ok {
		return domain.JobRecord{}, domain.ErrJobNotFound
	}
	if job.State == domain.JobStateCompleted || job.State == domain.JobStateCancelled || job.State == domain.JobStateFailed {
		return domain.JobRecord{}, domain.ErrJobNotCompletable
	}
	provider, ok := s.store.GetProvider(job.ProviderID)
	if !ok {
		return domain.JobRecord{}, domain.ErrNoCapacity
	}
	actualCharge := quoteCost(provider.RateCard, domain.JobQuoteRequest{
		ConsumerWallet:       job.ConsumerWallet,
		ModelID:              job.ModelID,
		EstimatedInputTokens: req.PromptTokens,
		MaxOutputTokens:      req.CompletionTokens,
	})
	if actualCharge > job.ReservedUSDC {
		actualCharge = job.ReservedUSDC
	}
	refund := job.ReservedUSDC - actualCharge
	nonce := s.store.NextSettlementNonce(job.ConsumerWallet)
	s.store.UpdateBalance(job.ConsumerWallet, func(balance domain.WalletBalance) domain.WalletBalance {
		balance.ReservedUSDC -= job.ReservedUSDC
		balance.AvailableUSDC += refund
		return balance
	})
	s.store.UpdateBalance(provider.ProviderWallet, func(balance domain.WalletBalance) domain.WalletBalance {
		balance.WithdrawableUSDC += actualCharge
		return balance
	})
	provider.Status = domain.ProviderStatusHealthy
	s.store.UpsertProvider(provider)
	updated, _ := s.store.UpdateJob(jobID, func(current domain.JobRecord) domain.JobRecord {
		current.State = domain.JobStateCompleted
		current.BilledUSDC = actualCharge
		current.SettlementNonce = nonce
		current.PromptTokens = req.PromptTokens
		current.CompletionTokens = req.CompletionTokens
		return current
	})
	return updated, nil
}

func (s *Service) RunJob(jobID string, req domain.JobRunRequest) (domain.JobRunResult, error) {
	job, ok := s.store.GetJob(jobID)
	if !ok {
		return domain.JobRunResult{}, domain.ErrJobNotFound
	}
	provider, ok := s.store.GetProvider(job.ProviderID)
	if !ok || provider.ControlURL == "" {
		return domain.JobRunResult{}, domain.ErrProviderUnreachable
	}
	client := providerclient.NewClient(provider.ControlURL)
	executeReq := providerclient.ExecuteJobRequest{
		JobID:           jobID,
		Prompt:          req.Prompt,
		MaxOutputTokens: req.MaxOutputTokens,
	}
	if job.EncryptedJobEnvelope != "" && strings.HasPrefix(strings.TrimSpace(job.EncryptedJobEnvelope), "{") {
		executeReq.EncryptedEnvelope = job.EncryptedJobEnvelope
	}
	executed, err := client.ExecuteJob(context.Background(), providerclient.ExecuteJobRequest{
		JobID:             executeReq.JobID,
		Prompt:            executeReq.Prompt,
		MaxOutputTokens:   executeReq.MaxOutputTokens,
		EncryptedEnvelope: executeReq.EncryptedEnvelope,
	})
	if err != nil {
		return domain.JobRunResult{}, err
	}
	completed, err := s.CompleteJob(jobID, domain.JobCompletionRequest{
		PromptTokens:     executed.PromptTokens,
		CompletionTokens: executed.CompletionTokens,
	})
	if err != nil {
		return domain.JobRunResult{}, err
	}
	return domain.JobRunResult{
		JobID:            completed.JobID,
		OutputText:       executed.OutputText,
		PromptTokens:     executed.PromptTokens,
		CompletionTokens: executed.CompletionTokens,
		BilledUSDC:       completed.BilledUSDC,
		Status:           string(completed.State),
	}, nil
}

func (s *Service) SettlementVoucher(jobID string) (domain.SettlementVoucherResponse, error) {
	job, ok := s.store.GetJob(jobID)
	if !ok {
		return domain.SettlementVoucherResponse{}, domain.ErrJobNotFound
	}
	if job.State != domain.JobStateCompleted {
		return domain.SettlementVoucherResponse{}, domain.ErrJobNotCompletable
	}
	provider, ok := s.store.GetProvider(job.ProviderID)
	if !ok {
		return domain.SettlementVoucherResponse{}, domain.ErrNoCapacity
	}
	return makeSettlementResponse(
		s.signerKey,
		s.chainID,
		s.contract,
		job,
		job.ConsumerWallet,
		provider.ProviderWallet,
		s.now().UTC().Add(24*time.Hour),
	)
}

func normalizeRegistration(req domain.ProviderRegistration) domain.ProviderRegistration {
	req.ProviderWallet = strings.ToLower(req.ProviderWallet)
	req.RateCard = normalizeRateCard(req.RateCard)
	return req
}

func normalizeRateCard(card domain.RateCard) domain.RateCard {
	if card.MinJobUSDC < 0 {
		card.MinJobUSDC = 0
	}
	if card.Input1MUSDC < 0 {
		card.Input1MUSDC = 0
	}
	if card.Output1MUSDC < 0 {
		card.Output1MUSDC = 0
	}
	return card
}

func quoteCost(card domain.RateCard, req domain.JobQuoteRequest) int64 {
	usageCost := ((req.EstimatedInputTokens * card.Input1MUSDC) + 999_999) / 1_000_000
	usageCost += ((req.MaxOutputTokens * card.Output1MUSDC) + 999_999) / 1_000_000
	if usageCost < card.MinJobUSDC {
		return card.MinJobUSDC
	}
	return usageCost
}

func compareCosts(left, right int64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func (s *Service) catalogSupports(modelID string, memoryGB int) bool {
	entry, ok := s.catalogEntry(modelID)
	return ok && memoryGB >= entry.MinimumMemoryGB
}

func (s *Service) catalogEntry(modelID string) (domain.CatalogEntry, bool) {
	for _, entry := range s.catalog {
		if entry.ModelID == modelID {
			return entry, true
		}
	}
	return domain.CatalogEntry{}, false
}

func (s *Service) expireQuotes() {
	for _, quote := range s.store.ListQuotes() {
		if quote.Consumed || !s.now().After(quote.ExpiresAt) {
			continue
		}
		s.store.UpdateBalance(quote.ConsumerWallet, func(balance domain.WalletBalance) domain.WalletBalance {
			balance.ReservedUSDC -= quote.ReservationUSDC
			balance.AvailableUSDC += quote.ReservationUSDC
			return balance
		})
		s.store.DeleteQuote(quote.QuoteID)
	}
}

func ValidateRelayURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("relayURL must include scheme and host")
	}
	return nil
}
