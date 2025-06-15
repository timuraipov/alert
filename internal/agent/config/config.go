package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr     string `env:"ADDRESS"`
	ReportInterval int64  `env:"REPORT_INTERVAL"`
	PollInterval   int64  `env:"POLL_INTERVAL"`
	FlagLogLevel   string `env:"LOG_LEVEL"`
	SignBodyKey    string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func MustLoad() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to request server")
	flag.Int64Var(&cfg.ReportInterval, "r", 10, "reportInterval period")
	flag.Int64Var(&cfg.PollInterval, "p", 2, "pollInterval period")
	// flag.StringVar(&cfg.FlagLogLevel, "l", "info", "setup flagLogLevel")
	flag.StringVar(&cfg.SignBodyKey, "k", "", "setup flagLogLevel")
	flag.IntVar(&cfg.RateLimit, "l", 0, "need to work with worker pool mode")
	flag.Parse()
	err := env.Parse(cfg)

	return cfg, err
}
