// Package store holds chat exchanges.
package store

import (
	"sync"
	"time"
)

// Exchange is one prompt and its answer. These JSON names are the API
// contract the frontend reads.
type Exchange struct {
	Prompt    string    `json:"prompt"`
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

// Memory keeps exchanges in process memory. It is safe for concurrent use.
type Memory struct {
	mu        sync.RWMutex
	exchanges []Exchange // oldest first
}

// NewMemory returns an empty store.
func NewMemory() *Memory {
	return &Memory{}
}

// Save appends an exchange.
func (m *Memory) Save(e Exchange) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exchanges = append(m.exchanges, e)
}

// List returns up to limit exchanges, newest first. A limit of 0 means all of
// them. The result is a copy, so callers cannot mutate the store.
func (m *Memory) List(limit int) []Exchange {
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
	return out
}
