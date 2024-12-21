package main

import (
	"log"
	"net/http"

	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/server"
)

func main() {
	cfg, err := config.MustLoad()
	if err != nil {
		log.Fatal()
	}
	if err := logger.Initialize(cfg.FlagLogLevel); err != nil {
		panic(err)
	}
	server := server.New(cfg)
	err = http.ListenAndServe(cfg.FlagRunAddr, server)
	if err != nil {
		panic(err)
	}
}
