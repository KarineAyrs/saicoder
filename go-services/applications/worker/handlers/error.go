package handlers

import (
	"net/http"

	"github.com/KarineAyrs/safe-ai-coder/pkg/errors"
	"github.com/KarineAyrs/safe-ai-coder/pkg/server"
	"github.com/go-kit/log"
	"github.com/labstack/echo/v4"
)

func RegisterErrorHandler(e *echo.Echo, logger log.Logger) {
	e.HTTPErrorHandler = server.NewHTTPErrorHandler(NewErrorCodeToStatusCodeMaps(), logger).Handler
}

func NewErrorCodeToStatusCodeMaps() map[string]int {
	var errorCodeToStatusCodeMaps = make(map[string]int)
	errorCodeToStatusCodeMaps[errors.ErrBadParameter] = http.StatusBadRequest
	return errorCodeToStatusCodeMaps
}
