package store

import (
	"context"
	"sync"
	"testing"
	"time"
)

var ctx = context.Background()

func TestListReturnsNewestFirst(t *testing.T) {
	m := NewMemory()
	base := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for i, p := range []string{"first", "second", "third"} {
		_ = m.Save(ctx, Exchange{Prompt: p, Timestamp: base.Add(time.Duration(i) * time.Second)})
	}

	got := mustList(t, m, 0)
	want := []string{"third", "second", "first"}
	if len(got) != len(want) {
		t.Fatalf("got %d exchanges, want %d", len(got), len(want))
	}
	for i, p := range want {
		if got[i].Prompt != p {
			t.Errorf("position %d = %q, want %q", i, got[i].Prompt, p)
		}
	}
}

func TestListLimitKeepsTheNewest(t *testing.T) {
	m := NewMemory()
	base := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for i := range 5 {
		_ = m.Save(ctx, Exchange{Prompt: "p", Timestamp: base.Add(time.Duration(i) * time.Second)})
	}

	got := mustList(t, m, 2)
	if len(got) != 2 {
		t.Fatalf("got %d exchanges, want 2", len(got))
	}
	if !got[0].Timestamp.Equal(base.Add(4 * time.Second)) {
		t.Errorf("limit trimmed the wrong end: newest = %v", got[0].Timestamp)
	}
}

func TestListLimitLargerThanContents(t *testing.T) {
	m := NewMemory()
	_ = m.Save(ctx, Exchange{Prompt: "only"})

	if got := mustList(t, m, 10); len(got) != 1 {
		t.Errorf("got %d exchanges, want 1", len(got))
	}
}

func TestListEmpty(t *testing.T) {
	if got := mustList(t, NewMemory(), 0); len(got) != 0 {
		t.Errorf("got %d exchanges from an empty store", len(got))
	}
}

func TestListReturnsACopy(t *testing.T) {
	m := NewMemory()
	_ = m.Save(ctx, Exchange{Prompt: "original"})

	mustList(t, m, 0)[0].Prompt = "mutated"

	if got := mustList(t, m, 0); got[0].Prompt != "original" {
		t.Errorf("caller mutated the store: %q", got[0].Prompt)
	}
}

func TestConcurrentSaveAndList(t *testing.T) {
	// Run with -race to be meaningful.
	m := NewMemory()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func() { defer wg.Done(); _ = m.Save(ctx, Exchange{Prompt: "p"}) }()
		go func() { defer wg.Done(); _ = mustList(t, m, i%5) }()
	}
	wg.Wait()

	if got := mustList(t, m, 0); len(got) != 50 {
		t.Errorf("got %d exchanges, want 50", len(got))
	}
}

// mustList fails the test if List returns an error.
func mustList(t *testing.T, s Store, limit int) []Exchange {
	t.Helper()
	got, err := s.List(context.Background(), limit)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return got
}
