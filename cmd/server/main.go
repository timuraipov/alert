package main

import (
	"log"

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

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
