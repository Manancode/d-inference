package coordinator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dginf/dginf/services/providerd/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type RateCard struct {
	MinJobUSDC   int64 `json:"minJobUsdc"`
	Input1MUSDC  int64 `json:"input1mUsdc"`
	Output1MUSDC int64 `json:"output1mUsdc"`
}

type RegisterProviderRequest struct {
	ProviderWallet             string   `json:"providerWallet"`
	NodeID                     string   `json:"nodeId"`
	SecureEnclaveSigningPubkey string   `json:"secureEnclaveSigningPubkey"`
	ProviderSessionPubkey      string   `json:"providerSessionPubkey"`
	ProviderSessionSignature   string   `json:"providerSessionSignature"`
	ControlURL                 string   `json:"controlUrl,omitempty"`
	HardwareProfile            string   `json:"hardwareProfile"`
	MemoryGB                   int      `json:"memoryGb"`
	SelectedModelID            string   `json:"selectedModelId"`
	RateCard                   RateCard `json:"rateCard"`
}

type HeartbeatRequest struct {
	NodeID          string                      `json:"nodeId"`
	Status          string                      `json:"status"`
	SelectedModelID string                      `json:"selectedModelId"`
	Posture         *domain.SignedPostureReport `json:"posture,omitempty"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) RegisterProvider(ctx context.Context, req RegisterProviderRequest) error {
	return c.post(ctx, "/v1/providers/register", req, http.StatusCreated)
}

func (c *Client) Heartbeat(ctx context.Context, req HeartbeatRequest) error {
	return c.post(ctx, "/v1/providers/heartbeat", req, http.StatusOK)
}

func (c *Client) post(ctx context.Context, path string, payload any, expectedStatus int) error {
	if c == nil || c.baseURL == "" {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected coordinator status %d for %s", response.StatusCode, path)
	}
	return nil
}
