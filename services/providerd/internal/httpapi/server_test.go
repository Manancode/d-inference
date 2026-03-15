package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dginf/dginf/services/providerd/internal/runtime"
)

func TestHealthEndpoint(t *testing.T) {
	server := New(runtime.NewService())
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestLoadModelUpdatesHealth(t *testing.T) {
	server := New(runtime.NewService())
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/v1/runtime/load-model", bytes.NewBufferString(`{"model":"qwen3.5-35b-a3b"}`)),
	)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", recorder.Code)
	}

	var payload runtime.State
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.LoadedModel != "qwen3.5-35b-a3b" {
		t.Fatalf("expected loaded model to update, got %q", payload.LoadedModel)
	}
}

func TestLoadModelRejectsMissingModel(t *testing.T) {
	server := New(runtime.NewService())
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/v1/runtime/load-model", bytes.NewBufferString(`{"model":""}`)),
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
