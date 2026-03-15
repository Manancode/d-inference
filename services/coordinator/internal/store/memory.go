package store

import (
	"strings"
	"sync"

	"github.com/dginf/dginf/services/coordinator/internal/domain"
)

type Memory struct {
	mu         sync.RWMutex
	challenges map[string]domain.AuthChallenge
	sessions   map[string]string
	providers  map[string]domain.Provider
	balances   map[string]domain.WalletBalance
	quotes     map[string]domain.JobQuote
	jobs       map[string]domain.JobRecord
	nonces     map[string]uint64
}

func NewMemory() *Memory {
	return &Memory{
		challenges: map[string]domain.AuthChallenge{},
		sessions:   map[string]string{},
		providers:  map[string]domain.Provider{},
		balances:   map[string]domain.WalletBalance{},
		quotes:     map[string]domain.JobQuote{},
		jobs:       map[string]domain.JobRecord{},
		nonces:     map[string]uint64{},
	}
}

func normalizeWallet(wallet string) string {
	return strings.ToLower(wallet)
}

func (m *Memory) PutChallenge(ch domain.AuthChallenge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.challenges[ch.Nonce] = ch
}

func (m *Memory) GetChallenge(nonce string) (domain.AuthChallenge, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.challenges[nonce]
	return ch, ok
}

func (m *Memory) DeleteChallenge(nonce string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.challenges, nonce)
}

func (m *Memory) PutSession(token, wallet string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = normalizeWallet(wallet)
}

func (m *Memory) UpsertProvider(provider domain.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider.NodeID] = provider
}

func (m *Memory) GetProvider(nodeID string) (domain.Provider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	provider, ok := m.providers[nodeID]
	return provider, ok
}

func (m *Memory) ListProviders() []domain.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	providers := make([]domain.Provider, 0, len(m.providers))
	for _, provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers
}

func (m *Memory) PutBalance(balance domain.WalletBalance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	balance.Wallet = normalizeWallet(balance.Wallet)
	m.balances[balance.Wallet] = balance
}

func (m *Memory) GetBalance(wallet string) domain.WalletBalance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	balance, ok := m.balances[normalizeWallet(wallet)]
	if !ok {
		return domain.WalletBalance{Wallet: normalizeWallet(wallet)}
	}
	return balance
}

func (m *Memory) UpdateBalance(wallet string, fn func(balance domain.WalletBalance) domain.WalletBalance) domain.WalletBalance {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := normalizeWallet(wallet)
	balance := m.balances[key]
	balance.Wallet = key
	balance = fn(balance)
	m.balances[key] = balance
	return balance
}

func (m *Memory) PutQuote(quote domain.JobQuote) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quotes[quote.QuoteID] = quote
}

func (m *Memory) GetQuote(quoteID string) (domain.JobQuote, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	quote, ok := m.quotes[quoteID]
	return quote, ok
}

func (m *Memory) UpdateQuote(quoteID string, fn func(quote domain.JobQuote) domain.JobQuote) (domain.JobQuote, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	quote, ok := m.quotes[quoteID]
	if !ok {
		return domain.JobQuote{}, false
	}
	quote = fn(quote)
	m.quotes[quoteID] = quote
	return quote, true
}

func (m *Memory) ListQuotes() []domain.JobQuote {
	m.mu.RLock()
	defer m.mu.RUnlock()
	quotes := make([]domain.JobQuote, 0, len(m.quotes))
	for _, quote := range m.quotes {
		quotes = append(quotes, quote)
	}
	return quotes
}

func (m *Memory) DeleteQuote(quoteID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.quotes, quoteID)
}

func (m *Memory) PutJob(job domain.JobRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.JobID] = job
}

func (m *Memory) GetJob(jobID string) (domain.JobRecord, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[jobID]
	return job, ok
}

func (m *Memory) UpdateJob(jobID string, fn func(job domain.JobRecord) domain.JobRecord) (domain.JobRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return domain.JobRecord{}, false
	}
	job = fn(job)
	m.jobs[jobID] = job
	return job, true
}

func (m *Memory) NextSettlementNonce(wallet string) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := normalizeWallet(wallet)
	m.nonces[key]++
	return m.nonces[key]
}
