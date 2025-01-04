package main

import (
	"log"

	"github.com/timuraipov/alert/internal/agent"
	"github.com/timuraipov/alert/internal/agent/config"
	"github.com/timuraipov/alert/internal/logger"
	"go.uber.org/zap"
)

var PollCount int64

func main() {
	op := "agent.Main"
	cfg, err := config.MustLoad()
	if err != nil {
		log.Fatal(err)
	}
	err = logger.Initialize(cfg.FlagLogLevel)
	if err != nil {
		log.Print("problem with logger", err)
	}
	logger.Log.Info("Starting to collect agent data",
		zap.String("operation", op),
	)

	agent := agent.New(cfg.ServerAddr, cfg.ReportInterval, cfg.PollInterval)
	agent.Run()
}
