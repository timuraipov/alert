package server

import (
	"net/http"

	"github.com/timuraipov/alert/internal/handlers/metrics"
	"github.com/timuraipov/alert/internal/storage/inmemory"
)

func Run() error {
	saver, err := inmemory.New()
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", metrics.New(saver))
	return http.ListenAndServe(":8080", mux)
}
