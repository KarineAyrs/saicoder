package server

import (
	"context"
	"net/http"
	"time"
)

// Throttle provides a interceptor that limits the amount of messages processed per unit of time.
// This may be done e.g. to prevent excessive load caused by running a handler on a long queue of unprocessed messages.
type Throttle struct {
	throttle <-chan time.Time
}

// NewThrottle creates a new Throttle interceptor.
// Example duration and count: NewThrottle(10, time.Second) for 10 messages per second.
func NewThrottle(count int64, duration time.Duration) *Throttle {
	t := time.NewTicker(duration / time.Duration(count))
	return &Throttle{t.C}
}

// Intercept returns the Throttle interceptor.
func (t Throttle) Intercept() func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		if t.throttle == nil {
			return nil
		}
		<-t.throttle
		// throttle is shared by multiple handlers, which will wait for their "tick"

		return nil
	}
}
