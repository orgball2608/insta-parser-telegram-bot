package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Limiter interface {
	Allow(userID int64) bool
}

type InMemoryLimiter struct {
	users map[int64]*rate.Limiter
	mu    sync.Mutex
	r     rate.Limit
	b     int
}

func NewInMemoryLimiter(requests int, per time.Duration, burst int) Limiter {
	return &InMemoryLimiter{
		users: make(map[int64]*rate.Limiter),
		r:     rate.Every(per / time.Duration(requests)),
		b:     burst,
	}
}

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
