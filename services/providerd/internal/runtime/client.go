package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrRuntimeUnavailable = errors.New("runtime unavailable")

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type LoadModelRequest struct {
	ModelID   string `json:"model_id"`
	ModelPath string `json:"model_path"`
}

type LoadModelResponse struct {
	ModelID     string    `json:"model_id"`
	BackendName string    `json:"backend_name"`
	LoadedAt    time.Time `json:"loaded_at"`
}

type GenerateRequest struct {
	JobID           string `json:"job_id"`
	Prompt          string `json:"prompt"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}

type GenerateResponse struct {
	JobID            string    `json:"job_id"`
	ModelID          string    `json:"model_id"`
	OutputText       string    `json:"output_text"`
	PromptTokens     int64     `json:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens"`
	State            string    `json:"state"`
	FinishedAt       time.Time `json:"finished_at"`
}

type HealthResponse struct {
	Status        string   `json:"status"`
	BackendName   string   `json:"backend_name"`
	LoadedModelID string   `json:"loaded_model_id"`
	ActiveJobs    int      `json:"active_job_count"`
	Notes         []string `json:"notes"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) LoadModel(ctx context.Context, request LoadModelRequest) (LoadModelResponse, error) {
	var response LoadModelResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/models/load", request, &response)
	return response, err
}

func (c *Client) Generate(ctx context.Context, request GenerateRequest) (GenerateResponse, error) {
	var response GenerateResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/jobs/generate", request, &response)
	return response, err
}

func (c *Client) Health(ctx context.Context) (HealthResponse, error) {
	var response HealthResponse
	err := c.doJSON(ctx, http.MethodGet, "/v1/health", nil, &response)
	return response, err
}

func (c *Client) doJSON(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	if c == nil || c.baseURL == "" {
		return ErrRuntimeUnavailable
	}

	var bodyReader *bytes.Reader
	if requestBody == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRuntimeUnavailable, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return fmt.Errorf("%w: runtime returned %d", ErrRuntimeUnavailable, response.StatusCode)
	}
	if responseBody == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(responseBody)
}
