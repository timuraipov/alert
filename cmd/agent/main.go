package main

import (
	"log"

	"github.com/timuraipov/alert/internal/agent"
	"github.com/timuraipov/alert/internal/agent/config"
)

var PollCount int64

func main() {
	cfg, err := config.MustLoad()
	if err != nil {
		log.Fatal(err)
	}
	agent := agent.New(cfg.ServerAddr, cfg.ReportInterval, cfg.PollInterval)
	agent.Run()
}
