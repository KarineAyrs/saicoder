package handlers

import (
	"fmt"
	"net/http"

	"github.com/KarineAyrs/safe-ai-coder/applications/worker/domain"
	pkgerrors "github.com/KarineAyrs/safe-ai-coder/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Submit (POST /submit).
func (h *HTTPServer) Submit(ctx echo.Context) error {
	var req Submit
	err := ctx.Bind(&req)
	if err != nil {
		return pkgerrors.Error{
			Code:    pkgerrors.ErrBadParameter,
			Message: "can't parse request body",
			Inner:   err,
		}
	}

	err = h.svc.Submit(ctx.Request().Context(), toDomainTask(req))
	if err != nil {
		return fmt.Errorf("can't submit task: %w", err)
	}
	return ctx.NoContent(http.StatusCreated)
}

func toDomainTask(t Submit) domain.Task {
	return domain.Task{
		ID:        t.TaskId,
		Statement: t.Statement,
	}
}
