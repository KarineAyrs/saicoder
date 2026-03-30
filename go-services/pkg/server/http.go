package server

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	openapimiddleware "github.com/oapi-codegen/echo-middleware"
)

// RunHTTPServerWithAddr start new payments server with provided handlers and base set of middlewares.
func RunHTTPServerWithAddr(e *echo.Echo, addr string, createHandler func(e *echo.Echo)) error {
	setMiddlewares(e)

	if createHandler != nil {
		createHandler(e)
	}

	return e.Start(addr)
}

func setMiddlewares(e *echo.Echo) {
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(context echo.Context) bool {
			skipURL := []string{
				"/metrics",
				"/health",
				"/favicon.ico",
			}
			for _, s := range skipURL {
				if context.Request().URL.Path == s {
					return true
				}
			}
			return false
		},
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
			`"status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out},"content_type":${header:Content-Type"}}` + "\n",
	}))
}

// AddRequestValidation adds request validation middleware to http server.
func AddRequestValidation(g *echo.Group, swagger *openapi3.T) {
	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil
	validatorOptions := &openapimiddleware.Options{}
	// Disable auth open api validation because we have custom middleware for validation.
	validatorOptions.Options.AuthenticationFunc = func(c context.Context, input *openapi3filter.AuthenticationInput) error {
		return nil
	}
	g.Use(openapimiddleware.OapiRequestValidatorWithOptions(swagger, validatorOptions))
}
