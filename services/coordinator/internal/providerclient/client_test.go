package providerclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/jobs/execute" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jobId":"job-1","outputText":"hello","promptTokens":2,"completionTokens":2}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.ExecuteJob(context.Background(), ExecuteJobRequest{
		JobID:           "job-1",
		Prompt:          "hello world",
		MaxOutputTokens: 16,
	})
	if err != nil {
		t.Fatalf("execute job: %v", err)
	}
	if result.OutputText != "hello" || result.CompletionTokens != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
}
