package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/KarineAyrs/safe-ai-coder/applications/worker"
	"github.com/KarineAyrs/safe-ai-coder/applications/worker/adapters/scheduler"
	"github.com/KarineAyrs/safe-ai-coder/applications/worker/config"
	"github.com/KarineAyrs/safe-ai-coder/applications/worker/handlers"
	"github.com/KarineAyrs/safe-ai-coder/applications/worker/interfaces"
	"github.com/KarineAyrs/safe-ai-coder/applications/worker/service"
	scheduler2 "github.com/KarineAyrs/safe-ai-coder/pkg/scheduler"
	http2 "github.com/KarineAyrs/safe-ai-coder/pkg/scheduler/providers/http"
	"github.com/KarineAyrs/safe-ai-coder/pkg/server"
	"github.com/KarineAyrs/safe-ai-coder/pkg/server/prometheus"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// exitCode is a process termination code.
type exitCode int

// Possible process termination codes are listed below.
const (
	// exitSuccess is code for successful program termination.
	exitSuccess exitCode = 0
	// exitFailure is code for unsuccessful program termination.
	exitFailure exitCode = 1
)

// Shutdown timeout for servers.
const shutdownTimeout = 5 * time.Second

func main() {
	os.Exit(int(gracefulMain()))
}

func gracefulMain() exitCode {
	var logger log.Logger
	{
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configPath := fs.String("config", "/config.yml", "path to the config file")

	err := fs.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		return exitSuccess
	}
	if err != nil {
		logger.Log("msg", "parsing cli flags failed", "err", err)
		return exitFailure
	}

	logger.Log("configPath", *configPath)

	cfg, err := config.Parse(*configPath)
	if err != nil {
		logger.Log("msg", "cannot parse service config", "err", err)
		return exitFailure
	}

	var sch interfaces.Scheduler
	var schHttpProvider scheduler2.Provider
	var svc worker.Service
	{
		schLogger := log.WithPrefix(logger, "component", "scheduler-http")
		schHttpProvider, err = http2.NewProvider(
			cfg.SchedulerAPI.URL,
			http2.WithLogger(schLogger),
			http2.WithHTTPRequestTimeout(cfg.SchedulerAPI.RequestTimeout),
		)
		sch = scheduler.NewClient(schHttpProvider).WithLogger(log.WithPrefix(logger, "component", "scheduler-client"))
		svc = service.NewService(sch, cfg.Coder).WithLogger(log.WithPrefix(logger, "component", "service"))
	}

	// It's nice to be able to see panics in Logs, hence we monitor for panics after
	// logger has been bootstrapped.
	defer monitorPanic(logger)
	ctx := context.Background()

	eOps := echo.New()
	eOps.HideBanner = true
	// Expose endpoint for healthcheck.
	eOps.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	// Expose the registered Prometheus metrics via HTTP.
	eOps.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	eAPI := echo.New()
	eAPI.HideBanner = true

	eAPI.Use(echomw.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		level.Debug(logger).Log("msg", "request body dump",
			"content_type", c.Request().Header.Get("Content-Type"),
			"request_path", c.Request().URL.Path,
			"request_body", string(reqBody),
		)
	}))

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case s := <-sig:
			level.Info(logger).Log("msg", fmt.Sprintf("signal received: %v", s))
			level.Info(logger).Log("msg", "terminating...")
			return fmt.Errorf("signal received: %s", s)
		}
	})

	group.Go(func() error {
		level.Info(logger).Log("msg", "ops server is starting", "addr", cfg.Ops.HTTPAddr)
		go func() {
			<-ctx.Done()
			level.Info(logger).Log("msg", "ops server was interrupted")

			ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()

			shErr := eOps.Shutdown(ctx)
			if shErr != nil {
				level.Error(logger).Log("msg", "ops server shut down with error", "err", shErr)
			}
		}()

		err = server.RunHTTPServerWithAddr(eOps, cfg.Ops.HTTPAddr, nil)
		if err != nil {
			return fmt.Errorf("ops server start error: %w", err)
		}
		return nil
	})

	group.Go(func() error {
		level.Info(logger).Log("msg", "api server is starting", "addr", cfg.API.HTTPAddr)

		go func() {
			<-ctx.Done()
			level.Info(logger).Log("msg", "api server was interrupted")

			ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()

			if shErr := eAPI.Shutdown(ctx); shErr != nil {
				level.Error(logger).Log("msg", "api server shut down with error", "err", shErr)
			}
		}()

		nerSwagger, err := handlers.GetSwagger()
		if err != nil {
			return fmt.Errorf("ner swagger fetching failed: %w", err)
		}

		err = server.RunHTTPServerWithAddr(eAPI, cfg.API.HTTPAddr, func(e *echo.Echo) {
			p := prometheus.NewPrometheus("echo", nil, handlers.NewErrorCodeToStatusCodeMaps())
			p.Use(e)

			e.Use(echomw.RateLimiter(echomw.NewRateLimiterMemoryStore(rate.Limit(cfg.API.RateLimit))))
			e.Pre(echomw.RemoveTrailingSlash())

			nerGroup := e.Group("")
			nerServer := handlers.NewHTTPServer(svc).WithLogger(logger)
			server.AddRequestValidation(nerGroup, nerSwagger)
			handlers.RegisterHandlers(nerGroup, nerServer)

			// Errors are handled the same way for all groups.
			handlers.RegisterErrorHandler(e, logger)
		})

		if err != nil {
			return fmt.Errorf("api server start error: %w", err)
		}
		return nil
	})

	if err = group.Wait(); err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("actors stopped with err: %v", err))
		return exitFailure
	}

	level.Info(logger).Log("msg", "actors stopped without errors")

	return exitSuccess
}

// monitorPanic monitors panics and reports them somewhere (e.g. logs, ...).
func monitorPanic(logger log.Logger) {
	if rec := recover(); rec != nil {
		err := fmt.Sprintf("panic: %v \n stack trace: %s", rec, debug.Stack())
		level.Error(logger).Log("err", err)
		panic(err)
	}
}
