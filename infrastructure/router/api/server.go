package api

import (
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"currency-quotes/common"
)

type Server struct {
	client         *fasthttp.HostClient
	requestTimeout time.Duration
}

func NewServer(workerAddress string, requestTimeout time.Duration) *Server {
	return &Server{
		client:         &fasthttp.HostClient{Addr: workerAddress},
		requestTimeout: requestTimeout,
	}
}

func (s *Server) Handle(ctx *fasthttp.RequestCtx) {
	path, method := string(ctx.Path()), string(ctx.Method())

	switch {
	case method == fasthttp.MethodGet && strings.HasPrefix(path, ratesPrefix) && len(path) > len(ratesPrefix):
		s.GetRate(ctx)
	case method == fasthttp.MethodPost && path == "/api/push":
		s.PushUpdate(ctx)
	case method == fasthttp.MethodGet && strings.HasPrefix(path, updatesPrefix) && len(path) > len(updatesPrefix):
		s.GetUpdate(ctx)
	default:
		common.WriteError(ctx, fasthttp.StatusNotFound, "not found")
	}
}
