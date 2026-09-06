package main

import (
	"log"

	"github.com/valyala/fasthttp"
)

func main() {
	httpServer := fasthttp.Server{}

	err := httpServer.ListenAndServe(":8080")
	if err != nil {
		log.Fatal(err)
	}
}
