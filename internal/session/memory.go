package session

import (
	"errors"
	"github.com/varsilias/zero-downtime/pkg/types"
	"strings"
	"sync"
	"time"
)

type Store interface {
	Append(sessionID string, m types.Message) error
	Get(sessionID string) ([]types.Message, error)
}

type MemoryStore struct {
	mu      sync.RWMutex
	data    map[string][]types.Message
	updated map[string]time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:    make(map[string][]types.Message),
		updated: make(map[string]time.Time),
	}
}

func (s *MemoryStore) Append(sessionID string, m types.Message) error {
	if sessionID == "" {
		return errors.New("empty session id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[sessionID] = append(s.data[sessionID], m)
	return nil
}

func (s *MemoryStore) Get(sessionID string) ([]types.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.data[sessionID]
	out := make([]types.Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

// List returns lightweight session summaries (best effort).
type Summary struct {
	ID      string
	Title   string
	Updated time.Time
}

func (s *MemoryStore) List() []Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Summary, 0, len(s.data))
	for id, msgs := range s.data {
		out = append(out, Summary{ID: id, Title: titleFrom(msgs), Updated: s.updated[id]})
	}
	return out
}

// Touch ensures a session exists in the list.
func (s *MemoryStore) Touch(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[sessionID]; !ok {
		s.data[sessionID] = nil
	}
	s.updated[sessionID] = time.Now()
}

func titleFrom(msgs []types.Message) string {
	for _, m := range msgs {
		if m.Role == types.RoleUser {
			return clip(words(m.Content), 8)
		}
	}
	return ""
}

func words(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	parts := strings.Fields(s)
	if len(parts) <= 12 {
		return s
	}
	return strings.Join(parts[:12], " ")
}

func clip(s string, n int) string {
	if len(s) <= n*2 {
		return s
	}
	return s[:n*2] + "…"
}
