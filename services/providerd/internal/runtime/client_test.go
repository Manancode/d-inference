package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientLoadModelAndGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer server.Close()

	client := NewClient(server.URL)
	loadResult, err := client.LoadModel(context.Background(), LoadModelRequest{
		ModelID:   "qwen3.5-35b-a3b",
		ModelPath: "/tmp/models/qwen3.5-35b-a3b",
	})
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	if loadResult.ModelID != "qwen3.5-35b-a3b" {
		t.Fatalf("unexpected load result: %#v", loadResult)
	}

	generateResult, err := client.Generate(context.Background(), GenerateRequest{
		JobID:           "job-1",
		Prompt:          "hello world",
		MaxOutputTokens: 16,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if generateResult.CompletionTokens != 2 {
		t.Fatalf("unexpected completion tokens: %#v", generateResult)
	}
}
