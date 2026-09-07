package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"currency-quotes/common/logger"
	"currency-quotes/infrastructure/worker/common"
	workerRate "currency-quotes/infrastructure/worker/rate"
	rateConfig "currency-quotes/infrastructure/worker/rate/config"
)

func main() {
	lgr, err := logger.New()
	if err != nil {
		log.Fatalf("initialize logger: %v", err)
	}

	defer func() {
		_ = lgr.Sync()
	}()

	defer func() {
		if recovered := recover(); recovered != nil {
			lgr.Error("worker stopped after panic", zap.Any("panic", recovered))
		}
	}()

	workerType := parseWorkerTypeEnv()
	var worker common.Worker
	var port int

	switch workerType {
	case common.WorkerRate:
		cfg, err := rateConfig.NewConfig()
		if err != nil {
			lgr.Error("create rate worker config", zap.Error(err))

			return
		}

		worker = workerRate.NewWorker(cfg, lgr)
		port = cfg.Port
	default:
		lgr.Error("unknown worker type", zap.Uint8("worker_type", uint8(workerType)))

		return
	}

	ctx := context.Background()
	if err := worker.Init(ctx); err != nil {
		lgr.Error("initialize worker", zap.String("worker_type", workerType.String()), zap.Error(err))

		return
	}

	httpServer := &fasthttp.Server{
		Name:               workerType.String() + "_worker",
		Handler:            worker.CreateHTTPHandler(),
		ReadBufferSize:     1 << 12,
		WriteBufferSize:    1 << 12,
		MaxRequestBodySize: 1 << 12,
		CloseOnShutdown:    true,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		if err := httpServer.ListenAndServe(":" + strconv.Itoa(port)); err != nil {
			lgr.Error("failed to start HTTP server", zap.Error(err))
		}
	}()

	<-sigChan

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(); err != nil {
		lgr.Error("error in httpServer.Shutdown", zap.Error(err))
	}

	worker.Stop(shutdownCtx)
	lgr.Info("worker stopped gracefully")
}

func parseWorkerTypeEnv() common.WorkerType {
	value := os.Getenv("WORKER_TYPE")
	if value == "" {
		return common.WorkerRate
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return common.WorkerUnknown
	}

	return common.WorkerType(parsed)
}
