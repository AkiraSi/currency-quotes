package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const defaultAddress = "127.0.0.1:8090"

type cbrResponse struct {
	Timestamp time.Time              `json:"Timestamp"`
	Valute    map[string]cbrCurrency `json:"Valute"`
}

type cbrCurrency struct {
	NumCode string  `json:"NumCode"`
	Nominal int64   `json:"Nominal"`
	Value   float64 `json:"Value"`
}

//nolint:mnd
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/daily_json.js", serveCurrencies)

	server := &http.Server{
		Addr:              defaultAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("mock CBR listening on http://%s/daily_json.js", defaultAddress)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

//nolint:mnd
func serveCurrencies(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	response := cbrResponse{
		Timestamp: time.Now().UTC().Truncate(time.Second),
		Valute: map[string]cbrCurrency{
			"EUR": {
				NumCode: "978",
				Nominal: 1,
				Value:   100,
			},
			"USD": {
				NumCode: "840",
				Nominal: 1,
				Value:   80,
			},
		},
	}

	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		log.Printf("encode response: %v", err)
	}
}
