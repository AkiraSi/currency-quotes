package common

import (
	"context"

	"github.com/valyala/fasthttp"
)

type Worker interface {
	Init(ctx context.Context) error
	Stop(ctx context.Context)
	CreateHTTPHandler() fasthttp.RequestHandler
}
