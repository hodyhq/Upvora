// Package ratelimit is a small in-memory sliding-window limiter.
// ponytail: single-instance; move to the DB or Redis if Upvora runs multi-node.
package ratelimit

import (
	"sync"
	"time"
)

const maxKeys = 10000

// Window allows at most max events per key within per.
type Window struct {
	mu   sync.Mutex
	max  int
	per  time.Duration
	hits map[string][]time.Time
}

// New creates a limiter allowing max events per key within per.
func New(max int, per time.Duration) *Window {
	return &Window{max: max, per: per, hits: map[string][]time.Time{}}
}

// Allow records an event for key and reports whether it is within the limit.
func (w *Window) Allow(key string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-w.per)
	if len(w.hits) > maxKeys {
		w.sweep(cutoff)
	}
	kept := w.hits[key][:0]
	for _, t := range w.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= w.max {
		w.hits[key] = kept
		return false
	}
	w.hits[key] = append(kept, now)
	return true
}

// sweep drops keys with no event inside the window, so spoofed or one-off keys
// cannot grow memory without bound. If every key is still live, the oldest
// state is dropped wholesale (fail open on memory, never on correctness of
// live keys within the cap).
func (w *Window) sweep(cutoff time.Time) {
	for k, ts := range w.hits {
		if len(ts) == 0 || !ts[len(ts)-1].After(cutoff) {
			delete(w.hits, k)
		}
	}
	if len(w.hits) > maxKeys {
		w.hits = map[string][]time.Time{}
	}
}

// Keys reports how many keys are tracked (for tests).
func (w *Window) Keys() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.hits)
}
