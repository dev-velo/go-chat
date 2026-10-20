// Package store holds chat exchanges.
package store

import (
	"context"
	"time"
)

// Exchange is one prompt and its answer. These JSON names are the API
// contract the frontend reads.
type Exchange struct {
	Prompt    string    `json:"prompt"`
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

// Store persists exchanges and reads them back.
type Store interface {
	// Save persists one exchange.
	Save(ctx context.Context, e Exchange) error
	// List returns up to limit exchanges, newest first. A limit of 0 means
	// no limit. Implementations return an empty slice, never nil.
	List(ctx context.Context, limit int) ([]Exchange, error)
}
