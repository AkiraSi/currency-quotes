package api

import (
	"currency-quotes/infrastructure/admin/internal/updater"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

type Server struct {
	updater    *updater.Updater
	httpServer *fasthttp.Server
}

func NewServer(updater *updater.Updater) *Server {
	server := &Server{
		updater: updater,
	}

	server.httpServer = &fasthttp.Server{
		Name:            "API Server",
		ReadBufferSize:  1 << 12,
		WriteBufferSize: 1 << 12,
		Handler:         server.httpHandle(),
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     60 * time.Second,
		Logger:          nil,
	}

	return server
}

func (s *Server) ListenAndServe(addr string) error {
	return s.httpServer.ListenAndServe(addr)
}

func (s *Server) httpHandle() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx.Error("internal server error", fasthttp.StatusInternalServerError)
			}
		}()

		s.handle(ctx)
	}
}

func (s *Server) handle(ctx *fasthttp.RequestCtx) {
	method, path := string(ctx.Method()), string(ctx.Path())

	switch {
	case method == fasthttp.MethodPost && path == "/currencies/update":
		if err := s.updater.UpdateCurrencies(); err != nil {
			ctx.Error(fmt.Sprintf("update currencies: %v", err), fasthttp.StatusBadGateway)

			return
		}

		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.Error("not found", fasthttp.StatusNotFound)
	}
}
