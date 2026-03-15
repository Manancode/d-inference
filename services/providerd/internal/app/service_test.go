package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coordclient "github.com/dginf/dginf/services/providerd/internal/coordinator"
	"github.com/dginf/dginf/services/providerd/internal/domain"
	"github.com/dginf/dginf/services/providerd/internal/identity"
	"github.com/dginf/dginf/services/providerd/internal/posture"
	"github.com/dginf/dginf/services/providerd/internal/runtime"
	"github.com/dginf/dginf/services/providerd/internal/store"
)

func TestNodeLifecycle(t *testing.T) {
	signer, err := identity.NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	svc := NewService(store.NewMemory(), signer, sessionKeys, nil, nil, nil, func() time.Time { return time.Unix(1_700_000_000, 0) })
	status, err := svc.Bootstrap(domain.NodeConfig{
		NodeID:         "node-1",
		ProviderWallet: "0xprovider",
		PublicURL:      "http://127.0.0.1:8787",
		SelectedModel:  "qwen3.5-35b-a3b",
	})
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if status.State != domain.NodeStateReady {
		t.Fatalf("expected ready state, got %s", status.State)
	}
	status, err = svc.StartJob(domain.StartJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("start job: %v", err)
	}
	if status.State != domain.NodeStateBusy {
		t.Fatalf("expected busy state, got %s", status.State)
	}
	if _, err := svc.StartJob(domain.StartJobRequest{JobID: "job-2"}); err != ErrNodeBusy {
		t.Fatalf("expected busy error, got %v", err)
	}
	status, err = svc.CompleteJob()
	if err != nil {
		t.Fatalf("complete job: %v", err)
	}
	if status.State != domain.NodeStateReady {
		t.Fatalf("expected ready after complete, got %s", status.State)
	}
}

func TestPausePreventsJobStart(t *testing.T) {
	signer, err := identity.NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	svc := NewService(store.NewMemory(), signer, sessionKeys, nil, nil, nil, time.Now)
	if _, err := svc.Bootstrap(domain.NodeConfig{
		NodeID:         "node-1",
		ProviderWallet: "0xprovider",
		PublicURL:      "http://127.0.0.1:8787",
		SelectedModel:  "qwen3.5-35b-a3b",
	}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	svc.Pause()
	if _, err := svc.StartJob(domain.StartJobRequest{JobID: "job-1"}); err != ErrNodePaused {
		t.Fatalf("expected paused error, got %v", err)
	}
}

func TestLoadSelectedModelAndExecuteJob(t *testing.T) {
	server := httptestRuntimeServer(t)
	signer, err := identity.NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	svc := NewService(store.NewMemory(), signer, sessionKeys, nil, runtime.NewClient(server.URL), nil, time.Now)
	if _, err := svc.Bootstrap(domain.NodeConfig{
		NodeID:         "node-1",
		ProviderWallet: "0xprovider",
		PublicURL:      "http://127.0.0.1:8787",
		SelectedModel:  "qwen3.5-35b-a3b",
	}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := svc.LoadSelectedModel(context.Background()); err != nil {
		t.Fatalf("load selected model: %v", err)
	}
	result, err := svc.ExecuteJob(context.Background(), domain.ExecuteJobRequest{
		JobID:           "job-1",
		Prompt:          "hello world",
		MaxOutputTokens: 16,
	})
	if err != nil {
		t.Fatalf("execute job: %v", err)
	}
	if result.CompletionTokens != 2 || result.OutputText == "" {
		t.Fatalf("unexpected execute result: %#v", result)
	}
	if svc.Status().State != domain.NodeStateReady {
		t.Fatalf("expected ready after execute, got %s", svc.Status().State)
	}
}

func TestRegisterAndHeartbeatWithCoordinator(t *testing.T) {
	requestCount := 0
	coordinatorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch r.URL.Path {
		case "/v1/providers/register":
			w.WriteHeader(http.StatusCreated)
		case "/v1/providers/heartbeat":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer coordinatorServer.Close()

	signer, err := identity.NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	svc := NewService(store.NewMemory(), signer, sessionKeys, coordclient.NewClient(coordinatorServer.URL), nil, posture.NewCollector(time.Now), time.Now)
	if _, err := svc.Bootstrap(domain.NodeConfig{
		NodeID:          "node-1",
		ProviderWallet:  "0xprovider",
		PublicURL:       "http://127.0.0.1:8787",
		SelectedModel:   "qwen3.5-35b-a3b",
		MemoryGB:        64,
		HardwareProfile: "M3 Max 64GB",
		MinJobUSDC:      100,
		Input1MUSDC:     10_000,
		Output1MUSDC:    20_000,
	}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	if err := svc.RegisterWithCoordinator(context.Background()); err != nil {
		t.Fatalf("register with coordinator: %v", err)
	}
	if err := svc.SendHeartbeat(context.Background()); err != nil {
		t.Fatalf("send heartbeat: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 coordinator requests, got %d", requestCount)
	}
}

func TestSignedPostureProducesSignature(t *testing.T) {
	signer, err := identity.NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	sessionKeys, err := identity.NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	collector := &posture.Collector{}
	collector = posture.NewCollector(func() time.Time { return time.Unix(1_700_000_000, 0) })
	svc := NewService(store.NewMemory(), signer, sessionKeys, nil, nil, collector, time.Now)
	report, err := svc.SignedPosture()
	if err != nil {
		t.Fatalf("signed posture: %v", err)
	}
	if report == nil || report.Signature == "" {
		t.Fatalf("expected posture signature, got %#v", report)
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
