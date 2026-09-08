package api

import (
	"strings"

	"github.com/valyala/fasthttp"

	"currency-quotes/common"
)

func (s *Server) GetRate(ctx *fasthttp.RequestCtx) {
	s.proxy(ctx, strings.TrimPrefix(string(ctx.Path()), apiPrefix))
}

func (s *Server) PushUpdate(ctx *fasthttp.RequestCtx) {
	s.proxy(ctx, "/push")
}

func (s *Server) GetUpdate(ctx *fasthttp.RequestCtx) {
	s.proxy(ctx, strings.TrimPrefix(string(ctx.Path()), apiPrefix))
}

func (s *Server) proxy(ctx *fasthttp.RequestCtx, workerPath string) {
	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	ctx.Request.CopyTo(request)
	request.URI().SetPath(workerPath)
	request.Header.SetHost(s.workerAddress)

	if err := s.client.DoTimeout(request, response, s.requestTimeout); err != nil {
		common.WriteError(ctx, fasthttp.StatusBadGateway, "rate worker unavailable")

		return
	}

	response.CopyTo(&ctx.Response)
}
