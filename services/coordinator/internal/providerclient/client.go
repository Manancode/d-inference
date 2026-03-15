package providerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type ExecuteJobRequest struct {
	JobID             string `json:"jobId"`
	Prompt            string `json:"prompt"`
	MaxOutputTokens   int    `json:"maxOutputTokens"`
	EncryptedEnvelope string `json:"encryptedEnvelope,omitempty"`
}

type ExecuteJobResult struct {
	JobID            string `json:"jobId"`
	OutputText       string `json:"outputText"`
	PromptTokens     int64  `json:"promptTokens"`
	CompletionTokens int64  `json:"completionTokens"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) ExecuteJob(ctx context.Context, req ExecuteJobRequest) (ExecuteJobResult, error) {
	var result ExecuteJobResult
	raw, err := json.Marshal(req)
	if err != nil {
		return result, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/jobs/execute", bytes.NewReader(raw))
	if err != nil {
		return result, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return result, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("provider returned status %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}
