package store

import (
	"sync"

	"github.com/dginf/dginf/services/providerd/internal/domain"
)

type Memory struct {
	mu     sync.RWMutex
	config domain.NodeConfig
	status domain.NodeStatus
}

func NewMemory() *Memory {
	return &Memory{}
}

func (m *Memory) LoadConfig() domain.NodeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Memory) SaveConfig(config domain.NodeConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

func (m *Memory) LoadStatus() domain.NodeStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Memory) SaveStatus(status domain.NodeStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
}
