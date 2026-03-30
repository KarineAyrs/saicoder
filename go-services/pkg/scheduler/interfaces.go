package scheduler

import (
	"context"
)

type Provider interface {
	UpdateTask(ctx context.Context, t Task) error
}
