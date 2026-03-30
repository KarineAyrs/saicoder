// Package http ...
//
//go:generate oapi-codegen -config openapi-client.config.yaml ../../../../api/scheduler/scheduler.openapi.yaml
//go:generate oapi-codegen -config openapi-types.config.yaml ../../../../api/scheduler/scheduler.openapi.yaml
package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/KarineAyrs/safe-ai-coder/pkg/scheduler"
	"github.com/KarineAyrs/safe-ai-coder/pkg/scheduler/providers/http/client"
	"github.com/KarineAyrs/safe-ai-coder/pkg/server"
	"github.com/go-kit/log"
)

type ClientProvider struct {
	url          string
	client       *client.ClientWithResponses
	httpClient   *http.Client
	logger       log.Logger
	getThrottler *server.Throttle
}

func (c ClientProvider) UpdateTask(ctx context.Context, t scheduler.Task) error {
	body := client.Task{
		Result: t.Result,
		Status: t.Status,
	}
	response, err := c.client.UpdateTaskWithResponse(ctx, t.ID, body)
	if err != nil {
		return err
	}

	if isErrorResponse(response.StatusCode()) {
		return fmt.Errorf("invalid status code: %d, body: %s", response.StatusCode(), response.Body)
	}
	return nil
}

func isErrorResponse(statusCode int) bool {
	return statusCode != http.StatusOK &&
		statusCode != http.StatusCreated && statusCode != http.StatusAccepted
}
