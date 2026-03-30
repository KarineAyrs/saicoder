// Package handlers contains http handlers for bonuses processing.
//
//go:generate oapi-codegen -config openapi-api.config.yaml ../../../api/worker/worker.openapi.yaml
//go:generate oapi-codegen -config openapi-types.config.yaml ../../../api/worker/worker.openapi.yaml
package handlers

import (
	"github.com/KarineAyrs/safe-ai-coder/applications/worker"
	"github.com/go-kit/log"
)

type HTTPServer struct {
	svc    worker.Service
	logger log.Logger
}

func NewHTTPServer(svc worker.Service) *HTTPServer {
	return &HTTPServer{
		svc:    svc,
		logger: log.NewNopLogger(),
	}
}

func (h *HTTPServer) WithLogger(l log.Logger) *HTTPServer {
	h.logger = l

	return h
}
