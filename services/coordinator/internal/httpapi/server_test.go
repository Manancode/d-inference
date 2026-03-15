package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/auth"
	"github.com/dginf/dginf/services/coordinator/internal/catalog"
)

func TestHealthz(t *testing.T) {
	server := New(auth.NewChallengeService(nil), catalog.DefaultEntries())
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestModelsEndpointReturnsCatalog(t *testing.T) {
	server := New(auth.NewChallengeService(nil), catalog.DefaultEntries())
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/models", nil))

	var payload struct {
		Models []catalog.Entry `json:"models"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Models) != 4 {
		t.Fatalf("expected 4 catalog entries, got %d", len(payload.Models))
	}
}

func TestAuthChallengeEndpoint(t *testing.T) {
	now := func() time.Time {
		return time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	}
	server := New(auth.NewChallengeService(now), catalog.DefaultEntries())
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"wallet":"0xabc"}`)

	server.Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/v1/auth/challenge", body),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["nonce"] == "" {
		t.Fatal("expected nonce to be present")
	}
}

func TestAuthChallengeRejectsMissingWallet(t *testing.T) {
	server := New(auth.NewChallengeService(nil), catalog.DefaultEntries())
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/v1/auth/challenge", bytes.NewBufferString(`{"wallet":""}`)),
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
