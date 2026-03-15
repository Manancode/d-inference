package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dginf/dginf/services/providerd/internal/app"
	"github.com/dginf/dginf/services/providerd/internal/domain"
	"github.com/dginf/dginf/services/providerd/internal/identity"
	"github.com/dginf/dginf/services/providerd/internal/runtime"
	"github.com/dginf/dginf/services/providerd/internal/store"
)

func TestStatusEndpoint(t *testing.T) {
	signer, err := identity.NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	service := app.NewService(store.NewMemory(), signer, sessionKeys, nil, nil, nil, time.Now)
	if _, err := service.Bootstrap(domain.NodeConfig{
		NodeID:         "node-1",
		ProviderWallet: "0xprovider",
		PublicURL:      "http://127.0.0.1:8787",
		SelectedModel:  "qwen3.5-35b-a3b",
	}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	server := NewServer(service)
	request := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func httptestRuntimeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models/load":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"model_id":"qwen3.5-35b-a3b","backend_name":"mlx","loaded_at":"2026-03-14T18:00:00Z"}`))
		case "/v1/jobs/generate":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"job_id":"job-1","model_id":"qwen3.5-35b-a3b","output_text":"hello world","prompt_tokens":2,"completion_tokens":2,"state":"completed","finished_at":"2026-03-14T18:00:01Z"}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestExecuteJobEndpoint(t *testing.T) {
	runtimeServer := httptestRuntimeServer(t)
	signer, err := identity.NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	service := app.NewService(store.NewMemory(), signer, sessionKeys, nil, runtime.NewClient(runtimeServer.URL), nil, time.Now)
	if _, err := service.Bootstrap(domain.NodeConfig{
		NodeID:         "node-1",
		ProviderWallet: "0xprovider",
		PublicURL:      "http://127.0.0.1:8787",
		SelectedModel:  "qwen3.5-35b-a3b",
	}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := service.LoadSelectedModel(context.Background()); err != nil {
		t.Fatalf("load selected model: %v", err)
	}
	server := NewServer(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs/execute", bytes.NewBufferString(`{"jobId":"job-1","prompt":"hello world","maxOutputTokens":16}`))
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}
