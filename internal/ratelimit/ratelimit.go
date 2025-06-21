package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter defines the interface for rate limiting
type Limiter interface {
	Allow(userID int64) bool
}

// InMemoryLimiter is an implementation of Limiter stored in memory
type InMemoryLimiter struct {
	users map[int64]*rate.Limiter
	mu    sync.Mutex
	r     rate.Limit // Rate of adding tokens (e.g., 1 token every 5 seconds)
	b     int        // Bucket size (e.g., can perform 3 commands in a row)
}

// NewInMemoryLimiter creates a new rate limiter
// Example: NewInMemoryLimiter(1, 5*time.Second, 3) -> allows 1 command every 5 seconds, burst of 3 commands
func NewInMemoryLimiter(requests int, per time.Duration, burst int) Limiter {
	return &InMemoryLimiter{
		users: make(map[int64]*rate.Limiter),
		r:     rate.Every(per / time.Duration(requests)),
		b:     burst,
	}
}

// Allow checks if a user is allowed to perform an action
func (l *InMemoryLimiter) Allow(userID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.users[userID]
	if !exists {
		limiter = rate.NewLimiter(l.r, l.b)
		l.users[userID] = limiter
	}

	return limiter.Allow()
}
