package indexer

import (
	"context"
	"sync"
	"time"
)

// RequestLimiter serializes requests to an indexer to a maximum average rate.
// A zero or negative rate means "disabled".
type RequestLimiter struct {
	interval    time.Duration
	nextAllowed time.Time
	mu          sync.Mutex
}

func NewRequestLimiter(rps int) *RequestLimiter {
	if rps <= 0 {
		return nil
	}
	return &RequestLimiter{
		interval: time.Second / time.Duration(rps),
	}
}

func (l *RequestLimiter) Wait(ctx context.Context) error {
	if l == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	now := time.Now()

	l.mu.Lock()
	if l.nextAllowed.Before(now) {
		l.nextAllowed = now
	}
	wait := l.nextAllowed.Sub(now)
	l.nextAllowed = l.nextAllowed.Add(l.interval)
	l.mu.Unlock()

	if wait <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
