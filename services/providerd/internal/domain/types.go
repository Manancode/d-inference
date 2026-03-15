package domain

import "time"

type NodeState string

const (
	NodeStateOffline NodeState = "offline"
	NodeStateReady   NodeState = "ready"
	NodeStateBusy    NodeState = "busy"
	NodeStatePaused  NodeState = "paused"
)

type NodeConfig struct {
	NodeID          string    `json:"nodeId"`
	ProviderWallet  string    `json:"providerWallet"`
	PublicURL       string    `json:"publicUrl"`
	SelectedModel   string    `json:"selectedModel"`
	MemoryGB        int       `json:"memoryGb"`
	HardwareProfile string    `json:"hardwareProfile"`
	MinJobUSDC      int64     `json:"minJobUsdc"`
	Input1MUSDC     int64     `json:"input1mUsdc"`
	Output1MUSDC    int64     `json:"output1mUsdc"`
	HourlyRateUSDC  int64     `json:"hourlyRateUsdc"`
	LastUpdatedAt   time.Time `json:"lastUpdatedAt"`
}

type NodeStatus struct {
	NodeID                  string    `json:"nodeId"`
	State                   NodeState `json:"state"`
	SelectedModel           string    `json:"selectedModel"`
	CurrentJobID            string    `json:"currentJobId,omitempty"`
	IdentityPubkey          string    `json:"identityPubkey"`
	SessionEncryptionPubkey string    `json:"sessionEncryptionPubkey"`
	LastUpdatedAt           time.Time `json:"lastUpdatedAt"`
}

type StartJobRequest struct {
	JobID string `json:"jobId"`
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

type PostureReport struct {
	OSVersion       string    `json:"osVersion"`
	SIPStatus       string    `json:"sipStatus"`
	FileVaultStatus string    `json:"fileVaultStatus"`
	CollectedAt     time.Time `json:"collectedAt"`
}

type SignedPostureReport struct {
	Report    PostureReport `json:"report"`
	Signature string        `json:"signature"`
}
