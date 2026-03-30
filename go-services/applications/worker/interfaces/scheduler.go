package interfaces

import (
	"context"

	"github.com/KarineAyrs/safe-ai-coder/applications/worker/domain"
)

type Scheduler interface {
	UpdateTask(ctx context.Context, r domain.Result) error
}
