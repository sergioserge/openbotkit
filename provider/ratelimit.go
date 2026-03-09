package provider

import (
	"context"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(requestsPerHour int) *RateLimiter {
	r := float64(requestsPerHour) / 3600.0
	return &RateLimiter{limiter: rate.NewLimiter(rate.Limit(r), 10)}
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}
