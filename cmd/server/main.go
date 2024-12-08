package main

import (
	"net/http"

	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/server"
)

func main() {
	cfg := config.MustLoad()

	server := server.New()
	err := http.ListenAndServe(cfg.FlagRunAddr, server)
	if err != nil {
		panic(err)
	}
}
