package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"streamnzb/pkg/core/logger"
	"sync"
	"time"
)

const saveDebounceInterval = 2 * time.Second

type StateManager struct {
	filePath  string
	data      map[string]json.RawMessage
	mu        sync.RWMutex
	saveTimer *time.Timer
	saveMu    sync.Mutex
}

var globalManager *StateManager
var managerMu sync.Mutex

func GetManager(dataDir string) (*StateManager, error) {
	managerMu.Lock()
	defer managerMu.Unlock()

	if globalManager != nil {
		return globalManager, nil
	}

	path := filepath.Join(dataDir, "state.json")
	m := &StateManager{
		filePath: path,
		data:     make(map[string]json.RawMessage),
	}

	if err := m.load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	globalManager = m
	return m, nil
}

func (m *StateManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {

			usagePath := filepath.Join(filepath.Dir(m.filePath), "usage.json")
			if _, err := os.Stat(usagePath); err == nil {
				logger.Info("Migrating usage.json to state.json")
				usageData, err := os.ReadFile(usagePath)
				if err == nil {

					m.data["indexer_usage"] = usageData
					if err := m.saveLocked(); err == nil {
						os.Remove(usagePath)
						return nil
					}
				}
			}
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &m.data)
}

func (m *StateManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveLocked()
}

func (m *StateManager) saveLocked() error {

	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0644)
}

func (m *StateManager) Get(key string, target interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	raw, ok := m.data[key]
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return true, err
	}

	return true, nil
}

func (m *StateManager) Set(key string, value interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.data[key] = raw
	m.mu.Unlock()

	m.scheduleSave()
	return nil
}

func (m *StateManager) Delete(key string) error {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	m.scheduleSave()
	return nil
}

func (m *StateManager) scheduleSave() {
	m.saveMu.Lock()
	defer m.saveMu.Unlock()

	if m.saveTimer != nil {
		m.saveTimer.Stop()
	}
	m.saveTimer = time.AfterFunc(saveDebounceInterval, func() {
		m.saveMu.Lock()
		m.saveTimer = nil
		m.saveMu.Unlock()
		if err := m.Save(); err != nil {
			logger.Error("Failed to save state", "err", err)
		}
	})
}

func (m *StateManager) Flush() error {
	m.saveMu.Lock()
	if m.saveTimer != nil {
		m.saveTimer.Stop()
		m.saveTimer = nil
	}
	m.saveMu.Unlock()
	return m.Save()
}
