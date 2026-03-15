package coordinator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAndHeartbeat(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		switch r.URL.Path {
		case "/v1/providers/register":
			w.WriteHeader(http.StatusCreated)
		case "/v1/providers/heartbeat":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if err := client.RegisterProvider(context.Background(), RegisterProviderRequest{
		ProviderWallet:             "0xprovider",
		NodeID:                     "node-1",
		SecureEnclaveSigningPubkey: "pk",
		ProviderSessionPubkey:      "session",
		ProviderSessionSignature:   "sig",
		HardwareProfile:            "M3 Max 64GB",
		MemoryGB:                   64,
		SelectedModelID:            "qwen3.5-35b-a3b",
		RateCard: RateCard{
			MinJobUSDC:   100,
			Input1MUSDC:  10_000,
			Output1MUSDC: 20_000,
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	if err := client.Heartbeat(context.Background(), HeartbeatRequest{
		NodeID:          "node-1",
		Status:          "ready",
		SelectedModelID: "qwen3.5-35b-a3b",
	}); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	first := <-requests
	second := <-requests
	if first != "/v1/providers/register" || second != "/v1/providers/heartbeat" {
		t.Fatalf("unexpected path order: %s then %s", first, second)
	}
}
