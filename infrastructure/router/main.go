package main

import (
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"currency-quotes/common/logger"
	"currency-quotes/infrastructure/router/api"
	"currency-quotes/infrastructure/router/config"
)

func main() {
	lgr, err := logger.New()
	if err != nil {
		log.Fatalf("initialize logger: %v", err)
	}

	defer func() {
		_ = lgr.Sync()
	}()

	cfg, err := config.NewConfig()
	if err != nil {
		lgr.Error("create router config", zap.Error(err))

		return
	}

	apiServer := api.NewServer(cfg.RateWorkerAddress(), cfg.RequestTimeout)
	httpServer := &fasthttp.Server{
		Name:               "currency_quotes_router",
		Handler:            createHTTPHandler(apiServer, lgr),
		ReadBufferSize:     1 << 12, //nolint:mnd
		WriteBufferSize:    1 << 12, //nolint:mnd
		MaxRequestBodySize: 1 << 12, //nolint:mnd
		CloseOnShutdown:    true,
		ReadTimeout:        cfg.RequestTimeout,
		WriteTimeout:       cfg.RequestTimeout,
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)

	serverError := make(chan error, 1)
	go func() {
		serverError <- httpServer.ListenAndServe(cfg.RouterAddress())
	}()

	lgr.Info(
		"router started",
		zap.String("address", cfg.RouterAddress()),
		zap.String("rate_worker_address", cfg.RateWorkerAddress()),
	)

	select {
	case sig := <-signalChannel:
		lgr.Info("router shutdown requested", zap.String("signal", sig.String()))
	case err := <-serverError:
		lgr.Error("router server stopped", zap.Error(err))

		return
	}

	if err := httpServer.Shutdown(); err != nil {
		lgr.Error("shutdown router server", zap.Error(err))

		return
	}

	lgr.Info("router stopped gracefully")
}

func createHTTPHandler(server *api.Server, lgr logger.Logger) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		requestLogger := lgr.With(
			zap.String("method", string(ctx.Method())),
			zap.String("path", string(ctx.Path())),
		)

		defer func() {
			if recovered := recover(); recovered != nil {
				requestLogger.Error(
					"panic in router HTTP handler",
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				)

				ctx.Response.Reset()
				ctx.Response.Header.SetContentType("application/json")
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.SetBodyString(`{"error":"internal server error"}`)
			}

			requestLogger.Info("request finished", zap.Int("status", ctx.Response.StatusCode()))
		}()

		server.Handle(ctx)
	}
}
