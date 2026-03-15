package app

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	coordclient "github.com/dginf/dginf/services/providerd/internal/coordinator"
	"github.com/dginf/dginf/services/providerd/internal/domain"
	"github.com/dginf/dginf/services/providerd/internal/identity"
	"github.com/dginf/dginf/services/providerd/internal/posture"
	"github.com/dginf/dginf/services/providerd/internal/runtime"
	"github.com/dginf/dginf/services/providerd/internal/store"
)

var (
	ErrNodeBusy    = errors.New("node busy")
	ErrNodePaused  = errors.New("node paused")
	ErrNoActiveJob = errors.New("no active job")
	ErrNoRuntime   = errors.New("runtime client not configured")
)

type Service struct {
	store         *store.Memory
	signer        identity.Signer
	sessionKeys   *identity.SessionKeyPair
	coordinator   *coordclient.Client
	runtimeClient *runtime.Client
	posture       *posture.Collector
	now           func() time.Time
}

func NewService(memory *store.Memory, signer identity.Signer, sessionKeys *identity.SessionKeyPair, coordinator *coordclient.Client, runtimeClient *runtime.Client, postureCollector *posture.Collector, now func() time.Time) *Service {
	if memory == nil {
		memory = store.NewMemory()
	}
	if now == nil {
		now = time.Now
	}
	return &Service{
		store:         memory,
		signer:        signer,
		sessionKeys:   sessionKeys,
		coordinator:   coordinator,
		runtimeClient: runtimeClient,
		posture:       postureCollector,
		now:           now,
	}
}

func (s *Service) Bootstrap(config domain.NodeConfig) (domain.NodeStatus, error) {
	publicKey, err := s.signer.PublicKey()
	if err != nil {
		return domain.NodeStatus{}, err
	}
	config.LastUpdatedAt = s.now().UTC()
	s.store.SaveConfig(config)
	status := domain.NodeStatus{
		NodeID:                  config.NodeID,
		State:                   domain.NodeStateReady,
		SelectedModel:           config.SelectedModel,
		IdentityPubkey:          publicKey,
		SessionEncryptionPubkey: sessionPubkey(s.sessionKeys),
		LastUpdatedAt:           config.LastUpdatedAt,
	}
	s.store.SaveStatus(status)
	return status, nil
}

func (s *Service) Status() domain.NodeStatus {
	return s.store.LoadStatus()
}

func (s *Service) Pause() domain.NodeStatus {
	status := s.store.LoadStatus()
	status.State = domain.NodeStatePaused
	status.LastUpdatedAt = s.now().UTC()
	s.store.SaveStatus(status)
	return status
}

func (s *Service) Resume() domain.NodeStatus {
	status := s.store.LoadStatus()
	status.State = domain.NodeStateReady
	status.LastUpdatedAt = s.now().UTC()
	s.store.SaveStatus(status)
	return status
}

func (s *Service) StartJob(req domain.StartJobRequest) (domain.NodeStatus, error) {
	status := s.store.LoadStatus()
	switch status.State {
	case domain.NodeStateBusy:
		return domain.NodeStatus{}, ErrNodeBusy
	case domain.NodeStatePaused:
		return domain.NodeStatus{}, ErrNodePaused
	}
	status.State = domain.NodeStateBusy
	status.CurrentJobID = req.JobID
	status.LastUpdatedAt = s.now().UTC()
	s.store.SaveStatus(status)
	return status, nil
}

func (s *Service) CompleteJob() (domain.NodeStatus, error) {
	status := s.store.LoadStatus()
	if status.CurrentJobID == "" {
		return domain.NodeStatus{}, ErrNoActiveJob
	}
	status.State = domain.NodeStateReady
	status.CurrentJobID = ""
	status.LastUpdatedAt = s.now().UTC()
	s.store.SaveStatus(status)
	return status, nil
}

func (s *Service) LoadSelectedModel(ctx context.Context) error {
	if s.runtimeClient == nil {
		return ErrNoRuntime
	}
	config := s.store.LoadConfig()
	if config.SelectedModel == "" {
		return runtime.ErrModelRequired
	}
	health, err := s.runtimeClient.Health(ctx)
	if err == nil && health.LoadedModelID == config.SelectedModel {
		return nil
	}
	_, err = s.runtimeClient.LoadModel(ctx, runtime.LoadModelRequest{
		ModelID:   config.SelectedModel,
		ModelPath: config.SelectedModel,
	})
	return err
}

func (s *Service) ExecuteJob(ctx context.Context, req domain.ExecuteJobRequest) (domain.ExecuteJobResult, error) {
	if s.runtimeClient == nil {
		return domain.ExecuteJobResult{}, ErrNoRuntime
	}
	if err := s.LoadSelectedModel(ctx); err != nil {
		return domain.ExecuteJobResult{}, err
	}
	prompt := req.Prompt
	maxOutputTokens := req.MaxOutputTokens
	if req.EncryptedEnvelope != "" && s.sessionKeys != nil && identity.LooksEncryptedEnvelope(req.EncryptedEnvelope) {
		decrypted, err := s.sessionKeys.DecryptEnvelope(req.EncryptedEnvelope)
		if err != nil {
			return domain.ExecuteJobResult{}, err
		}
		prompt = decrypted.Prompt
		maxOutputTokens = decrypted.MaxOutputTokens
	}
	if _, err := s.StartJob(domain.StartJobRequest{JobID: req.JobID}); err != nil {
		return domain.ExecuteJobResult{}, err
	}
	response, err := s.runtimeClient.Generate(ctx, runtime.GenerateRequest{
		JobID:           req.JobID,
		Prompt:          prompt,
		MaxOutputTokens: maxOutputTokens,
	})
	if err != nil {
		_, _ = s.CompleteJob()
		return domain.ExecuteJobResult{}, err
	}
	if _, err := s.CompleteJob(); err != nil {
		return domain.ExecuteJobResult{}, err
	}
	return domain.ExecuteJobResult{
		JobID:            response.JobID,
		OutputText:       response.OutputText,
		PromptTokens:     response.PromptTokens,
		CompletionTokens: response.CompletionTokens,
	}, nil
}

func (s *Service) RegisterWithCoordinator(ctx context.Context) error {
	if s.coordinator == nil {
		return nil
	}
	config := s.store.LoadConfig()
	status := s.store.LoadStatus()
	if config.NodeID == "" || config.ProviderWallet == "" {
		return errors.New("provider wallet and node id are required")
	}
	return s.coordinator.RegisterProvider(ctx, coordclient.RegisterProviderRequest{
		ProviderWallet:             config.ProviderWallet,
		NodeID:                     config.NodeID,
		SecureEnclaveSigningPubkey: status.IdentityPubkey,
		ProviderSessionPubkey:      status.SessionEncryptionPubkey,
		ProviderSessionSignature:   signSessionPubkey(s.signer, status.SessionEncryptionPubkey),
		ControlURL:                 config.PublicURL,
		HardwareProfile:            config.HardwareProfile,
		MemoryGB:                   config.MemoryGB,
		SelectedModelID:            config.SelectedModel,
		RateCard: coordclient.RateCard{
			MinJobUSDC:   config.MinJobUSDC,
			Input1MUSDC:  config.Input1MUSDC,
			Output1MUSDC: config.Output1MUSDC,
		},
	})
}

func sessionPubkey(keys *identity.SessionKeyPair) string {
	if keys == nil {
		return ""
	}
	return keys.PublicKey()
}

func signSessionPubkey(signer identity.Signer, pubkey string) string {
	if signer == nil || pubkey == "" {
		return ""
	}
	signature, err := signer.Sign([]byte(pubkey))
	if err != nil {
		return ""
	}
	return signature
}

func (s *Service) SignedPosture() (*domain.SignedPostureReport, error) {
	if s.posture == nil || s.signer == nil {
		return nil, nil
	}
	report := s.posture.Snapshot()
	raw, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	signature, err := s.signer.Sign(raw)
	if err != nil {
		return nil, err
	}
	return &domain.SignedPostureReport{
		Report:    report,
		Signature: signature,
	}, nil
}

func (s *Service) SendHeartbeat(ctx context.Context) error {
	if s.coordinator == nil {
		return nil
	}
	status := s.store.LoadStatus()
	coordinatorStatus := string(status.State)
	switch status.State {
	case domain.NodeStateReady:
		coordinatorStatus = "healthy"
	case domain.NodeStateBusy:
		coordinatorStatus = "busy"
	case domain.NodeStatePaused:
		coordinatorStatus = "paused"
	case domain.NodeStateOffline:
		coordinatorStatus = "offline"
	}
	postureReport, _ := s.SignedPosture()
	return s.coordinator.Heartbeat(ctx, coordclient.HeartbeatRequest{
		NodeID:          status.NodeID,
		Status:          coordinatorStatus,
		SelectedModelID: status.SelectedModel,
		Posture:         postureReport,
	})
}
