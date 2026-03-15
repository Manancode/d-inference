package app

import (
	"crypto/ecdsa"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/domain"
	"github.com/dginf/dginf/services/coordinator/internal/store"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestQuoteSelectsLowestCostHealthyProvider(t *testing.T) {
	memory := store.NewMemory()
	now := time.Unix(1_700_000_000, 0)
	svc := NewService(memory, "", func() time.Time { return now })
	svc.SeedBalance("0xabc", 10_000)

	err := svc.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider1",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk1",
		ProviderSessionPubkey:      "session1",
		ProviderSessionSignature:   "sig1",
		HardwareProfile:            "M3 Max",
		MemoryGB:                   64,
		SelectedModelID:            "qwen3.5-35b-a3b",
		RateCard: domain.RateCard{
			MinJobUSDC:   10,
			Input1MUSDC:  20_000,
			Output1MUSDC: 40_000,
		},
	})
	if err != nil {
		t.Fatalf("register first provider: %v", err)
	}
	err = svc.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider2",
		NodeID:                     "node-2",
		SecureEnclaveSigningPubkey: "pk2",
		ProviderSessionPubkey:      "session2",
		ProviderSessionSignature:   "sig2",
		ControlURL:                 "http://127.0.0.1:9999",
		HardwareProfile:            "M3 Max",
		MemoryGB:                   64,
		SelectedModelID:            "qwen3.5-35b-a3b",
		RateCard: domain.RateCard{
			MinJobUSDC:   10,
			Input1MUSDC:  10_000,
			Output1MUSDC: 20_000,
		},
	})
	if err != nil {
		t.Fatalf("register second provider: %v", err)
	}

	quote, err := svc.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xabc",
		ModelID:              "qwen3.5-35b-a3b",
		EstimatedInputTokens: 2_000,
		MaxOutputTokens:      1_000,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	if quote.ProviderID != "node-2" {
		t.Fatalf("expected cheaper provider node-2, got %s", quote.ProviderID)
	}
	balance := svc.Balance("0xabc")
	if balance.AvailableUSDC != 9_960 || balance.ReservedUSDC != 40 {
		t.Fatalf("unexpected balance after reservation: %#v", balance)
	}
}

func TestCreateAndCancelJobRestoresReservation(t *testing.T) {
	memory := store.NewMemory()
	now := time.Unix(1_700_000_000, 0)
	svc := NewService(memory, "", func() time.Time { return now })
	svc.SeedBalance("0xconsumer", 20_000)
	if err := svc.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk",
		ProviderSessionPubkey:      "session",
		ProviderSessionSignature:   "sig",
		ControlURL:                 "http://127.0.0.1:9999",
		HardwareProfile:            "M3 Max",
		MemoryGB:                   128,
		SelectedModelID:            "qwen3.5-122b-a10b",
		RateCard: domain.RateCard{
			MinJobUSDC:   100,
			Input1MUSDC:  30_000,
			Output1MUSDC: 50_000,
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	quote, err := svc.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xconsumer",
		ModelID:              "qwen3.5-122b-a10b",
		EstimatedInputTokens: 1_000,
		MaxOutputTokens:      1_000,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	session, err := svc.CreateJob(domain.JobCreateRequest{
		QuoteID:               quote.QuoteID,
		ClientEphemeralPubkey: "client",
		EncryptedJobEnvelope:  "ciphertext",
		MaxSpendUSDC:          quote.ReservationUSDC,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	job, err := svc.Job(session.JobID)
	if err != nil {
		t.Fatalf("load job: %v", err)
	}
	if job.State != domain.JobStateSessionOpen {
		t.Fatalf("expected session_open, got %s", job.State)
	}
	if err := svc.CancelJob(session.JobID); err != nil {
		t.Fatalf("cancel job: %v", err)
	}
	balance := svc.Balance("0xconsumer")
	if balance.AvailableUSDC != 20_000 || balance.ReservedUSDC != 0 {
		t.Fatalf("expected reservation restored, got %#v", balance)
	}
}

func TestCompleteJobSettlesBalancesAndProviderEarnings(t *testing.T) {
	memory := store.NewMemory()
	now := time.Unix(1_700_000_000, 0)
	svc := NewService(memory, "", func() time.Time { return now })
	svc.SeedBalance("0xconsumer", 20_000)
	if err := svc.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk",
		ProviderSessionPubkey:      "session",
		ProviderSessionSignature:   "sig",
		ControlURL:                 "http://127.0.0.1:9999",
		HardwareProfile:            "M3 Max",
		MemoryGB:                   128,
		SelectedModelID:            "qwen3.5-122b-a10b",
		RateCard: domain.RateCard{
			MinJobUSDC:   100,
			Input1MUSDC:  30_000,
			Output1MUSDC: 50_000,
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	quote, err := svc.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xconsumer",
		ModelID:              "qwen3.5-122b-a10b",
		EstimatedInputTokens: 1_000,
		MaxOutputTokens:      1_000,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	session, err := svc.CreateJob(domain.JobCreateRequest{
		QuoteID:               quote.QuoteID,
		ClientEphemeralPubkey: "client",
		EncryptedJobEnvelope:  "ciphertext",
		MaxSpendUSDC:          quote.ReservationUSDC,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	job, err := svc.CompleteJob(session.JobID, domain.JobCompletionRequest{
		PromptTokens:     800,
		CompletionTokens: 600,
	})
	if err != nil {
		t.Fatalf("complete job: %v", err)
	}
	if job.State != domain.JobStateCompleted {
		t.Fatalf("expected completed state, got %s", job.State)
	}
	if job.BilledUSDC != 100 {
		t.Fatalf("expected minimum charge of 100, got %d", job.BilledUSDC)
	}
	consumerBalance := svc.Balance("0xconsumer")
	if consumerBalance.ReservedUSDC != 0 || consumerBalance.AvailableUSDC != 19_900 {
		t.Fatalf("unexpected consumer balance: %#v", consumerBalance)
	}
	providerBalance := svc.Balance("0xprovider")
	if providerBalance.WithdrawableUSDC != 100 {
		t.Fatalf("unexpected provider balance: %#v", providerBalance)
	}
}

func TestRunJobDispatchesToProviderAndSettles(t *testing.T) {
	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/jobs/execute" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jobId":"job-1","outputText":"hello","promptTokens":12,"completionTokens":8}`))
	}))
	defer providerServer.Close()

	memory := store.NewMemory()
	now := time.Unix(1_700_000_000, 0)
	svc := NewService(memory, "", func() time.Time { return now })
	svc.SeedBalance("0xconsumer", 20_000)
	if err := svc.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk",
		ProviderSessionPubkey:      "session",
		ProviderSessionSignature:   "sig",
		ControlURL:                 providerServer.URL,
		HardwareProfile:            "M3 Max",
		MemoryGB:                   64,
		SelectedModelID:            "qwen3.5-35b-a3b",
		RateCard: domain.RateCard{
			MinJobUSDC:   100,
			Input1MUSDC:  10_000,
			Output1MUSDC: 20_000,
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	quote, err := svc.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xconsumer",
		ModelID:              "qwen3.5-35b-a3b",
		EstimatedInputTokens: 12,
		MaxOutputTokens:      8,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	session, err := svc.CreateJob(domain.JobCreateRequest{
		QuoteID:               quote.QuoteID,
		ClientEphemeralPubkey: "client",
		EncryptedJobEnvelope:  "ciphertext",
		MaxSpendUSDC:          quote.ReservationUSDC,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	result, err := svc.RunJob(session.JobID, domain.JobRunRequest{
		Prompt:          "hello world",
		MaxOutputTokens: 8,
	})
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if result.OutputText != "hello" || result.BilledUSDC != 100 {
		t.Fatalf("unexpected run result: %#v", result)
	}
}

func TestSettlementVoucherReturnsSignedPayload(t *testing.T) {
	signer := mustTestSigner(t)
	memory := store.NewMemory()
	now := time.Unix(1_700_000_000, 0)
	svc := NewServiceWithSigner(memory, "", func() time.Time { return now }, signer, 8453, "0x0000000000000000000000000000000000000001")
	svc.SeedBalance("0xconsumer", 20_000)
	if err := svc.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk",
		ProviderSessionPubkey:      "session",
		ProviderSessionSignature:   "sig",
		ControlURL:                 "http://127.0.0.1:9999",
		HardwareProfile:            "M3 Max",
		MemoryGB:                   64,
		SelectedModelID:            "qwen3.5-35b-a3b",
		RateCard: domain.RateCard{
			MinJobUSDC:   100,
			Input1MUSDC:  10_000,
			Output1MUSDC: 20_000,
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	quote, err := svc.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xconsumer",
		ModelID:              "qwen3.5-35b-a3b",
		EstimatedInputTokens: 12,
		MaxOutputTokens:      8,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	session, err := svc.CreateJob(domain.JobCreateRequest{
		QuoteID:               quote.QuoteID,
		ClientEphemeralPubkey: "client",
		EncryptedJobEnvelope:  "ciphertext",
		MaxSpendUSDC:          quote.ReservationUSDC,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := svc.CompleteJob(session.JobID, domain.JobCompletionRequest{PromptTokens: 12, CompletionTokens: 8}); err != nil {
		t.Fatalf("complete job: %v", err)
	}
	voucher, err := svc.SettlementVoucher(session.JobID)
	if err != nil {
		t.Fatalf("settlement voucher: %v", err)
	}
	if voucher.Voucher.Nonce != 1 || voucher.Signature == "" {
		t.Fatalf("unexpected voucher response: %#v", voucher)
	}
}

func mustTestSigner(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	if err != nil {
		t.Fatalf("load signer: %v", err)
	}
	return key
}
