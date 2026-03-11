package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const saveDebounceInterval = 2 * time.Second

type StateManager struct {
	db        *sql.DB
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

	db, err := openDB(dataDir)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	if err := migrateFromStateJSON(db, dataDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate state: %w", err)
	}

	m := &StateManager{db: db}
	globalManager = m
	return m, nil
}

func (m *StateManager) Get(key string, target interface{}) (bool, error) {
	m.mu.RLock()
	value, ok, err := getKV(m.db, key)
	m.mu.RUnlock()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(value, target); err != nil {
		return true, err
	}
	return true, nil
}

func (m *StateManager) withWriteLock(fn func(*sql.DB) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(m.db)
}

func (m *StateManager) Set(key string, value interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = m.withWriteLock(func(db *sql.DB) error {
		return setKV(db, key, raw)
	})
	if err != nil {
		return err
	}
	m.scheduleSave()
	return nil
}

func (m *StateManager) Delete(key string) error {
	err := m.withWriteLock(func(db *sql.DB) error {
		return deleteKV(db, key)
	})
	if err != nil {
		return err
	}
	m.scheduleSave()
	return nil
}

func (m *StateManager) Save() error {
	// KV writes are immediate (no dirty buffer); Save is a no-op except for debounce.
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
		// No-op for SQLite; kept for API compatibility.
	})
}

func (m *StateManager) Flush() error {
	m.saveMu.Lock()
	if m.saveTimer != nil {
		m.saveTimer.Stop()
		m.saveTimer = nil
	}
	m.saveMu.Unlock()
	return nil
}
