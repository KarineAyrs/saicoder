package scheduler

import (
	"context"

	"github.com/KarineAyrs/safe-ai-coder/applications/worker/domain"
	"github.com/KarineAyrs/safe-ai-coder/pkg/scheduler"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"go.opencensus.io/trace"
)

type Client struct {
	logger   log.Logger
	provider scheduler.Provider
}

func NewClient(provider scheduler.Provider) *Client {
	return &Client{
		provider: provider,
		logger:   log.NewNopLogger(),
	}
}

func (c *Client) WithLogger(logger log.Logger) *Client {
	c.logger = logger
	return c
}

func (c *Client) UpdateTask(ctx context.Context, r domain.Result) error {
	op := "Client.UpdateTask"
	ctx, span := trace.StartSpan(ctx, op)
	defer span.End()

	level.Debug(c.logger).Log("msg", "GOT RESULT", "result", r)
	return c.provider.UpdateTask(ctx, toTask(r))
}

func toTask(r domain.Result) scheduler.Task {
	return scheduler.Task{
		Status: string(r.Status),
		Result: r.Base64,
		ID:     r.ID,
	}
}
