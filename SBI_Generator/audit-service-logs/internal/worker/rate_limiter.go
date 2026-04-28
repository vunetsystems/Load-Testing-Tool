package worker

import (
	"context"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(eps int) *RateLimiter {
	// Allow a burst of up to 10% of EPS or minimum 100
	burst := eps / 10
	if burst < 100 {
		burst = 100
	}
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(eps), burst),
	}
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}
