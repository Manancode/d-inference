package domain

import "time"

type ProviderStatus string

const (
	ProviderStatusHealthy ProviderStatus = "healthy"
	ProviderStatusBusy    ProviderStatus = "busy"
	ProviderStatusPaused  ProviderStatus = "paused"
	ProviderStatusOffline ProviderStatus = "offline"
)

type JobState string

const (
	JobStateQuoted               JobState = "quoted"
	JobStateSessionOpen          JobState = "session_open"
	JobStateRunning              JobState = "running"
	JobStateCompleted            JobState = "completed"
	JobStateCancelled            JobState = "cancelled"
	JobStateFailed               JobState = "failed"
	JobStateInterruptedReconcile JobState = "interrupted_pending_reconcile"
)

type RateCard struct {
	MinJobUSDC   int64 `json:"minJobUsdc"`
	Input1MUSDC  int64 `json:"input1mUsdc"`
	Output1MUSDC int64 `json:"output1mUsdc"`
}

type CatalogEntry struct {
	ModelID         string `json:"modelId"`
	MinimumMemoryGB int    `json:"minimumMemoryGb"`
	Description     string `json:"description"`
}

type ProviderRegistration struct {
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

type ProviderHeartbeat struct {
	NodeID          string           `json:"nodeId"`
	Status          ProviderStatus   `json:"status"`
	SelectedModelID string           `json:"selectedModelId"`
	Posture         *ProviderPosture `json:"posture,omitempty"`
}

type Provider struct {
	ProviderRegistration
	Status          ProviderStatus   `json:"status"`
	Allowlisted     bool             `json:"allowlisted"`
	LastHeartbeatAt time.Time        `json:"lastHeartbeatAt"`
	Posture         *ProviderPosture `json:"posture,omitempty"`
}

type WalletBalance struct {
	Wallet           string `json:"wallet"`
	AvailableUSDC    int64  `json:"availableUsdc"`
	ReservedUSDC     int64  `json:"reservedUsdc"`
	WithdrawableUSDC int64  `json:"withdrawableUsdc"`
}

type AuthChallengeRequest struct {
	Wallet  string `json:"wallet"`
	ChainID int64  `json:"chainId"`
}

type AuthChallengeResponse struct {
	Nonce     string    `json:"nonce"`
	Message   string    `json:"message"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AuthVerifyRequest struct {
	Wallet    string `json:"wallet"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

type AuthVerifyResponse struct {
	SessionToken string `json:"sessionToken"`
	Wallet       string `json:"wallet"`
}

type JobQuoteRequest struct {
	ConsumerWallet       string `json:"consumerWallet"`
	ModelID              string `json:"modelId"`
	EstimatedInputTokens int64  `json:"estimatedInputTokens"`
	MaxOutputTokens      int64  `json:"maxOutputTokens"`
}

type JobQuote struct {
	QuoteID                  string    `json:"quoteId"`
	ProviderID               string    `json:"providerId"`
	ReservationUSDC          int64     `json:"reservationUsdc"`
	MinJobUSDC               int64     `json:"minJobUsdc"`
	Input1MUSDC              int64     `json:"input1mUsdc"`
	Output1MUSDC             int64     `json:"output1mUsdc"`
	ProviderSigningPubkey    string    `json:"providerSigningPubkey"`
	ProviderSessionPubkey    string    `json:"providerSessionPubkey"`
	ProviderSessionSignature string    `json:"providerSessionSignature"`
	ExpiresAt                time.Time `json:"expiresAt"`
	ConsumerWallet           string    `json:"consumerWallet"`
	ModelID                  string    `json:"modelId"`
	Consumed                 bool      `json:"consumed"`
}

type JobCreateRequest struct {
	QuoteID               string `json:"quoteId"`
	ClientEphemeralPubkey string `json:"clientEphemeralPubkey"`
	EncryptedJobEnvelope  string `json:"encryptedJobEnvelope"`
	MaxSpendUSDC          int64  `json:"maxSpendUsdc"`
}

type SessionDescriptor struct {
	JobID                    string    `json:"jobId"`
	SessionID                string    `json:"sessionId"`
	RelayURL                 string    `json:"relayUrl"`
	ProviderNodeID           string    `json:"providerNodeId"`
	ProviderSigningPubkey    string    `json:"providerSigningPubkey"`
	ProviderSessionPubkey    string    `json:"providerSessionPubkey"`
	ProviderSessionSignature string    `json:"providerSessionSignature"`
	ExpiresAt                time.Time `json:"expiresAt"`
}

type JobRecord struct {
	JobID                string    `json:"jobId"`
	SessionID            string    `json:"sessionId"`
	State                JobState  `json:"state"`
	ProviderID           string    `json:"providerId"`
	ConsumerWallet       string    `json:"consumerWallet"`
	ModelID              string    `json:"modelId"`
	EncryptedJobEnvelope string    `json:"-"`
	ReservedUSDC         int64     `json:"reservedUsdc"`
	BilledUSDC           int64     `json:"billedUsdc"`
	SettlementNonce      uint64    `json:"settlementNonce"`
	PromptTokens         int64     `json:"promptTokens"`
	CompletionTokens     int64     `json:"completionTokens"`
	CreatedAt            time.Time `json:"createdAt"`
}

type AuthChallenge struct {
	Wallet    string
	Message   string
	Nonce     string
	ChainID   int64
	ExpiresAt time.Time
}

type JobCompletionRequest struct {
	PromptTokens     int64 `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`
}

type SeedBalanceRequest struct {
	Wallet           string `json:"wallet"`
	AvailableUSDC    int64  `json:"availableUsdc"`
	WithdrawableUSDC int64  `json:"withdrawableUsdc"`
}

type JobRunRequest struct {
	Prompt          string `json:"prompt"`
	MaxOutputTokens int    `json:"maxOutputTokens"`
}

type JobRunResult struct {
	JobID            string `json:"jobId"`
	OutputText       string `json:"outputText"`
	PromptTokens     int64  `json:"promptTokens"`
	CompletionTokens int64  `json:"completionTokens"`
	BilledUSDC       int64  `json:"billedUsdc"`
	Status           string `json:"status"`
}

type SettlementVoucher struct {
	Consumer    string `json:"consumer"`
	Provider    string `json:"provider"`
	Amount      int64  `json:"amount"`
	PlatformFee int64  `json:"platformFee"`
	Nonce       uint64 `json:"nonce"`
	JobIDHash   string `json:"jobIdHash"`
	Deadline    int64  `json:"deadline"`
}

type SettlementVoucherResponse struct {
	Voucher        SettlementVoucher `json:"voucher"`
	Signature      string            `json:"signature"`
	SignerAddress  string            `json:"signerAddress"`
	VerifyingChain uint64            `json:"verifyingChain"`
	Contract       string            `json:"contract"`
}

type ProviderPostureReport struct {
	OSVersion       string    `json:"osVersion"`
	SIPStatus       string    `json:"sipStatus"`
	FileVaultStatus string    `json:"fileVaultStatus"`
	CollectedAt     time.Time `json:"collectedAt"`
}

type ProviderPosture struct {
	Report    ProviderPostureReport `json:"report"`
	Signature string                `json:"signature"`
}
