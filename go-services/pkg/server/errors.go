package server

import (
	"errors"
	"net/http"

	codererrors "github.com/KarineAyrs/safe-ai-coder/pkg/errors"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/labstack/echo/v4"
)

// HTTPErrorHandler is an error handler.
type HTTPErrorHandler struct {
	errorCodeToHTTPStatusCodeMap map[string]int
	logger                       log.Logger
}

// NewHTTPErrorHandler creates a new instance of the HTTPErrorHandler.
func NewHTTPErrorHandler(errorCodeToStatusCodeMaps map[string]int, logger log.Logger) *HTTPErrorHandler {
	return &HTTPErrorHandler{
		errorCodeToHTTPStatusCodeMap: errorCodeToStatusCodeMaps,
		logger:                       logger,
	}
}

func (h *HTTPErrorHandler) getStatusCode(errorCode string) int {
	status, ok := h.errorCodeToHTTPStatusCodeMap[errorCode]
	if ok {
		return status
	}

	return http.StatusInternalServerError
}

// Handler handles error returned by echo Handlers.
func (h *HTTPErrorHandler) Handler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var agentErr codererrors.Error
	var statusCode int

	if !errors.As(err, &agentErr) {
		agentErr = codererrors.Error{
			Code:    "internal",
			Message: "an internal server error has occurred",
			Inner:   err,
		}
	}

	he, ok := err.(*echo.HTTPError)
	if ok {
		if he.Internal != nil {
			if herr, ok := he.Internal.(*echo.HTTPError); ok {
				he = herr
			}
		}

		m, _ := he.Message.(string)
		agentErr = codererrors.Error{
			Message: m,
			Inner:   err,
		}
		statusCode = he.Code
	} else {
		statusCode = h.getStatusCode(agentErr.Code)
	}
	// We are interested in logging only internal errors, every other error - is expected behavior.
	if agentErr.Code == "internal" {
		level.Error(h.logger).Log(
			"msg", "encountered internal error serving API request",
			"err", agentErr,
		)
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(he.Code)
		} else {
			err = c.JSON(statusCode, ErrResponse{Error: &agentErr})
		}
		if err != nil {
			level.Error(h.logger).Log(
				"msg", "error encountered when sending HTTP (echo) response",
				"err", err,
			)
		}
	}
}

// ErrResponse from server.
type ErrResponse struct {
	Error error `json:"error,omitempty"`
}
