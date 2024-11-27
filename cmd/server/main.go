package main

import (
	"net/http"

	"github.com/timuraipov/alert/internal/http-server/handlers/metrics"
	"github.com/timuraipov/alert/internal/storage/inmemory"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	saver, err := inmemory.New()
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", metrics.New(saver))
	return http.ListenAndServe(":8080", mux)
}
