package main

import (
	"log"
	"net/http"

	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/server"
)

func main() {
	cfg, err := config.MustLoad()
	if err != nil {
		log.Fatal()
	}
	server := server.New()
	err = http.ListenAndServe(cfg.FlagRunAddr, server)
	if err != nil {
		panic(err)
	}
}
