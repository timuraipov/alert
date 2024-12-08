package main

import (
	"github.com/timuraipov/alert/internal/agent"
	"github.com/timuraipov/alert/internal/agent/config"
)

var PollCount int64

func main() {
	cfg := config.MustLoad()
	agent := agent.New(cfg.ServerAddr, cfg.ReportInterval, cfg.PollInterval)
	agent.Run()
}
