package nntp

import (
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/core/persistence"
	"sync"
)

type ProviderUsageData struct {
	TotalBytes int64 `json:"total_bytes"`
}

type ProviderUsageManager struct {
	state         *persistence.StateManager
	data          map[string]*ProviderUsageData
	lastPersisted map[string]int64
	mu            sync.RWMutex
}

var providerManager *ProviderUsageManager
var providerManagerMu sync.Mutex

func GetProviderUsageManager(sm *persistence.StateManager) (*ProviderUsageManager, error) {
	providerManagerMu.Lock()
	defer providerManagerMu.Unlock()

	if providerManager != nil {
		return providerManager, nil
	}

	m := &ProviderUsageManager{
		state:         sm,
		data:          make(map[string]*ProviderUsageData),
		lastPersisted: make(map[string]int64),
	}

	if err := m.load(); err != nil {
		return nil, err
	}
	m.initLastPersisted()

	providerManager = m
	return m, nil
}

func (m *ProviderUsageManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.state.Get("provider_usage", &m.data)
	return err
}

func (m *ProviderUsageManager) initLastPersisted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, d := range m.data {
		if d != nil {
			m.lastPersisted[name] = d.TotalBytes
		}
	}
}

func (m *ProviderUsageManager) save() error {
	return m.state.Set("provider_usage", m.data)
}

func (m *ProviderUsageManager) GetUsage(name string) *ProviderUsageData {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, ok := m.data[name]
	if !ok {
		data = &ProviderUsageData{}
		m.data[name] = data
	}
	return data
}

func (m *ProviderUsageManager) AddBytes(name string, delta int64) {
	if delta <= 0 {
		return
	}
	m.mu.Lock()
	data, ok := m.data[name]
	if !ok {
		data = &ProviderUsageData{}
		m.data[name] = data
	}
	data.TotalBytes += delta
	total := data.TotalBytes
	last := m.lastPersisted[name]
	m.mu.Unlock()

	if total-last >= 1024*1024 {
		m.persistAndUpdateLast()
	}
}

func (m *ProviderUsageManager) persistAndUpdateLast() {
	if err := m.save(); err != nil {
		logger.Error("Failed to save provider usage data", "err", err)
		return
	}
	m.mu.Lock()
	for n, d := range m.data {
		if d != nil {
			m.lastPersisted[n] = d.TotalBytes
		}
	}
	m.mu.Unlock()
}

func (m *ProviderUsageManager) FlushProvider(name string) {
	m.mu.Lock()
	data := m.data[name]
	if data == nil {
		m.mu.Unlock()
		return
	}
	total := data.TotalBytes
	last := m.lastPersisted[name]
	m.mu.Unlock()

	if total > last {
		m.persistAndUpdateLast()
	}
}

func (m *ProviderUsageManager) SyncUsage(activeNames []string) {
	m.mu.Lock()

	activeMap := make(map[string]bool)
	for _, name := range activeNames {
		activeMap[name] = true
	}

	changed := false
	for name := range m.data {
		if !activeMap[name] {
			logger.Info("Removing orphaned usage data for provider", "name", name)
			delete(m.data, name)
			delete(m.lastPersisted, name)
			changed = true
		}
	}
	m.mu.Unlock()

	if changed {
		if err := m.save(); err != nil {
			logger.Error("Failed to save provider usage data after sync", "err", err)
		}
	}
}
