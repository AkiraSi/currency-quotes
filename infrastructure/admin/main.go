package main

import (
	"currency-quotes/infrastructure/admin/api"
	"currency-quotes/infrastructure/admin/internal/config"
	"currency-quotes/infrastructure/admin/internal/updater"
	"log"
	"net"
)

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("admin server stopped after panic: %v", recovered)
		}
	}()

	cfg := config.Default()
	currencyUpdater := updater.NewUpdater()

	server := api.NewServer(currencyUpdater)

	if err := server.ListenAndServe(net.JoinHostPort("127.0.0.1", cfg.Port)); err != nil {
		log.Fatal(err)
	}
}
