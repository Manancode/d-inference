package runtime

import (
	"errors"
	"sync"
)

var ErrModelRequired = errors.New("model is required")

type State struct {
	LoadedModel string `json:"loaded_model"`
	Status      string `json:"status"`
}

type Service struct {
	mu          sync.RWMutex
	loadedModel string
	status      string
}

func NewService() *Service {
	return &Service{status: "idle"}
}

func (s *Service) LoadModel(model string) error {
	if model == "" {
		return ErrModelRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadedModel = model
	s.status = "ready"
	return nil
}

func (s *Service) Health() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return State{
		LoadedModel: s.loadedModel,
		Status:      s.status,
	}
}
