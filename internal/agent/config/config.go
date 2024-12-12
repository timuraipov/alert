package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr     string `env:"ADDRESS"`
	ReportInterval int64  `env:"REPORT_INTERVAL"`
	PollInterval   int64  `env:"POLL_INTERVAL"`
}

func MustLoad() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to request server")
	flag.Int64Var(&cfg.ReportInterval, "r", 10, "reportInterval period")
	flag.Int64Var(&cfg.PollInterval, "p", 2, "pollInterval period")
	flag.Parse()
	err := env.Parse(cfg)

	return cfg, err
}
