package api

import (
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"currency-quotes/common"
)

const (
	apiPrefix     = "/api"
	pushPath      = apiPrefix + "/push"
	ratesPrefix   = apiPrefix + "/rates/"
	updatesPrefix = apiPrefix + "/updates/"
)

type httpClient interface {
	DoTimeout(req *fasthttp.Request, resp *fasthttp.Response, timeout time.Duration) error
}

type Server struct {
	client         httpClient
	workerAddress  string
	requestTimeout time.Duration
}

func NewServer(workerAddress string, requestTimeout time.Duration) *Server {
	return newServer(
		&fasthttp.HostClient{Addr: workerAddress},
		workerAddress,
		requestTimeout,
	)
}

func newServer(client httpClient, workerAddress string, requestTimeout time.Duration) *Server {
	return &Server{
		client:         client,
		workerAddress:  workerAddress,
		requestTimeout: requestTimeout,
	}
}

func (s *Server) Handle(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	method := string(ctx.Method())

	switch {
	case method == fasthttp.MethodGet && strings.HasPrefix(path, ratesPrefix) && len(path) > len(ratesPrefix):
		s.GetRate(ctx)
	case method == fasthttp.MethodPost && path == pushPath:
		s.PushUpdate(ctx)
	case method == fasthttp.MethodGet && strings.HasPrefix(path, updatesPrefix) && len(path) > len(updatesPrefix):
		s.GetUpdate(ctx)
	default:
		common.WriteError(ctx, fasthttp.StatusNotFound, "not found")
	}
}
