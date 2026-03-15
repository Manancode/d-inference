package api

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/app"
	"github.com/dginf/dginf/services/coordinator/internal/domain"
	"github.com/dginf/dginf/services/coordinator/internal/store"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestChallengeEndpoint(t *testing.T) {
	service := app.NewService(store.NewMemory(), "", func() time.Time {
		return time.Unix(1_700_000_000, 0)
	})
	server := NewServer(service)
	body := bytes.NewBufferString(`{"wallet":"0xabc","chainId":8453}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/challenge", body)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var response domain.AuthChallengeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Nonce == "" || response.Message == "" {
		t.Fatalf("expected nonce and message, got %#v", response)
	}
}

func TestQuoteEndpoint(t *testing.T) {
	service := app.NewService(store.NewMemory(), "", func() time.Time {
		return time.Unix(1_700_000_000, 0)
	})
	service.SeedBalance("0xconsumer", 10_000)
	if err := service.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk",
		ProviderSessionPubkey:      "session",
		ProviderSessionSignature:   "sig",
		HardwareProfile:            "M3 Max",
		MemoryGB:                   64,
		SelectedModelID:            "qwen3.5-35b-a3b",
		RateCard: domain.RateCard{
			MinJobUSDC:   100,
			Input1MUSDC:  20_000,
			Output1MUSDC: 40_000,
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	server := NewServer(service)
	body := bytes.NewBufferString(`{"consumerWallet":"0xconsumer","modelId":"qwen3.5-35b-a3b","estimatedInputTokens":1000,"maxOutputTokens":1000}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs/quote", body)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestProvidersEndpoint(t *testing.T) {
	service := app.NewService(store.NewMemory(), "", time.Now)
	if err := service.RegisterProvider(domain.ProviderRegistration{
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
	server := NewServer(service)
	request := httptest.NewRequest(http.MethodGet, "/v1/providers", nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCompleteJobEndpoint(t *testing.T) {
	service := app.NewService(store.NewMemory(), "", func() time.Time {
		return time.Unix(1_700_000_000, 0)
	})
	service.SeedBalance("0xconsumer", 20_000)
	if err := service.RegisterProvider(domain.ProviderRegistration{
		ProviderWallet:             "0xprovider",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk",
		ProviderSessionPubkey:      "session",
		ProviderSessionSignature:   "sig",
		HardwareProfile:            "M3 Max",
		MemoryGB:                   64,
		SelectedModelID:            "qwen3.5-35b-a3b",
		RateCard: domain.RateCard{
			MinJobUSDC:   100,
			Input1MUSDC:  20_000,
			Output1MUSDC: 40_000,
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	quote, err := service.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xconsumer",
		ModelID:              "qwen3.5-35b-a3b",
		EstimatedInputTokens: 1_000,
		MaxOutputTokens:      1_000,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	session, err := service.CreateJob(domain.JobCreateRequest{
		QuoteID:               quote.QuoteID,
		ClientEphemeralPubkey: "client",
		EncryptedJobEnvelope:  "ciphertext",
		MaxSpendUSDC:          quote.ReservationUSDC,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	server := NewServer(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs/"+session.JobID+"/complete", bytes.NewBufferString(`{"promptTokens":800,"completionTokens":600}`))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSeedBalanceEndpoint(t *testing.T) {
	service := app.NewService(store.NewMemory(), "", time.Now)
	server := NewServer(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/dev/seed-balance", bytes.NewBufferString(`{"wallet":"0xabc","availableUsdc":1234,"withdrawableUsdc":56}`))
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRunJobEndpoint(t *testing.T) {
	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/jobs/execute" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jobId":"job-1","outputText":"hello","promptTokens":12,"completionTokens":8}`))
	}))
	defer providerServer.Close()

	service := app.NewService(store.NewMemory(), "", func() time.Time {
		return time.Unix(1_700_000_000, 0)
	})
	service.SeedBalance("0xconsumer", 20_000)
	if err := service.RegisterProvider(domain.ProviderRegistration{
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
	quote, err := service.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xconsumer",
		ModelID:              "qwen3.5-35b-a3b",
		EstimatedInputTokens: 12,
		MaxOutputTokens:      8,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	job, err := service.CreateJob(domain.JobCreateRequest{
		QuoteID:               quote.QuoteID,
		ClientEphemeralPubkey: "client",
		EncryptedJobEnvelope:  "ciphertext",
		MaxSpendUSDC:          quote.ReservationUSDC,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	server := NewServer(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs/"+job.JobID+"/run", bytes.NewBufferString(`{"prompt":"hello world","maxOutputTokens":8}`))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSettlementVoucherEndpoint(t *testing.T) {
	service := app.NewServiceWithSigner(store.NewMemory(), "", func() time.Time {
		return time.Unix(1_700_000_000, 0)
	}, mustTestSigner(t), 8453, "0x0000000000000000000000000000000000000001")
	service.SeedBalance("0xconsumer", 20_000)
	if err := service.RegisterProvider(domain.ProviderRegistration{
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
	quote, err := service.QuoteJob(domain.JobQuoteRequest{
		ConsumerWallet:       "0xconsumer",
		ModelID:              "qwen3.5-35b-a3b",
		EstimatedInputTokens: 12,
		MaxOutputTokens:      8,
	})
	if err != nil {
		t.Fatalf("quote job: %v", err)
	}
	job, err := service.CreateJob(domain.JobCreateRequest{
		QuoteID:               quote.QuoteID,
		ClientEphemeralPubkey: "client",
		EncryptedJobEnvelope:  "ciphertext",
		MaxSpendUSDC:          quote.ReservationUSDC,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := service.CompleteJob(job.JobID, domain.JobCompletionRequest{PromptTokens: 12, CompletionTokens: 8}); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	server := NewServer(service)
	request := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+job.JobID+"/settlement-voucher", nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
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
