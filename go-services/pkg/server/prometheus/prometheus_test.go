package prometheus

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agenterrors "github.com/KarineAyrs/safe-ai-coder/pkg/errors"
	"github.com/KarineAyrs/safe-ai-coder/pkg/server"
	"github.com/appleboy/gofight/v2"
	"github.com/go-kit/log"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func unregister(p *Prometheus) {
	prometheus.Unregister(p.reqCnt)
	prometheus.Unregister(p.reqDur)
	prometheus.Unregister(p.reqSz)
	prometheus.Unregister(p.resSz)
}

func TestPrometheus_Use(t *testing.T) {
	e := echo.New()
	p := NewPrometheus("echo", nil, nil)
	p.Use(e)

	assert.Equal(t, 1, len(e.Routes()), "only one route should be added")
	assert.NotNil(t, e, "the engine should not be empty")
	assert.Equal(t, e.Routes()[0].Path, p.MetricsPath, "the path should match the metrics path")
	unregister(p)
}

func TestPath(t *testing.T) {
	p := NewPrometheus("echo", nil, nil)
	assert.Equal(t, p.MetricsPath, defaultMetricPath, "no usage of path should yield default path")
	unregister(p)
}

func TestSubsystem(t *testing.T) {
	p := NewPrometheus("echo", nil, nil)
	assert.Equal(t, p.Subsystem, "echo", "subsystem should be default")
	unregister(p)
}

func TestUse(t *testing.T) {
	e := echo.New()
	p := NewPrometheus("echo", nil, nil)

	g := gofight.New()
	g.GET(p.MetricsPath).Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNotFound, r.Code)
	})

	p.Use(e)

	g.GET(p.MetricsPath).Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})
	unregister(p)
}

func TestIgnore(t *testing.T) {
	e := echo.New()

	ipath := "/ping"
	lipath := fmt.Sprintf(`path="%s"`, ipath)
	ignore := func(c echo.Context) bool {
		return strings.HasPrefix(c.Path(), ipath)
	}
	p := NewPrometheus("echo", ignore, nil)
	p.Use(e)

	req := httptest.NewRequest(http.MethodGet, p.MetricsPath, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), fmt.Sprintf("%s_requests_total", p.Subsystem))

	req = httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	req = httptest.NewRequest(http.MethodGet, p.MetricsPath, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), lipath, "ignored path must not be present")

	unregister(p)
}

func TestMetricsGenerated(t *testing.T) {
	e := echo.New()
	p := NewPrometheus("echo", nil, nil)
	p.Use(e)

	req := httptest.NewRequest(http.MethodGet, "/ping?test=1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	req = httptest.NewRequest(http.MethodGet, p.MetricsPath, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	s := rec.Body.String()
	assert.Contains(t, s, `url="/ping"`, "path must be present")
	assert.Contains(t, s, `host="example.com"`, "host must be present")

	unregister(p)
}

func TestMetricsPathIgnored(t *testing.T) {
	e := echo.New()
	p := NewPrometheus("echo", nil, nil)
	p.Use(e)

	g := gofight.New()
	g.GET(p.MetricsPath).Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), fmt.Sprintf("%s_requests_total", p.Subsystem))
	})
	unregister(p)
}

func TestMetricsForErrors(t *testing.T) {
	e := echo.New()
	errorsMap := map[string]int{"test": http.StatusAlreadyReported}
	p := NewPrometheus("echo", nil, errorsMap)
	p.Use(e)

	e.GET("/handler_for_ok", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "OK")
	})
	e.GET("/handler_for_nok", func(c echo.Context) error {
		return c.JSON(http.StatusConflict, "NOK")
	})
	e.GET("/handler_for_error", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadGateway, "BAD")
	})
	e.GET("/handler_for_llm_fw_error", func(c echo.Context) error {
		return agenterrors.Error{
			Code:    "test",
			Message: "test",
		}
	})

	g := gofight.New()

	e.HTTPErrorHandler = server.NewHTTPErrorHandler(errorsMap, log.NewNopLogger()).Handler

	g.GET("/handler_for_ok").Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })

	g.GET("/handler_for_nok").Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusConflict, r.Code) })
	g.GET("/handler_for_nok").Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusConflict, r.Code) })

	g.GET("/handler_for_error").Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusBadGateway, r.Code) })

	g.GET("/handler_for_llm_fw_error").Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusAlreadyReported, r.Code)
	})

	g.GET(p.MetricsPath).Run(e, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		body := r.Body.String()
		assert.Contains(t, body, fmt.Sprintf("%s_requests_total", p.Subsystem))
		assert.Contains(t, body, `echo_requests_total{code="200",host="",method="GET",url="/handler_for_ok"} 1`)
		assert.Contains(t, body, `echo_requests_total{code="409",host="",method="GET",url="/handler_for_nok"} 2`)
		assert.Contains(t, body, `echo_requests_total{code="502",host="",method="GET",url="/handler_for_error"} 1`)
		assert.Contains(t, body, `echo_requests_total{code="208",host="",method="GET",url="/handler_for_llm_fw_error"} 1`)
	})
	unregister(p)
}
