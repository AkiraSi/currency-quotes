package common

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type errorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(ctx *fasthttp.RequestCtx, statusCode int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		ctx.Error("internal server error", fasthttp.StatusInternalServerError)

		return
	}

	ctx.Response.Header.SetContentType("application/json")
	ctx.SetStatusCode(statusCode)
	ctx.SetBody(body)
}

func WriteError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	WriteJSON(ctx, statusCode, errorResponse{Error: message})
}
