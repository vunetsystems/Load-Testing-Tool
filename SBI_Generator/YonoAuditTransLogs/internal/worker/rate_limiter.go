package worker

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiter wraps the rate limiter
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(eps int, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(eps), burst),
	}
}

// Wait waits for permission to proceed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// Allow checks if an event can proceed without waiting
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}
