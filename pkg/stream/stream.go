package stream

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"sync"
)

const (
	GlobalStreamID = "global"

	GlobalStreamName = "Global"
)

type Stream struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	Filters config.FilterConfig `json:"filters"`
	Sorting config.SortConfig   `json:"sorting"`

	IndexerOverrides map[string]config.IndexerSearchConfig `json:"indexer_overrides,omitempty"`

	ShowAllStream bool `json:"show_all_stream"`

	PriorityGridAdded []string `json:"priority_grid_added,omitempty"`
}

type Manager struct {
	mu      sync.RWMutex
	streams map[string]*Stream
	cfg     *config.Config
	saveFn  func() error
}

func NewManagerFromConfig(cfg *config.Config, saveFn func() error) (*Manager, error) {
	if cfg.Streams == nil {
		cfg.Streams = []*config.StreamEntry{}
	}
	m := &Manager{
		streams: make(map[string]*Stream),
		cfg:     cfg,
		saveFn:  saveFn,
	}
	if err := m.load(); err != nil {
		return nil, fmt.Errorf("load streams from config: %w", err)
	}
	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.streams = make(map[string]*Stream)
	for _, e := range m.cfg.Streams {
		if e == nil || e.ID == "" {
			continue
		}
		m.streams[e.ID] = streamFromEntry(e)
	}
	if len(m.streams) == 0 {
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

func streamFromEntry(e *config.StreamEntry) *Stream {
	s := &Stream{
		ID:                e.ID,
		Name:              e.Name,
		Filters:           e.Filters,
		Sorting:           e.Sorting,
		ShowAllStream:     e.ShowAllStream,
		IndexerOverrides:  e.IndexerOverrides,
		PriorityGridAdded: e.PriorityGridAdded,
	}
	if s.IndexerOverrides == nil {
		s.IndexerOverrides = make(map[string]config.IndexerSearchConfig)
	}
	return s
}

func entryFromStream(s *Stream) *config.StreamEntry {
	e := &config.StreamEntry{
		ID:                s.ID,
		Name:              s.Name,
		Filters:           s.Filters,
		Sorting:           s.Sorting,
		IndexerOverrides:  s.IndexerOverrides,
		ShowAllStream:     s.ShowAllStream,
		PriorityGridAdded: s.PriorityGridAdded,
	}
	if e.IndexerOverrides == nil {
		e.IndexerOverrides = make(map[string]config.IndexerSearchConfig)
	}
	return e
}

func (m *Manager) saveLocked() error {
	m.cfg.Streams = make([]*config.StreamEntry, 0, len(m.streams))
	for _, s := range m.streams {
		m.cfg.Streams = append(m.cfg.Streams, entryFromStream(s))
	}
	if m.saveFn != nil {
		return m.saveFn()
	}
	return nil
}

func (m *Manager) GetGlobal() *Stream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.streams[GlobalStreamID]; ok {
		return s
	}
	for _, s := range m.streams {
		return s
	}

	return &Stream{
		ID:      GlobalStreamID,
		Name:    GlobalStreamName,
		Filters: config.DefaultFilterConfig(),
		Sorting: config.DefaultSortConfig(),
	}
}

func (m *Manager) DefaultStreamID() string {
	return m.GetGlobal().ID
}

func (m *Manager) Get(id string) (*Stream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.streams[id]
	if !ok {
		return nil, fmt.Errorf("stream not found: %s", id)
	}
	return s, nil
}

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

func (m *Manager) List() []*Stream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Stream, 0, len(m.streams))
	for _, s := range m.streams {
		out = append(out, s)
	}
	return out
}

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
