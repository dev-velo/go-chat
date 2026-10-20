package store

import (
	"context"
	"sync"
)

// Memory keeps exchanges in process memory. It is safe for concurrent use,
// and is the backend for local development and tests -- never for a deployed
// instance, where anything it holds dies with the container.
type Memory struct {
	mu        sync.RWMutex
	exchanges []Exchange // oldest first
}

// Makes sure Memory implements Store and complains at compile time if it doesn't.
var _ Store = (*Memory)(nil)

// NewMemory returns an empty store.
func NewMemory() *Memory {
	return &Memory{}
}

// Save appends an exchange. It never fails.
func (m *Memory) Save(_ context.Context, e Exchange) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exchanges = append(m.exchanges, e)
	return nil
}

// List returns up to limit exchanges, newest first. The result is a copy, so
// callers cannot mutate the store.
func (m *Memory) List(_ context.Context, limit int) ([]Exchange, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	n := len(m.exchanges)
	if limit > 0 && limit < n {
		n = limit
	}

	out := make([]Exchange, 0, n)
	for i := len(m.exchanges) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, m.exchanges[i])
	}
	return out, nil
}
