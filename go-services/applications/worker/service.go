package worker

import (
	"context"

	"github.com/KarineAyrs/safe-ai-coder/applications/worker/domain"
)

type Service interface {
	Submit(ctx context.Context, t domain.Task) error
}
