package stream

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/core/persistence"
	"strings"
	"sync"
)

const (
	// GlobalStreamID is the fixed id for the single global stream (v1).
	GlobalStreamID = "global"
	// GlobalStreamName is the display name for the global stream.
	GlobalStreamName = "Global"
)

// Stream is a named playback configuration: filters and sorting used for search and catalog.
// For v1 there is one global stream; devices are tokens-only for auth.
type Stream struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Filters and Sorting define what releases are shown and their priority (same types as config).
	Filters config.FilterConfig `json:"filters"`
	Sorting config.SortConfig   `json:"sorting"`
	// IndexerOverrides optional per-indexer search overrides; key = indexer name. V1 can be nil.
	IndexerOverrides map[string]config.IndexerSearchConfig `json:"indexer_overrides,omitempty"`
}

// Manager loads and saves streams from state. For v1 we have one global stream.
type Manager struct {
	mu      sync.RWMutex
	streams map[string]*Stream // id -> stream
	manager *persistence.StateManager
}

// GetManager returns a stream manager using the same state as the rest of the app.
func GetManager(sm *persistence.StateManager) (*Manager, error) {
	m := &Manager{
		streams: make(map[string]*Stream),
		manager: sm,
	}
	if err := m.load(); err != nil {
		return nil, fmt.Errorf("load streams: %w", err)
	}
	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []*Stream
	found, err := m.manager.Get("streams", &list)
	if err != nil {
		return err
	}
	if found && len(list) > 0 {
		m.streams = make(map[string]*Stream, len(list))
		for _, s := range list {
			if s != nil && s.ID != "" {
				m.streams[s.ID] = s
			}
		}
	}
	if len(m.streams) == 0 {
		// Bootstrap global stream with default filters/sorting
		global := &Stream{
			ID:      GlobalStreamID,
			Name:    GlobalStreamName,
			Filters: config.DefaultFilterConfig(),
			Sorting: config.DefaultSortConfig(),
		}
		m.streams[GlobalStreamID] = global
		if err := m.saveLocked(); err != nil {
			return err
		}
		logger.Info("Bootstrapped global stream", "name", GlobalStreamName)
	}
	return nil
}

func (m *Manager) saveLocked() error {
	list := make([]*Stream, 0, len(m.streams))
	for _, s := range m.streams {
		list = append(list, s)
	}
	return m.manager.Set("streams", list)
}

// GetGlobal returns the default stream used for catalog/play when no stream is specified.
// Prefers a stream with id GlobalStreamID; otherwise returns the first stream in the list.
func (m *Manager) GetGlobal() *Stream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.streams[GlobalStreamID]; ok {
		return s
	}
	for _, s := range m.streams {
		return s
	}
	// Fallback if state was corrupted (no streams)
	return &Stream{
		ID:      GlobalStreamID,
		Name:    GlobalStreamName,
		Filters: config.DefaultFilterConfig(),
		Sorting: config.DefaultSortConfig(),
	}
}

// DefaultStreamID returns the id of the default stream (for legacy URLs that omit stream id).
func (m *Manager) DefaultStreamID() string {
	return m.GetGlobal().ID
}

// Get returns a stream by id.
func (m *Manager) Get(id string) (*Stream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.streams[id]
	if !ok {
		return nil, fmt.Errorf("stream not found: %s", id)
	}
	return s, nil
}

// SetGlobal updates the global stream (name, filters, sorting). Id remains GlobalStreamID.
func (m *Manager) SetGlobal(s *Stream) error {
	if s == nil {
		return fmt.Errorf("stream is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s.ID = GlobalStreamID
	m.streams[GlobalStreamID] = s
	return m.saveLocked()
}

// List returns all streams in no particular order.
func (m *Manager) List() []*Stream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Stream, 0, len(m.streams))
	for _, s := range m.streams {
		out = append(out, s)
	}
	return out
}

// Set creates or updates a stream by id. The stream's ID field is set to id.
func (m *Manager) Set(id string, s *Stream) error {
	if id == "" {
		return fmt.Errorf("stream id is required")
	}
	if s == nil {
		return fmt.Errorf("stream is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s.ID = id
	m.streams[id] = s
	return m.saveLocked()
}

// Create adds a new stream with a generated id. Name must be set. Returns the new id.
func (m *Manager) Create(s *Stream) (string, error) {
	if s == nil {
		return "", fmt.Errorf("stream is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := generateStreamID(m.streams)
	if err != nil {
		return "", err
	}
	s.ID = id
	if s.Name == "" {
		s.Name = "New stream"
	}
	m.streams[id] = s
	if err := m.saveLocked(); err != nil {
		return "", err
	}
	return id, nil
}

func generateStreamID(streams map[string]*Stream) (string, error) {
	for i := 0; i < 20; i++ {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		id := "stream_" + hex.EncodeToString(b[:])
		if _, exists := streams[id]; !exists {
			return id, nil
		}
	}
	return "", fmt.Errorf("could not generate unique stream id")
}

// Delete removes a stream by id. Returns error if it is the last stream (at least one required).
func (m *Manager) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("stream id is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.streams) <= 1 {
		return fmt.Errorf("cannot delete the last stream")
	}
	delete(m.streams, id)
	return m.saveLocked()
}

// NormalizeID returns id for use in URLs (no slashes). Empty or invalid returns empty.
func NormalizeID(id string) string {
	s := strings.TrimSpace(id)
	if s == "" || strings.Contains(s, "/") {
		return ""
	}
	return s
}
